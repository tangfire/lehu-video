package biz

import (
	"context"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type CampusBatchProcessor[T any] struct {
	name      string
	size      int
	interval  time.Duration
	processor func(context.Context, []T) error
	log       *log.Helper

	mu      sync.Mutex
	items   []T
	timer   *time.Timer
	stopped bool
}

func NewCampusBatchProcessor[T any](name string, size int, interval time.Duration, processor func(context.Context, []T) error, logger log.Logger) *CampusBatchProcessor[T] {
	if size <= 0 {
		size = 100
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	b := &CampusBatchProcessor[T]{
		name:      name,
		size:      size,
		interval:  interval,
		processor: processor,
		log:       log.NewHelper(logger),
		items:     make([]T, 0, size),
	}
	b.timer = time.AfterFunc(interval, b.flushOnTimer)
	b.timer.Stop()
	return b
}

func (b *CampusBatchProcessor[T]) Add(ctx context.Context, item T) error {
	if b == nil || b.processor == nil {
		return nil
	}
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		return b.processor(ctx, []T{item})
	}
	b.items = append(b.items, item)
	shouldFlush := len(b.items) >= b.size
	if len(b.items) == 1 {
		b.timer.Reset(b.interval)
	}
	if !shouldFlush {
		b.mu.Unlock()
		return nil
	}
	items := b.takeLocked()
	b.mu.Unlock()
	return b.process(ctx, items)
}

func (b *CampusBatchProcessor[T]) AddAsync(item T) {
	if b == nil || b.processor == nil {
		return
	}
	b.mu.Lock()
	if b.stopped {
		b.mu.Unlock()
		go b.processWithTimeout([]T{item})
		return
	}
	b.items = append(b.items, item)
	shouldFlush := len(b.items) >= b.size
	if len(b.items) == 1 {
		b.timer.Reset(b.interval)
	}
	if !shouldFlush {
		b.mu.Unlock()
		return
	}
	items := b.takeLocked()
	b.mu.Unlock()
	go b.processWithTimeout(items)
}

func (b *CampusBatchProcessor[T]) Flush(ctx context.Context) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	items := b.takeLocked()
	b.mu.Unlock()
	return b.process(ctx, items)
}

func (b *CampusBatchProcessor[T]) processWithTimeout(items []T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := b.process(ctx, items); err != nil {
		b.log.Warnf("async campus batch failed: name=%s count=%d err=%v", b.name, len(items), err)
	}
}

func (b *CampusBatchProcessor[T]) Stop(ctx context.Context) error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	b.stopped = true
	if b.timer != nil {
		b.timer.Stop()
	}
	items := b.takeLocked()
	b.mu.Unlock()
	return b.process(ctx, items)
}

func (b *CampusBatchProcessor[T]) flushOnTimer() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := b.Flush(ctx); err != nil {
		b.log.Warnf("flush campus batch failed: name=%s err=%v", b.name, err)
	}
}

func (b *CampusBatchProcessor[T]) takeLocked() []T {
	if b.timer != nil {
		b.timer.Stop()
	}
	if len(b.items) == 0 {
		return nil
	}
	items := b.items
	b.items = make([]T, 0, b.size)
	return items
}

func (b *CampusBatchProcessor[T]) process(ctx context.Context, items []T) error {
	if len(items) == 0 || b.processor == nil {
		return nil
	}
	if err := b.processor(ctx, items); err != nil {
		b.log.WithContext(ctx).Warnf("process campus batch failed: name=%s count=%d err=%v", b.name, len(items), err)
		return err
	}
	return nil
}
