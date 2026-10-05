package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestAPI(t *testing.T) http.Handler {
	t.Helper()
	logger := newAsyncLogger(100, func(string) {})
	t.Cleanup(logger.Close)
	return NewTaskHandler(NewTaskStorage(), logger).Routes()
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestTaskLifecycle(t *testing.T) {
	api := newTestAPI(t)

	rec := do(api, http.MethodPost, "/tasks", `{"title":"  Изучить Go  "}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	if loc := rec.Header().Get("Location"); loc != "/tasks/1" {
		t.Errorf("Location = %q", loc)
	}
	var created Task
	_ = json.NewDecoder(rec.Body).Decode(&created)
	if created.Title != "Изучить Go" || created.Completed {
		t.Errorf("created = %+v", created)
	}

	rec = do(api, http.MethodPatch, "/tasks/1", `{"completed":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d", rec.Code)
	}

	rec = do(api, http.MethodGet, "/tasks?completed=true", "")
	var list []Task
	_ = json.NewDecoder(rec.Body).Decode(&list)
	if len(list) != 1 || !list[0].Completed {
		t.Errorf("filtered list = %+v", list)
	}

	if rec = do(api, http.MethodDelete, "/tasks/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	if rec = do(api, http.MethodGet, "/tasks/1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("get after delete: %d", rec.Code)
	}
}

func TestEmptyListIsArray(t *testing.T) {
	rec := do(newTestAPI(t), http.MethodGet, "/tasks", "")
	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Fatalf("body = %q, want []", body)
	}
}

func TestValidationErrors(t *testing.T) {
	api := newTestAPI(t)
	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"invalid json", http.MethodPost, "/tasks", "{", http.StatusBadRequest},
		{"empty title", http.MethodPost, "/tasks", `{"title":"   "}`, http.StatusBadRequest},
		{"long title", http.MethodPost, "/tasks", `{"title":"` + strings.Repeat("a", maxTitleLen+1) + `"}`, http.StatusBadRequest},
		{"bad filter", http.MethodGet, "/tasks?completed=maybe", "", http.StatusBadRequest},
		{"bad id", http.MethodGet, "/tasks/abc", "", http.StatusBadRequest},
		{"zero id", http.MethodGet, "/tasks/0", "", http.StatusBadRequest},
		{"missing", http.MethodGet, "/tasks/99", "", http.StatusNotFound},
		{"patch without field", http.MethodPatch, "/tasks/1", `{}`, http.StatusBadRequest},
		{"patch missing", http.MethodPatch, "/tasks/99", `{"completed":true}`, http.StatusNotFound},
		{"method not allowed", http.MethodPut, "/tasks/1", "", http.StatusMethodNotAllowed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if rec := do(api, tc.method, tc.path, tc.body); rec.Code != tc.want {
				t.Errorf("status = %d, want %d (%s)", rec.Code, tc.want, rec.Body)
			}
		})
	}
}
