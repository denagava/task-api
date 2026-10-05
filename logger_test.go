package main

import (
	"sync"
	"testing"
)

func TestAsyncLogger_FlushesOnClose(t *testing.T) {
	var (
		mu  sync.Mutex
		got []string
	)
	l := newAsyncLogger(10, func(s string) {
		mu.Lock()
		got = append(got, s)
		mu.Unlock()
	})
	for _, s := range []string{"a", "b", "c"} {
		l.Log(s)
	}
	l.Close()

	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("got %v, want [a b c] in order", got)
	}
}

func TestAsyncLogger_DoesNotBlockWhenFull(t *testing.T) {
	release := make(chan struct{})
	l := newAsyncLogger(1, func(string) { <-release }) // writer is stuck

	for range 10 {
		l.Log("entry") // must return immediately even though nobody reads
	}
	if l.Dropped() == 0 {
		t.Fatal("expected some entries to be dropped")
	}
	close(release)
	l.Close()
}
