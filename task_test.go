package main

import (
	"errors"
	"sync"
	"testing"
)

func TestStorage_CRUD(t *testing.T) {
	s := NewTaskStorage()

	a := s.Create("first", false)
	b := s.Create("second", true)
	if a.ID != 1 || b.ID != 2 {
		t.Fatalf("ids = %d, %d", a.ID, b.ID)
	}

	got, err := s.Get(1)
	if err != nil || got.Title != "first" {
		t.Fatalf("Get(1) = %+v, %v", got, err)
	}

	if _, err := s.SetCompleted(1, true); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(1); !got.Completed {
		t.Error("task 1 should be completed")
	}

	if err := s.Delete(2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(2); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete: err = %v", err)
	}
	if err := s.Delete(2); !errors.Is(err, ErrNotFound) {
		t.Errorf("double delete: err = %v", err)
	}
}

func TestStorage_ListSortedAndFiltered(t *testing.T) {
	s := NewTaskStorage()
	for i := range 10 {
		s.Create("t", i%2 == 0)
	}

	all := s.List(nil)
	if len(all) != 10 {
		t.Fatalf("len = %d", len(all))
	}
	for i, task := range all {
		if task.ID != i+1 {
			t.Fatalf("not sorted: position %d has id %d", i, task.ID)
		}
	}

	done := true
	for _, task := range s.List(&done) {
		if !task.Completed {
			t.Errorf("filter returned incomplete task %d", task.ID)
		}
	}
}

func TestStorage_ConcurrentCreate(t *testing.T) {
	s := NewTaskStorage()
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Create("x", false)
		}()
	}
	wg.Wait()

	seen := map[int]bool{}
	for _, task := range s.List(nil) {
		if seen[task.ID] {
			t.Fatalf("duplicate id %d", task.ID)
		}
		seen[task.ID] = true
	}
	if len(seen) != 100 {
		t.Fatalf("got %d tasks, want 100", len(seen))
	}
}
