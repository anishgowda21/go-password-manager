package database

import (
	"bytes"
	"errors"
	"testing"
)

func blobOf(b byte, n int) []byte { return bytes.Repeat([]byte{b}, n) }

func TestCreateItem(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	other := newTestUser(t, db)
	id := uniq("item")

	it, err := CreateItem(ctx, db, user, id, blobOf(1, 48))
	if err != nil {
		t.Fatal(err)
	}
	if it.ID != id || it.Revision != 1 || it.CreatedAt != it.UpdatedAt || !bytes.Equal(it.Blob, blobOf(1, 48)) {
		t.Fatalf("unexpected item: %+v", it)
	}

	t.Run("the same id again for the same user is ErrItemExists", func(t *testing.T) {
		if _, err := CreateItem(ctx, db, user, id, blobOf(2, 48)); !errors.Is(err, ErrItemExists) {
			t.Fatalf("err = %v, want ErrItemExists", err)
		}
	})

	t.Run("the same id for another user is fine (composite primary key)", func(t *testing.T) {
		if _, err := CreateItem(ctx, db, other, id, blobOf(3, 48)); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})
}

// UpdateItem runs the update and, on no match, a second query to tell
// "someone else's / missing item" (ErrNotFound) from "wrong revision" (ErrConflict).
func TestUpdateItem_OutcomeDependsOnOwnershipAndRevision(t *testing.T) {
	db := newTestDB(t)
	owner := newTestUser(t, db)
	stranger := newTestUser(t, db)
	id := uniq("item")

	original, err := CreateItem(ctx, db, owner, id, blobOf(1, 48))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		user     string
		id       string
		revision int
		wantErr  error
	}{
		{"another user's item looks missing", stranger, id, 1, ErrNotFound},
		{"missing item", owner, uniq("missing"), 1, ErrNotFound},
		{"stale revision", owner, id, 0, ErrConflict},
		{"future revision", owner, id, 99, ErrConflict},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UpdateItem(ctx, db, tc.user, tc.id, tc.revision, blobOf(9, 48))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}

	t.Run("failed updates leave the item untouched", func(t *testing.T) {
		items, err := ListItems(ctx, db, owner)
		if err != nil || len(items) != 1 {
			t.Fatalf("ListItems = %v, %v", items, err)
		}
		if items[0].Revision != 1 || !bytes.Equal(items[0].Blob, original.Blob) {
			t.Fatalf("item changed: %+v", items[0])
		}
	})
}

func TestUpdateItem_BumpsRevisionAndKeepsCreatedAt(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	id := uniq("item")

	original, err := CreateItem(ctx, db, user, id, blobOf(1, 48))
	if err != nil {
		t.Fatal(err)
	}

	updated, err := UpdateItem(ctx, db, user, id, 1, blobOf(2, 64))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 {
		t.Fatalf("revision = %d, want 2", updated.Revision)
	}
	if updated.CreatedAt != original.CreatedAt {
		t.Fatal("createdAt must not change")
	}
	if !bytes.Equal(updated.Blob, blobOf(2, 64)) {
		t.Fatal("the new blob must be returned")
	}

	// Chained: the next update needs revision 2.
	if _, err := UpdateItem(ctx, db, user, id, 2, blobOf(3, 48)); err != nil {
		t.Fatalf("update at revision 2: %v", err)
	}
	if _, err := UpdateItem(ctx, db, user, id, 2, blobOf(4, 48)); !errors.Is(err, ErrConflict) {
		t.Fatalf("reusing revision 2 = %v, want ErrConflict", err)
	}
}

func TestDeleteItem_IsScopedToTheOwner(t *testing.T) {
	db := newTestDB(t)
	owner := newTestUser(t, db)
	stranger := newTestUser(t, db)
	id := uniq("item")

	if _, err := CreateItem(ctx, db, owner, id, blobOf(1, 48)); err != nil {
		t.Fatal(err)
	}

	if err := DeleteItem(ctx, db, stranger, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger delete = %v, want ErrNotFound", err)
	}
	if n := count(t, db, `SELECT count(*) FROM items WHERE user_id = ? AND id = ?`, owner, id); n != 1 {
		t.Fatal("the stranger's delete must not remove the owner's item")
	}

	if err := DeleteItem(ctx, db, owner, id); err != nil {
		t.Fatalf("owner delete = %v, want nil", err)
	}
	if err := DeleteItem(ctx, db, owner, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleting twice = %v, want ErrNotFound", err)
	}
}

func TestListItems(t *testing.T) {
	db := newTestDB(t)
	user := newTestUser(t, db)
	other := newTestUser(t, db)

	t.Run("an empty vault is a non-nil empty slice", func(t *testing.T) {
		items, err := ListItems(ctx, db, user)
		if err != nil {
			t.Fatal(err)
		}
		if items == nil || len(items) != 0 {
			t.Fatalf("want a non-nil empty slice (so JSON is [], not null), got %#v", items)
		}
	})

	for _, id := range []string{"b", "a", "c"} {
		if _, err := CreateItem(ctx, db, user, id, blobOf(1, 48)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := CreateItem(ctx, db, other, "z", blobOf(1, 48)); err != nil {
		t.Fatal(err)
	}

	// Force known timestamps so the order is not left to the clock:
	// ordered by created_at, then id.
	mustExec(t, db, `UPDATE items SET created_at = 100 WHERE user_id = ? AND id IN ('a', 'b')`, user)
	mustExec(t, db, `UPDATE items SET created_at = 50 WHERE user_id = ? AND id = 'c'`, user)

	t.Run("ordered by created_at then id, and only this user's items", func(t *testing.T) {
		items, err := ListItems(ctx, db, user)
		if err != nil {
			t.Fatal(err)
		}

		var got []string
		for _, it := range items {
			got = append(got, it.ID)
		}
		want := []string{"c", "a", "b"}
		if len(got) != len(want) {
			t.Fatalf("ids = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("ids = %v, want %v", got, want)
			}
		}
	})
}
