package service

import (
	"context"
	"fmt"
	"sync"
)

type WorkerPool struct {
	tasks   chan func(ctx context.Context) error // 处理的任务方法
	wg      sync.WaitGroup                       // 处理子协程
	ctx     context.Context                      // 全文的上下文
	cancel  context.CancelFunc                   // 取消函数
	stopped bool                                 // 标识worker pool是否已经停止
}

// 初始化worker pool
func NewWorkerPool(size int, ctx context.Context, cancel context.CancelFunc) *WorkerPool {
	return &WorkerPool{
		tasks:  make(chan func(ctx context.Context) error, size),
		ctx:    ctx,
		cancel: cancel,
	}
}

// 开始
func (wp *WorkerPool) Start() {
	for i := 0; i < cap(wp.tasks); i++ {
		go func() {
			for task := range wp.tasks {
				wp.wg.Add(1)
				_ = task(wp.ctx)
				wp.wg.Done()
			}
		}()
	}
}

// 停止
func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
	wp.stopped = true
}

// 提交任务至任务池
func (wp *WorkerPool) Submit(task func(ctx context.Context) error) error {
	if wp.stopped {
		return fmt.Errorf("worker pool has been stopped")
	}
	select {
	case wp.tasks <- task:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}
