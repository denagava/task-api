package main

import (
	"errors"
	"slices"
	"sync"
	"time"
)

var ErrNotFound = errors.New("task not found")

type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

// TaskStorage is a concurrency-safe in-memory task store.
type TaskStorage struct {
	mu     sync.RWMutex
	tasks  map[int]Task
	nextID int
}

func NewTaskStorage() *TaskStorage {
	return &TaskStorage{tasks: make(map[int]Task), nextID: 1}
}

func (s *TaskStorage) Create(title string, completed bool) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := Task{ID: s.nextID, Title: title, Completed: completed, CreatedAt: time.Now().UTC()}
	s.tasks[t.ID] = t
	s.nextID++
	return t
}

func (s *TaskStorage) Get(id int) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

// List returns tasks ordered by ID, optionally filtered by completion status.
func (s *TaskStorage) List(completed *bool) []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		if completed == nil || t.Completed == *completed {
			result = append(result, t)
		}
	}
	slices.SortFunc(result, func(a, b Task) int { return a.ID - b.ID })
	return result
}

// SetCompleted updates the completion status of a task.
func (s *TaskStorage) SetCompleted(id int, completed bool) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	t.Completed = completed
	s.tasks[id] = t
	return t, nil
}

func (s *TaskStorage) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
