package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Task представляет задачу для выполнения.
type Task func(ctx context.Context) error

// Pool представляет пул воркеров для асинхронной обработки задач.
type Pool struct {
	ctx       context.Context
	taskQueue chan Task
	cancel    context.CancelFunc
	logger    *zap.Logger
	wg        sync.WaitGroup
	workers   int
}

// NewPool создает новый пул воркеров.
func NewPool(workers int, queueSize int, logger *zap.Logger) *Pool {
	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		workers:   workers,
		taskQueue: make(chan Task, queueSize),
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
	}
}

// Start запускает воркеры.
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	p.logger.Info("Worker pool started", zap.Int("workers", p.workers))
}

// worker обрабатывает задачи из очереди.
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			p.logger.Info("Worker stopped", zap.Int("worker_id", id))
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}

			if err := task(p.ctx); err != nil {
				p.logger.Error("Task execution failed",
					zap.Int("worker_id", id),
					zap.Error(err),
				)
			}
		}
	}
}

// Submit отправляет задачу в очередь.
func (p *Pool) Submit(task Task) error {
	select {
	case p.taskQueue <- task:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

// Shutdown завершает работу пула воркеров.
func (p *Pool) Shutdown() {
	p.logger.Info("Shutting down worker pool...")
	close(p.taskQueue)
	p.cancel()
	p.wg.Wait()
	p.logger.Info("Worker pool shutdown complete")
}
