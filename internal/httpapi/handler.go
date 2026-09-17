package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"todo-api/internal/tasks"
)

type TaskHandler struct {
	store tasks.Store
}

func NewTaskHandler(store tasks.Store) *TaskHandler {
	return &TaskHandler{store: store}
}

func (h *TaskHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch {
	case r.URL.Path == "/tasks" && r.Method == http.MethodGet:
		h.handleGetAll(w, r)
	case r.URL.Path == "/tasks" && r.Method == http.MethodPost:
		h.handleCreate(w, r)
	case strings.HasPrefix(r.URL.Path, "/tasks/") && r.Method == http.MethodGet:
		h.handleGetByID(w, r)
	case strings.HasPrefix(r.URL.Path, "/tasks/") && r.Method == http.MethodPut:
		h.handleUpdate(w, r)
	case strings.HasPrefix(r.URL.Path, "/tasks/") && r.Method == http.MethodDelete:
		h.handleDelete(w, r)
	case r.URL.Path == "/tasks" || strings.HasPrefix(r.URL.Path, "/tasks/"):
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	default:
		respondError(w, http.StatusNotFound, "not found")
	}
}

func (h *TaskHandler) handleGetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	items := h.store.GetAll()

	values, hasDone := r.URL.Query()["done"]
	if hasDone {
		if len(values) != 1 || (values[0] != "true" && values[0] != "false") {
			respondError(w, http.StatusBadRequest, "done must be true or false")
			return
		}

		wantDone := values[0] == "true"
		filtered := make([]tasks.Task, 0)
		for _, item := range items {
			if item.Done == wantDone {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	respondJSON(w, http.StatusOK, items)
}

func (h *TaskHandler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r.URL.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	item, err := h.store.GetByID(id)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, item)
}

func (h *TaskHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var item tasks.Task
	if err := decodeJSON(r, &item); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(item.Title) == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	created, err := h.store.Create(item)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, created)
}

func (h *TaskHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r.URL.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	var item tasks.Task
	if err := decodeJSON(r, &item); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(item.Title) == "" {
		respondError(w, http.StatusBadRequest, "title is required")
		return
	}

	updated, err := h.store.Update(id, item)
	if err != nil {
		h.respondStoreError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, updated)
}

func (h *TaskHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseTaskID(r.URL.Path)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	if err := h.store.Delete(id); err != nil {
		h.respondStoreError(w, err)
		return
	}

	respondJSON(w, http.StatusNoContent, nil)
}

func (h *TaskHandler) respondStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, tasks.ErrNotFound) {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}

	log.Printf("store error: %v", err)
	respondError(w, http.StatusInternalServerError, "internal server error")
}

func parseTaskID(path string) (int, error) {
	const prefix = "/tasks/"
	if !strings.HasPrefix(path, prefix) {
		return 0, errors.New("task path has no ID")
	}

	rawID := strings.TrimPrefix(path, prefix)
	if rawID == "" || strings.Contains(rawID, "/") {
		return 0, errors.New("invalid task path")
	}

	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid task ID %q", rawID)
	}
	return id, nil
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	if err := writeJSON(w, status, data); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	if err := writeError(w, status, message); err != nil {
		log.Printf("failed to write error response: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) error {

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		return err
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, message string) error {
	return writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain exactly one JSON value")
	}

	return nil
}
