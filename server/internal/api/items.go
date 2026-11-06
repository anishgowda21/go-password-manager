package api

import (
	"encoding/json/v2"
	"errors"
	"log"
	"net/http"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

const (
	minBlobLen  = 29       // version byte + 12-byte nonce + 16-byte GCM tag
	maxBlobLen  = 64 << 10 // 64 KiB per item
	maxItemBody = 96 << 10 // base64 inflates by a third, plus JSON overhead
)

type itemsResponse struct {
	Items []itemResponse `json:"items"`
}

type itemResponse struct {
	ID        string `json:"id"`
	Blob      []byte `json:"blob"`
	Revision  int    `json:"revision"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type createItemRequest struct {
	ID   string `json:"id"`
	Blob []byte `json:"blob"`
}

type updateItemRequest struct {
	Blob     []byte `json:"blob"`
	Revision int    `json:"revision"`
}

func toItemResponse(it database.Item) itemResponse {
	return itemResponse{ID: it.ID, Blob: it.Blob, Revision: it.Revision, CreatedAt: it.CreatedAt, UpdatedAt: it.UpdatedAt}
}

func validateBlob(b []byte) error {
	if len(b) < minBlobLen || len(b) > maxBlobLen {
		return errors.New("invalid blob size")
	}
	return nil
}

func (s *server) syncHandler(w http.ResponseWriter, r *http.Request) {
	items, err := database.ListItems(r.Context(), s.db, userIDFrom(r.Context()))

	if err != nil {
		log.Printf("sync: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var resp itemsResponse
	resp.Items = make([]itemResponse, 0, len(items))
	for _, it := range items {
		converted := toItemResponse(it)
		resp.Items = append(resp.Items, converted)
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, resp)
}

func (s *server) createItemHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxItemBody)

	var req createItemRequest
	if err := json.UnmarshalRead(r.Body, &req, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if !uuidRe.MatchString(req.ID) {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	if err := validateBlob(req.Blob); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	it, err := database.CreateItem(r.Context(), s.db, userIDFrom(r.Context()), req.ID, req.Blob)

	switch {
	case errors.Is(err, database.ErrItemExists):
		writeError(w, http.StatusConflict, "item already exists")
		return
	case err != nil:
		log.Printf("create item: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, toItemResponse(it))
}

func (s *server) updateItemHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidRe.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxItemBody)
	var req updateItemRequest
	if err := json.UnmarshalRead(r.Body, &req, json.RejectUnknownMembers(true)); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if req.Revision < 1 {
		writeError(w, http.StatusBadRequest, "invalid revision")
		return
	}

	if err := validateBlob(req.Blob); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	it, err := database.UpdateItem(r.Context(), s.db, userIDFrom(r.Context()), id, req.Revision, req.Blob)
	switch {
	case errors.Is(err, database.ErrNotFound):
		writeError(w, http.StatusNotFound, "item not found")
		return
	case errors.Is(err, database.ErrConflict):
		writeError(w, http.StatusConflict, "revision conflict")
		return
	case err != nil:
		log.Printf("update item: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, toItemResponse(it))

}

func (s *server) deleteItemHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidRe.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	err := database.DeleteItem(r.Context(), s.db, userIDFrom(r.Context()), id)
	switch {
	case errors.Is(err, database.ErrNotFound):
		writeError(w, http.StatusNotFound, "item not found")
		return
	case err != nil:
		log.Printf("delete item: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
