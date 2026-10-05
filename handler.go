package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxTitleLen = 200
	maxBodySize = 1 << 16
)

type TaskHandler struct {
	store  *TaskStorage
	logger *AsyncLogger
}

func NewTaskHandler(store *TaskStorage, logger *AsyncLogger) *TaskHandler {
	return &TaskHandler{store: store, logger: logger}
}

func (h *TaskHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", h.List)
	mux.HandleFunc("POST /tasks", h.Create)
	mux.HandleFunc("GET /tasks/{id}", h.Get)
	mux.HandleFunc("PATCH /tasks/{id}", h.Update)
	mux.HandleFunc("DELETE /tasks/{id}", h.Delete)
	return mux
}

// GET /tasks?completed=true|false
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	var filter *bool
	if param := r.URL.Query().Get("completed"); param != "" {
		v, err := strconv.ParseBool(param)
		if err != nil {
			writeError(w, http.StatusBadRequest, "completed must be true or false")
			return
		}
		filter = &v
	}
	tasks := h.store.List(filter)
	h.logger.Log(fmt.Sprintf("list tasks: %d found", len(tasks)))
	writeJSON(w, http.StatusOK, tasks)
}

// POST /tasks {"title": "...", "completed": false}
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	title := strings.TrimSpace(req.Title)
	switch {
	case title == "":
		writeError(w, http.StatusBadRequest, "title is required")
		return
	case utf8.RuneCountInString(title) > maxTitleLen:
		writeError(w, http.StatusBadRequest, fmt.Sprintf("title must be at most %d characters", maxTitleLen))
		return
	}

	t := h.store.Create(title, req.Completed)
	h.logger.Log(fmt.Sprintf("task #%d created", t.ID))
	w.Header().Set("Location", fmt.Sprintf("/tasks/%d", t.ID))
	writeJSON(w, http.StatusCreated, t)
}

// GET /tasks/{id}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := h.store.Get(id)
	if h.notFound(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// PATCH /tasks/{id} {"completed": true}
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req struct {
		Completed *bool `json:"completed"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Completed == nil {
		writeError(w, http.StatusBadRequest, "completed is required")
		return
	}
	t, err := h.store.SetCompleted(id, *req.Completed)
	if h.notFound(w, err) {
		return
	}
	h.logger.Log(fmt.Sprintf("task #%d completed=%t", id, t.Completed))
	writeJSON(w, http.StatusOK, t)
}

// DELETE /tasks/{id}
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if h.notFound(w, h.store.Delete(id)) {
		return
	}
	h.logger.Log(fmt.Sprintf("task #%d deleted", id))
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) notFound(w http.ResponseWriter, err error) bool {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return true
	}
	return false
}

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return 0, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
