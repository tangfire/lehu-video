// biz/batch_processor.go
package biz

import (
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// BatchProcessor 通用批量处理器
type BatchProcessor[T any] struct {
	buffer    []T
	mu        sync.Mutex
	batchSize int           // 触发批量的大小
	interval  time.Duration // 触发间隔
	processFn func([]T) error
	stopCh    chan struct{}
	log       *log.Helper
}

func NewBatchProcessor[T any](
	batchSize int,
	interval time.Duration,
	processFn func([]T) error,
	logger log.Logger,
) *BatchProcessor[T] {
	p := &BatchProcessor[T]{
		buffer:    make([]T, 0, batchSize),
		batchSize: batchSize,
		interval:  interval,
		processFn: processFn,
		stopCh:    make(chan struct{}),
		log:       log.NewHelper(logger),
	}
	p.start()
	return p
}

// Add 添加单个元素到缓冲区
func (p *BatchProcessor[T]) Add(item T) {
	p.mu.Lock()
	p.buffer = append(p.buffer, item)
	shouldFlush := len(p.buffer) >= p.batchSize
	p.mu.Unlock()

	if shouldFlush {
		p.Flush()
	}
}

// Flush 立即刷新缓冲区
func (p *BatchProcessor[T]) Flush() {
	p.mu.Lock()
	if len(p.buffer) == 0 {
		p.mu.Unlock()
		return
	}
	batch := make([]T, len(p.buffer))
	copy(batch, p.buffer)
	p.buffer = p.buffer[:0]
	p.mu.Unlock()

	// 异步执行，避免阻塞调用者
	go func() {
		if err := p.processFn(batch); err != nil {
			p.log.Errorf("批量处理失败: %v", err)
			// 可选：将失败数据写入死信队列或重试
		}
	}()
}

// start 启动定时刷新
func (p *BatchProcessor[T]) start() {
	ticker := time.NewTicker(p.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				p.Flush()
			case <-p.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop 停止处理器并执行最后一次刷新
func (p *BatchProcessor[T]) Stop() {
	close(p.stopCh)
	p.Flush()
}
