package tasks

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestMemoryStoreCreate(t *testing.T) {
	store := NewMemoryStore()
	input := Task{
		ID:          99,
		Title:       "Изучить Go",
		Description: "Разобраться с интерфейсами",
		Done:        true,
		CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}

	beforeCreate := time.Now()
	created, err := store.Create(input)
	afterCreate := time.Now()

	if err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}
	if created.ID != 1 {
		t.Errorf("Create() ID = %d, want 1", created.ID)
	}
	if created.Title != input.Title {
		t.Errorf("Create() Title = %q, want %q", created.Title, input.Title)
	}
	if created.Description != input.Description {
		t.Errorf("Create() Description = %q, want %q", created.Description, input.Description)
	}
	if created.Done != input.Done {
		t.Errorf("Create() Done = %t, want %t", created.Done, input.Done)
	}
	if created.CreatedAt.Before(beforeCreate) || created.CreatedAt.After(afterCreate) {
		t.Errorf("Create() CreatedAt = %v, want a time between %v and %v", created.CreatedAt, beforeCreate, afterCreate)
	}

	stored, ok := store.tasks[created.ID]
	if !ok {
		t.Fatalf("Create() did not save task with ID %d", created.ID)
	}
	if !reflect.DeepEqual(stored, created) {
		t.Errorf("stored task = %#v, want %#v", stored, created)
	}
}

func TestMemoryStoreCreateAssignsSequentialIDs(t *testing.T) {
	store := NewMemoryStore()

	first, err := store.Create(Task{Title: "Первая задача"})
	if err != nil {
		t.Fatalf("first Create() returned an unexpected error: %v", err)
	}
	second, err := store.Create(Task{Title: "Вторая задача"})
	if err != nil {
		t.Fatalf("second Create() returned an unexpected error: %v", err)
	}

	if first.ID != 1 {
		t.Errorf("first Create() ID = %d, want 1", first.ID)
	}
	if second.ID != 2 {
		t.Errorf("second Create() ID = %d, want 2", second.ID)
	}
}

func TestMemoryStoreGetByID(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(Task{Title: "Найти задачу"})
	if err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}

	got, err := store.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID() returned an unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, created) {
		t.Errorf("GetByID() = %#v, want %#v", got, created)
	}
}

func TestMemoryStoreGetByIDNotFound(t *testing.T) {
	store := NewMemoryStore()

	got, err := store.GetByID(42)

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
	if got != (Task{}) {
		t.Errorf("GetByID() task = %#v, want zero Task", got)
	}
}

func TestMemoryStoreGetAllEmpty(t *testing.T) {
	store := NewMemoryStore()

	got := store.GetAll()

	if got == nil {
		t.Fatal("GetAll() returned nil, want a non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len(GetAll()) = %d, want 0", len(got))
	}
}

func TestMemoryStoreGetAllReturnsTasksSortedByID(t *testing.T) {
	store := NewMemoryStore()
	store.tasks[3] = Task{ID: 3, Title: "Третья"}
	store.tasks[1] = Task{ID: 1, Title: "Первая"}
	store.tasks[2] = Task{ID: 2, Title: "Вторая"}

	got := store.GetAll()

	want := []Task{
		{ID: 1, Title: "Первая"},
		{ID: 2, Title: "Вторая"},
		{ID: 3, Title: "Третья"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("GetAll() = %#v, want %#v", got, want)
	}
}

func TestMemoryStoreGetAllReturnsCopy(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(Task{Title: "Исходное название"})
	if err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}

	got := store.GetAll()
	got[0].Title = "Изменённое название"

	stored, err := store.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID() returned an unexpected error: %v", err)
	}
	if stored.Title != created.Title {
		t.Errorf("stored Title = %q after modifying GetAll() result, want %q", stored.Title, created.Title)
	}
}

func TestMemoryStoreConcurrentCreateAndGetAll(t *testing.T) {
	store := NewMemoryStore()
	const operations = 100

	var wg sync.WaitGroup
	wg.Add(operations * 2)
	for i := 0; i < operations; i++ {
		go func() {
			defer wg.Done()
			if _, err := store.Create(Task{Title: "Параллельная задача"}); err != nil {
				t.Errorf("Create() returned an unexpected error: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			_ = store.GetAll()
		}()
	}
	wg.Wait()

	if got := len(store.GetAll()); got != operations {
		t.Errorf("len(GetAll()) = %d, want %d", got, operations)
	}
}

func TestMemoryStoreUpdate(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(Task{Title: "Старое название", Description: "Старое описание"})
	if err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}

	input := Task{
		ID:          999,
		Title:       "Новое название",
		Description: "Новое описание",
		Done:        true,
		CreatedAt:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
	updated, err := store.Update(created.ID, input)

	if err != nil {
		t.Fatalf("Update() returned an unexpected error: %v", err)
	}
	if updated.ID != created.ID {
		t.Errorf("Update() ID = %d, want %d", updated.ID, created.ID)
	}
	if updated.CreatedAt != created.CreatedAt {
		t.Errorf("Update() CreatedAt = %v, want %v", updated.CreatedAt, created.CreatedAt)
	}
	if updated.Title != input.Title {
		t.Errorf("Update() Title = %q, want %q", updated.Title, input.Title)
	}
	if updated.Description != input.Description {
		t.Errorf("Update() Description = %q, want %q", updated.Description, input.Description)
	}
	if updated.Done != input.Done {
		t.Errorf("Update() Done = %t, want %t", updated.Done, input.Done)
	}

	stored, err := store.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID() returned an unexpected error: %v", err)
	}
	if !reflect.DeepEqual(stored, updated) {
		t.Errorf("stored task = %#v, want %#v", stored, updated)
	}
}

func TestMemoryStoreUpdateNotFound(t *testing.T) {
	store := NewMemoryStore()

	got, err := store.Update(42, Task{Title: "Несуществующая задача"})

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
	if got != (Task{}) {
		t.Errorf("Update() task = %#v, want zero Task", got)
	}
}

func TestMemoryStoreDelete(t *testing.T) {
	store := NewMemoryStore()
	created, err := store.Create(Task{Title: "Удалить задачу"})
	if err != nil {
		t.Fatalf("Create() returned an unexpected error: %v", err)
	}

	if err := store.Delete(created.ID); err != nil {
		t.Fatalf("Delete() returned an unexpected error: %v", err)
	}
	if _, err := store.GetByID(created.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryStoreDeleteNotFound(t *testing.T) {
	store := NewMemoryStore()

	err := store.Delete(42)

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}
}
