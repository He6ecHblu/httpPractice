package tasks

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTaskJSON(t *testing.T) {
	task := Task{
		ID:          7,
		Title:       "Написать тест",
		Description: "Проверить JSON-контракт",
		Done:        true,
		CreatedAt:   time.Date(2026, time.September, 17, 10, 30, 0, 0, time.UTC),
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("json.Marshal() returned an unexpected error: %v", err)
	}

	want := `{"id":7,"title":"Написать тест","description":"Проверить JSON-контракт","done":true,"created_at":"2026-09-17T10:30:00Z"}`
	if string(data) != want {
		t.Errorf("json.Marshal() = %s, want %s", data, want)
	}
}

func TestTaskJSONOmitsEmptyDescription(t *testing.T) {
	data, err := json.Marshal(Task{Title: "Задача без описания"})
	if err != nil {
		t.Fatalf("json.Marshal() returned an unexpected error: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("json.Unmarshal() returned an unexpected error: %v", err)
	}
	if _, exists := fields["description"]; exists {
		t.Errorf("json.Marshal() included an empty description: %s", data)
	}
}
