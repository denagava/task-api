package main

import (
	"log"
	"sync"
	"sync/atomic"
)

// AsyncLogger writes log entries from a background goroutine so request
// handlers never wait on I/O. If the buffer is full, entries are dropped and
// counted instead of blocking the handler.
type AsyncLogger struct {
	ch      chan string
	wg      sync.WaitGroup
	dropped atomic.Int64
	write   func(string)
}

func NewAsyncLogger(buffer int) *AsyncLogger {
	return newAsyncLogger(buffer, func(s string) { log.Printf("[LOG] %s", s) })
}

func newAsyncLogger(buffer int, write func(string)) *AsyncLogger {
	l := &AsyncLogger{ch: make(chan string, buffer), write: write}
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		for entry := range l.ch {
			l.write(entry)
		}
	}()
	return l
}

// Log enqueues an entry without blocking.
func (l *AsyncLogger) Log(entry string) {
	select {
	case l.ch <- entry:
	default:
		l.dropped.Add(1)
	}
}

// Dropped reports how many entries were discarded because the buffer was full.
func (l *AsyncLogger) Dropped() int64 { return l.dropped.Load() }

// Close stops accepting entries and waits until the queue is fully written.
// It must be called after the HTTP server has stopped.
func (l *AsyncLogger) Close() {
	close(l.ch)
	l.wg.Wait()
}
