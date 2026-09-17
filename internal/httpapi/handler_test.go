package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"todo-api/internal/tasks"
)

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	data := struct {
		Message string `json:"message"`
	}{Message: "готово"}

	err := writeJSON(recorder, http.StatusCreated, data)

	if err != nil {
		t.Fatalf("writeJSON() returned an unexpected error: %v", err)
	}
	if recorder.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json; charset=utf-8")
	}
	if got, want := recorder.Body.String(), "{\"message\":\"готово\"}\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestWriteJSONWithNilDataWritesNoBody(t *testing.T) {
	recorder := httptest.NewRecorder()

	err := writeJSON(recorder, http.StatusNoContent, nil)

	if err != nil {
		t.Fatalf("writeJSON() returned an unexpected error: %v", err)
	}
	if recorder.Code != http.StatusNoContent {
		t.Errorf("status code = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("body = %q, want an empty body", recorder.Body.String())
	}
}

func TestWriteJSONReturnsEncodingError(t *testing.T) {
	recorder := httptest.NewRecorder()

	err := writeJSON(recorder, http.StatusOK, make(chan int))

	if err == nil {
		t.Fatal("writeJSON() error = nil, want an encoding error")
	}
}

func TestWriteError(t *testing.T) {
	recorder := httptest.NewRecorder()

	err := writeError(recorder, http.StatusBadRequest, "некорректный запрос")

	if err != nil {
		t.Fatalf("writeError() returned an unexpected error: %v", err)
	}
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status code = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json; charset=utf-8")
	}
	if got, want := recorder.Body.String(), "{\"error\":\"некорректный запрос\"}\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestDecodeJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"Новая задача"}`))
	var body struct {
		Title string `json:"title"`
	}

	err := decodeJSON(request, &body)

	if err != nil {
		t.Fatalf("decodeJSON() returned an unexpected error: %v", err)
	}
	if body.Title != "Новая задача" {
		t.Errorf("decoded Title = %q, want %q", body.Title, "Новая задача")
	}
}

func TestDecodeJSONRejectsInvalidBodies(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty body", body: ""},
		{name: "malformed JSON", body: `{"title":`},
		{name: "multiple JSON values", body: `{"title":"Первая"} {"title":"Вторая"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(tt.body))
			var dst any

			if err := decodeJSON(request, &dst); err == nil {
				t.Fatal("decodeJSON() error = nil, want an error")
			}
		})
	}
}

func TestTaskHandlerHandleGetAll(t *testing.T) {
	store := tasks.NewMemoryStore()
	created, err := store.Create(tasks.Task{Title: "Первая задача"})
	if err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}
	handler := NewTaskHandler(store)
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	recorder := httptest.NewRecorder()

	handler.handleGetAll(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json; charset=utf-8")
	}
	var got []tasks.Task
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(response tasks) = %d, want 1", len(got))
	}
	if got[0].ID != created.ID || got[0].Title != created.Title || got[0].Description != created.Description || got[0].Done != created.Done {
		t.Errorf("response task = %#v, want %#v", got[0], created)
	}
	if !got[0].CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("response CreatedAt = %v, want %v", got[0].CreatedAt, created.CreatedAt)
	}
}

func TestTaskHandlerHandleGetAllReturnsEmptyArray(t *testing.T) {
	handler := NewTaskHandler(tasks.NewMemoryStore())
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	recorder := httptest.NewRecorder()

	handler.handleGetAll(recorder, request)

	if got, want := recorder.Body.String(), "[]\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestTaskHandlerImplementsHTTPHandler(t *testing.T) {
	var _ http.Handler = NewTaskHandler(tasks.NewMemoryStore())
}

func TestTaskHandlerCRUDLifecycle(t *testing.T) {
	handler := NewTaskHandler(tasks.NewMemoryStore())

	create := performRequest(handler, http.MethodPost, "/tasks", `{"id":99,"title":"Новая задача","description":"Описание","created_at":"2000-01-01T00:00:00Z"}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("POST /tasks status = %d, want %d; body: %s", create.Code, http.StatusCreated, create.Body.String())
	}
	var created tasks.Task
	decodeResponse(t, create, &created)
	if created.ID != 1 {
		t.Errorf("created ID = %d, want 1", created.ID)
	}
	if created.Title != "Новая задача" || created.Description != "Описание" {
		t.Errorf("created task = %#v, want submitted user fields", created)
	}
	if created.CreatedAt.IsZero() || created.CreatedAt.Year() == 2000 {
		t.Errorf("created CreatedAt = %v, want a server-assigned time", created.CreatedAt)
	}

	get := performRequest(handler, http.MethodGet, "/tasks/1", "")
	if get.Code != http.StatusOK {
		t.Fatalf("GET /tasks/1 status = %d, want %d; body: %s", get.Code, http.StatusOK, get.Body.String())
	}

	update := performRequest(handler, http.MethodPut, "/tasks/1", `{"id":77,"title":"Обновлённая задача","done":true,"created_at":"1999-01-01T00:00:00Z"}`)
	if update.Code != http.StatusOK {
		t.Fatalf("PUT /tasks/1 status = %d, want %d; body: %s", update.Code, http.StatusOK, update.Body.String())
	}
	var updated tasks.Task
	decodeResponse(t, update, &updated)
	if updated.ID != created.ID || updated.Title != "Обновлённая задача" || !updated.Done {
		t.Errorf("updated task = %#v, want preserved ID and updated fields", updated)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("updated CreatedAt = %v, want %v", updated.CreatedAt, created.CreatedAt)
	}

	done := performRequest(handler, http.MethodGet, "/tasks?done=true", "")
	if done.Code != http.StatusOK {
		t.Fatalf("GET /tasks?done=true status = %d, want %d", done.Code, http.StatusOK)
	}
	var doneTasks []tasks.Task
	decodeResponse(t, done, &doneTasks)
	if len(doneTasks) != 1 || !doneTasks[0].Done {
		t.Errorf("filtered tasks = %#v, want one completed task", doneTasks)
	}

	notDone := performRequest(handler, http.MethodGet, "/tasks?done=false", "")
	if got, want := notDone.Body.String(), "[]\n"; got != want {
		t.Errorf("GET /tasks?done=false body = %q, want %q", got, want)
	}

	remove := performRequest(handler, http.MethodDelete, "/tasks/1", "")
	if remove.Code != http.StatusNoContent {
		t.Fatalf("DELETE /tasks/1 status = %d, want %d", remove.Code, http.StatusNoContent)
	}
	if remove.Body.Len() != 0 {
		t.Errorf("DELETE /tasks/1 body = %q, want empty body", remove.Body.String())
	}
	repeatedRemove := performRequest(handler, http.MethodDelete, "/tasks/1", "")
	if repeatedRemove.Code != http.StatusNotFound {
		t.Errorf("repeated DELETE /tasks/1 status = %d, want %d", repeatedRemove.Code, http.StatusNotFound)
	}

	missing := performRequest(handler, http.MethodGet, "/tasks/1", "")
	if missing.Code != http.StatusNotFound {
		t.Errorf("GET deleted task status = %d, want %d", missing.Code, http.StatusNotFound)
	}
}

func TestTaskHandlerRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		body   string
		status int
	}{
		{name: "unknown path", method: http.MethodGet, target: "/unknown", status: http.StatusNotFound},
		{name: "unsupported collection method", method: http.MethodPatch, target: "/tasks", status: http.StatusMethodNotAllowed},
		{name: "unsupported item method", method: http.MethodPatch, target: "/tasks/1", status: http.StatusMethodNotAllowed},
		{name: "malformed ID", method: http.MethodGet, target: "/tasks/nope", status: http.StatusBadRequest},
		{name: "zero ID", method: http.MethodGet, target: "/tasks/0", status: http.StatusBadRequest},
		{name: "negative ID", method: http.MethodGet, target: "/tasks/-1", status: http.StatusBadRequest},
		{name: "extra path segment", method: http.MethodGet, target: "/tasks/1/extra", status: http.StatusBadRequest},
		{name: "invalid filter", method: http.MethodGet, target: "/tasks?done=yes", status: http.StatusBadRequest},
		{name: "duplicate filter", method: http.MethodGet, target: "/tasks?done=true&done=false", status: http.StatusBadRequest},
		{name: "empty create body", method: http.MethodPost, target: "/tasks", status: http.StatusBadRequest},
		{name: "invalid JSON", method: http.MethodPost, target: "/tasks", body: `{"title":`, status: http.StatusBadRequest},
		{name: "empty title", method: http.MethodPost, target: "/tasks", body: `{"title":"   "}`, status: http.StatusBadRequest},
		{name: "invalid update JSON", method: http.MethodPut, target: "/tasks/1", body: `{"title":`, status: http.StatusBadRequest},
		{name: "empty update title", method: http.MethodPut, target: "/tasks/1", body: `{"title":"   "}`, status: http.StatusBadRequest},
		{name: "update missing task", method: http.MethodPut, target: "/tasks/42", body: `{"title":"Нет задачи"}`, status: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewTaskHandler(tasks.NewMemoryStore())
			response := performRequest(handler, tt.method, tt.target, tt.body)

			if response.Code != tt.status {
				t.Errorf("status = %d, want %d; body: %s", response.Code, tt.status, response.Body.String())
			}
			if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want JSON", got)
			}
			var body map[string]string
			decodeResponse(t, response, &body)
			if body["error"] == "" {
				t.Errorf("error response = %#v, want a non-empty error message", body)
			}
		})
	}
}

type failingStore struct {
	tasks.Store
}

func (failingStore) GetByID(int) (tasks.Task, error) {
	return tasks.Task{}, errors.New("storage connection failed")
}

func TestTaskHandlerHidesInternalErrors(t *testing.T) {
	handler := NewTaskHandler(failingStore{})

	response := performRequest(handler, http.MethodGet, "/tasks/1", "")

	if response.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "storage connection failed") {
		t.Errorf("body exposes internal error: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "internal server error") {
		t.Errorf("body = %q, want a generic internal error", response.Body.String())
	}
}

func performRequest(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder, dst any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(dst); err != nil {
		t.Fatalf("decode response body: %v; body: %s", err, recorder.Body.String())
	}
}

func TestLoggingMiddleware(t *testing.T) {
	var logs bytes.Buffer
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	log.SetOutput(&logs)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "preserved")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("response body"))
	})
	handler := LoggingMiddleware(next)
	response := performRequest(handler, http.MethodPost, "/tasks?done=true", "")

	if response.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if got := response.Header().Get("X-Test"); got != "preserved" {
		t.Errorf("X-Test = %q, want %q", got, "preserved")
	}
	if got, want := response.Body.String(), "response body"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}

	logLine := logs.String()
	for _, want := range []string{"POST", "/tasks?done=true", "status=201", "duration="} {
		if !strings.Contains(logLine, want) {
			t.Errorf("log = %q, want it to contain %q", logLine, want)
		}
	}
}

func TestTaskHandlerWithGeneratedData(t *testing.T) {
	const taskCount = 100
	handler := NewTaskHandler(tasks.NewMemoryStore())
	generated := generateTestTasks(taskCount)

	for i, item := range generated {
		body, err := json.Marshal(item)
		if err != nil {
			t.Fatalf("json.Marshal(task %d): %v", i, err)
		}
		response := performRequest(handler, http.MethodPost, "/tasks", string(body))
		if response.Code != http.StatusCreated {
			t.Fatalf("create task %d status = %d, want %d; body: %s", i, response.Code, http.StatusCreated, response.Body.String())
		}

		var created tasks.Task
		decodeResponse(t, response, &created)
		if created.ID != i+1 {
			t.Fatalf("created task %d ID = %d, want %d", i, created.ID, i+1)
		}
	}

	allResponse := performRequest(handler, http.MethodGet, "/tasks", "")
	var all []tasks.Task
	decodeResponse(t, allResponse, &all)
	if len(all) != taskCount {
		t.Fatalf("GET /tasks returned %d tasks, want %d", len(all), taskCount)
	}
	for i, item := range all {
		if item.ID != i+1 {
			t.Errorf("task at index %d has ID %d, want %d", i, item.ID, i+1)
		}
		if item.Title != generated[i].Title || item.Description != generated[i].Description || item.Done != generated[i].Done {
			t.Errorf("task at index %d = %#v, want generated fields from %#v", i, item, generated[i])
		}
	}

	for _, tt := range []struct {
		name      string
		done      bool
		wantCount int
	}{
		{name: "completed", done: true, wantCount: taskCount / 2},
		{name: "not completed", done: false, wantCount: taskCount / 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := performRequest(handler, http.MethodGet, fmt.Sprintf("/tasks?done=%t", tt.done), "")
			var filtered []tasks.Task
			decodeResponse(t, response, &filtered)
			if len(filtered) != tt.wantCount {
				t.Fatalf("filtered task count = %d, want %d", len(filtered), tt.wantCount)
			}
			for _, item := range filtered {
				if item.Done != tt.done {
					t.Errorf("filtered task ID %d has done=%t, want %t", item.ID, item.Done, tt.done)
				}
			}
		})
	}
}

func generateTestTasks(count int) []tasks.Task {
	result := make([]tasks.Task, 0, count)
	for i := 0; i < count; i++ {
		description := ""
		if i%3 != 0 {
			description = fmt.Sprintf("Тестовое описание %03d", i+1)
		}
		result = append(result, tasks.Task{
			ID:          10_000 + i,
			Title:       fmt.Sprintf("Тестовая задача %03d", i+1),
			Description: description,
			Done:        i%2 == 0,
		})
	}
	return result
}
