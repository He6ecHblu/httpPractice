package tasks

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type MemoryStore struct {
	mu     sync.RWMutex
	tasks  map[int]Task
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		mu:     sync.RWMutex{},
		tasks:  make(map[int]Task),
		nextID: 1,
	}
}

func (m *MemoryStore) Create(t Task) (Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t.ID = m.nextID
	m.nextID++
	t.CreatedAt = time.Now()
	m.tasks[t.ID] = t

	return t, nil
}

func (m *MemoryStore) GetByID(id int) (Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if task, ok := m.tasks[id]; ok {
		return task, nil
	}

	return Task{}, fmt.Errorf("get task %d: %w", id, ErrNotFound)
}

func (m *MemoryStore) GetAll() []Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]Task, 0, len(m.tasks))

	for _, v := range m.tasks {
		tasks = append(tasks, v)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	return tasks
}

func (m *MemoryStore) Update(id int, t Task) (Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if task, ok := m.tasks[id]; ok {
		t.CreatedAt = task.CreatedAt
		t.ID = id
		m.tasks[id] = t
		return t, nil
	}

	return Task{}, fmt.Errorf("update task %d: %w", id, ErrNotFound)
}

func (m *MemoryStore) Delete(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tasks[id]; ok {
		delete(m.tasks, id)
		return nil
	}

	return fmt.Errorf("delete task %d: %w", id, ErrNotFound)
}
