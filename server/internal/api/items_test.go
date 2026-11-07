package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- helpers ----

func (e *testEnv) createItem(t *testing.T, token, id string, blob []byte) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodPost, "/api/items", token, map[string]any{"id": id, "blob": blob})
}

func (e *testEnv) updateItem(t *testing.T, token, id string, revision int, blob []byte) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodPut, "/api/items/"+id, token, map[string]any{"blob": blob, "revision": revision})
}

func (e *testEnv) deleteItem(t *testing.T, token, id string) *httptest.ResponseRecorder {
	t.Helper()
	return e.do(t, http.MethodDelete, "/api/items/"+id, token, nil)
}

// syncItems returns the user's vault keyed by item id.
func (e *testEnv) syncItems(t *testing.T, token string) map[string]itemResponse {
	t.Helper()
	rec := e.do(t, http.MethodGet, "/api/sync", token, nil)
	wantStatus(t, rec, http.StatusOK)

	out := map[string]itemResponse{}
	for _, it := range decode[itemsResponse](t, rec).Items {
		out[it.ID] = it
	}
	return out
}

// ---- sync ----

func TestSync_EmptyVaultIsAnEmptyArray(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	rec := env.do(t, http.MethodGet, "/api/sync", u.Access, nil)
	wantStatus(t, rec, http.StatusOK)

	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cc)
	}

	raw := decode[map[string]any](t, rec)
	items, ok := raw["items"].([]any)
	if !ok {
		t.Fatalf(`"items" must be an array, not null; got %#v`, raw["items"])
	}
	if len(items) != 0 {
		t.Fatalf("want 0 items, got %d", len(items))
	}
}

// ---- create ----

func TestCreateItem(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	id := newUUID(t)
	blob := randBytes(t, 48)

	rec := env.createItem(t, u.Access, id, blob)
	wantStatus(t, rec, http.StatusCreated)

	if raw := decode[map[string]any](t, rec); len(raw) != 5 {
		t.Fatalf("response must have exactly id, blob, revision, createdAt, updatedAt; got %v", raw)
	}

	it := decode[itemResponse](t, rec)
	if it.ID != id || !bytes.Equal(it.Blob, blob) {
		t.Fatal("id and blob must be echoed back")
	}
	if it.Revision != 1 {
		t.Fatalf("revision = %d, want 1", it.Revision)
	}
	if d := it.CreatedAt - time.Now().Unix(); d < -120 || d > 120 {
		t.Fatalf("createdAt %d is not close to now", it.CreatedAt)
	}
	if it.CreatedAt != it.UpdatedAt {
		t.Fatal("createdAt and updatedAt must match on create")
	}

	// The blob is stored as raw bytes, not as base64 text.
	var length, revision int
	err := env.db.QueryRow(
		`SELECT length(blob), revision FROM items WHERE user_id = ? AND id = ?`, u.UserID, id,
	).Scan(&length, &revision)
	if err != nil {
		t.Fatalf("load stored item: %v", err)
	}
	if length != 48 || revision != 1 {
		t.Fatalf("stored (length, revision) = (%d, %d), want (48, 1)", length, revision)
	}

	got, ok := env.syncItems(t, u.Access)[id]
	if !ok {
		t.Fatal("created item is missing from sync")
	}
	if !bytes.Equal(got.Blob, blob) || got.Revision != 1 {
		t.Fatalf("synced item mismatch: %+v", got)
	}
}

func TestCreateItem_DuplicateID(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)
	id := newUUID(t)

	wantStatus(t, env.createItem(t, u.Access, id, randBytes(t, 48)), http.StatusCreated)
	wantError(t, env.createItem(t, u.Access, id, randBytes(t, 48)), http.StatusConflict, "item already exists")
}

func TestCreateItem_Validation(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	valid := randBytes(t, 48)

	tests := []struct {
		name string
		body any
	}{
		{"id is not a uuid", map[string]any{"id": "hello", "blob": valid}},
		{"empty id", map[string]any{"id": "", "blob": valid}},
		{"uppercase id", map[string]any{"id": strings.ToUpper(newUUID(t)), "blob": valid}},
		{"blob of 28 bytes (too small)", map[string]any{"id": newUUID(t), "blob": randBytes(t, 28)}},
		{"blob of 65537 bytes (too large)", map[string]any{"id": newUUID(t), "blob": randBytes(t, 65537)}},
		{"blob is not base64", map[string]any{"id": newUUID(t), "blob": "***"}},
		{"empty blob", map[string]any{"id": newUUID(t), "blob": ""}},
		{"missing blob", map[string]any{"id": newUUID(t)}},
		{"unknown field", map[string]any{"id": newUUID(t), "blob": valid, "extra": 1}},
		{"malformed JSON", "{nope"},
		{"body over the size limit", `{"id":"` + newUUID(t) + `","blob":"` + strings.Repeat("A", 200000) + `"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantStatus(t, env.do(t, http.MethodPost, "/api/items", u.Access, tc.body), http.StatusBadRequest)
		})
	}
}

func TestCreateItem_BlobSizeBounds(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	for _, size := range []int{29, 65536} {
		t.Run(fmt.Sprintf("%d bytes", size), func(t *testing.T) {
			wantStatus(t, env.createItem(t, u.Access, newUUID(t), randBytes(t, size)), http.StatusCreated)
		})
	}
}

// ---- update ----

func TestUpdateItem(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	id := newUUID(t)
	created := decode[itemResponse](t, env.createItem(t, u.Access, id, randBytes(t, 48)))

	newBlob := randBytes(t, 64)
	rec := env.updateItem(t, u.Access, id, 1, newBlob)
	wantStatus(t, rec, http.StatusOK)

	up := decode[itemResponse](t, rec)
	if up.Revision != 2 {
		t.Fatalf("revision = %d, want 2", up.Revision)
	}
	if up.CreatedAt != created.CreatedAt {
		t.Fatal("createdAt must not change on update")
	}
	if up.UpdatedAt < created.UpdatedAt {
		t.Fatal("updatedAt must not go backwards")
	}
	if !bytes.Equal(up.Blob, newBlob) {
		t.Fatal("the new blob must be echoed back")
	}

	got := env.syncItems(t, u.Access)[id]
	if got.Revision != 2 || !bytes.Equal(got.Blob, newBlob) {
		t.Fatalf("sync must show the update at revision 2, got %+v", got)
	}
}

func TestUpdateItem_RevisionConflicts(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	id := newUUID(t)
	env.createItem(t, u.Access, id, randBytes(t, 48))
	wantStatus(t, env.updateItem(t, u.Access, id, 1, randBytes(t, 48)), http.StatusOK) // now at revision 2

	t.Run("stale revision", func(t *testing.T) {
		wantError(t, env.updateItem(t, u.Access, id, 1, randBytes(t, 48)), http.StatusConflict, "revision conflict")
	})
	t.Run("future revision", func(t *testing.T) {
		wantError(t, env.updateItem(t, u.Access, id, 99, randBytes(t, 48)), http.StatusConflict, "revision conflict")
	})
	t.Run("a rejected update changes nothing", func(t *testing.T) {
		if got := env.syncItems(t, u.Access)[id]; got.Revision != 2 {
			t.Fatalf("revision = %d, want 2", got.Revision)
		}
	})
}

func TestUpdateItem_Errors(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	id := newUUID(t)
	env.createItem(t, u.Access, id, randBytes(t, 48))

	t.Run("missing item", func(t *testing.T) {
		wantError(t, env.updateItem(t, u.Access, newUUID(t), 1, randBytes(t, 48)), http.StatusNotFound, "item not found")
	})
	t.Run("id is not a uuid", func(t *testing.T) {
		wantError(t, env.updateItem(t, u.Access, "hello", 1, randBytes(t, 48)), http.StatusBadRequest, "invalid item id")
	})
	t.Run("revision 0", func(t *testing.T) {
		wantError(t, env.updateItem(t, u.Access, id, 0, randBytes(t, 48)), http.StatusBadRequest, "invalid revision")
	})
	t.Run("revision -1", func(t *testing.T) {
		wantError(t, env.updateItem(t, u.Access, id, -1, randBytes(t, 48)), http.StatusBadRequest, "invalid revision")
	})
	t.Run("blob too short", func(t *testing.T) {
		wantStatus(t, env.updateItem(t, u.Access, id, 1, randBytes(t, 10)), http.StatusBadRequest)
	})
	t.Run("unknown field", func(t *testing.T) {
		rec := env.do(t, http.MethodPut, "/api/items/"+id, u.Access,
			map[string]any{"blob": randBytes(t, 48), "revision": 1, "extra": 1})
		wantStatus(t, rec, http.StatusBadRequest)
	})
	t.Run("malformed JSON", func(t *testing.T) {
		wantStatus(t, env.do(t, http.MethodPut, "/api/items/"+id, u.Access, "{nope"), http.StatusBadRequest)
	})
}

// Many clients update the same revision at once: exactly one may win.
func TestUpdateItem_ConcurrentSameRevisionOnlyOneWins(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	id := newUUID(t)
	wantStatus(t, env.createItem(t, u.Access, id, randBytes(t, 48)), http.StatusCreated)

	const n = 8
	bodies := make([]string, n)
	for i := range bodies {
		bodies[i] = fmt.Sprintf(`{"blob":%q,"revision":1}`, randB64(t, 48))
	}

	codes := make([]int, n)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			req := httptest.NewRequest(http.MethodPut, "/api/items/"+id, strings.NewReader(bodies[i]))
			req.Header.Set("Authorization", "Bearer "+u.Access)
			rec := httptest.NewRecorder()
			env.handler.ServeHTTP(rec, req)
			codes[i] = rec.Code
		}(i)
	}
	close(start)
	wg.Wait()

	var ok, conflict int
	for _, c := range codes {
		switch c {
		case http.StatusOK:
			ok++
		case http.StatusConflict:
			conflict++
		}
	}
	if ok != 1 || conflict != n-1 {
		t.Fatalf("want exactly 1 success and %d conflicts, got codes %v", n-1, codes)
	}

	if got := env.syncItems(t, u.Access)[id]; got.Revision != 2 {
		t.Fatalf("final revision = %d, want 2", got.Revision)
	}
}

// ---- delete ----

func TestDeleteItem(t *testing.T) {
	env := newTestEnv(t)
	u := env.registerUser(t)

	id, keepA, keepB := newUUID(t), newUUID(t), newUUID(t)
	env.createItem(t, u.Access, id, randBytes(t, 48))
	env.createItem(t, u.Access, keepA, randBytes(t, 29))
	env.createItem(t, u.Access, keepB, randBytes(t, 48))

	rec := env.deleteItem(t, u.Access, id)
	wantStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Fatalf("delete must return an empty body, got %q", rec.Body.String())
	}

	t.Run("deleting again is a 404", func(t *testing.T) {
		wantError(t, env.deleteItem(t, u.Access, id), http.StatusNotFound, "item not found")
	})
	t.Run("updating a deleted item is a 404", func(t *testing.T) {
		wantStatus(t, env.updateItem(t, u.Access, id, 1, randBytes(t, 48)), http.StatusNotFound)
	})
	t.Run("only the deleted item is gone", func(t *testing.T) {
		got := env.syncItems(t, u.Access)
		if _, exists := got[id]; exists {
			t.Fatal("deleted item must not appear in sync")
		}
		if _, exists := got[keepA]; !exists {
			t.Fatal("other items must remain")
		}
		if _, exists := got[keepB]; !exists {
			t.Fatal("other items must remain")
		}
	})
	t.Run("id is not a uuid", func(t *testing.T) {
		wantError(t, env.deleteItem(t, u.Access, "hello"), http.StatusBadRequest, "invalid item id")
	})
}

// ---- isolation between users ----

func TestItems_IsolationBetweenUsers(t *testing.T) {
	env := newTestEnv(t)
	a := env.registerUser(t)
	b := env.registerUser(t)

	id := newUUID(t)
	aBlob := randBytes(t, 48)
	bBlob := randBytes(t, 48)
	wantStatus(t, env.createItem(t, a.Access, id, aBlob), http.StatusCreated)

	t.Run("B does not see A's item", func(t *testing.T) {
		if _, exists := env.syncItems(t, b.Access)[id]; exists {
			t.Fatal("B must not see A's item")
		}
	})
	t.Run("B cannot update A's item", func(t *testing.T) {
		wantStatus(t, env.updateItem(t, b.Access, id, 1, bBlob), http.StatusNotFound)
	})
	t.Run("B cannot delete A's item", func(t *testing.T) {
		wantStatus(t, env.deleteItem(t, b.Access, id), http.StatusNotFound)
	})
	t.Run("B can create the same id for themselves", func(t *testing.T) {
		wantStatus(t, env.createItem(t, b.Access, id, bBlob), http.StatusCreated)
	})
	t.Run("A's item is untouched by B", func(t *testing.T) {
		got := env.syncItems(t, a.Access)[id]
		if !bytes.Equal(got.Blob, aBlob) || got.Revision != 1 {
			t.Fatalf("A's item changed: %+v", got)
		}
		if gotB := env.syncItems(t, b.Access)[id]; !bytes.Equal(gotB.Blob, bBlob) {
			t.Fatal("B must have their own copy")
		}
	})
	t.Run("B deleting their copy leaves A's alone", func(t *testing.T) {
		wantStatus(t, env.deleteItem(t, b.Access, id), http.StatusNoContent)
		if _, exists := env.syncItems(t, a.Access)[id]; !exists {
			t.Fatal("A's item must survive B's delete")
		}
	})
}
