package workerpool

import (
	"runtime"
	"sync"
)

type Job func()

type WorkerPool struct {
	workerCount int
	jobQueue    chan Job
	wg          sync.WaitGroup
}

func NewWorkerPool() *WorkerPool {
	return &WorkerPool{
		workerCount: runtime.NumCPU(),
		jobQueue:    make(chan Job, 100),
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			for job := range wp.jobQueue {
				job()
			}
		}()
	}
}

func (wp *WorkerPool) Submit(job Job) {
	wp.jobQueue <- job
}

func (wp *WorkerPool) Stop() {
	close(wp.jobQueue)
	wp.wg.Wait()
}
