package engine

import (
	"context"
	"sync"

	"github.com/digitalohara/webhound/internal/response"
)

// WorkerPool manages a fixed pool of goroutines consuming jobs.
type WorkerPool struct {
	size      int
	requester *Requester
}

// NewWorkerPool creates a WorkerPool with size concurrent workers.
func NewWorkerPool(size int, requester *Requester) *WorkerPool {
	return &WorkerPool{size: size, requester: requester}
}

// Run reads jobs from jobCh, executes them, and sends results to resultCh.
// It closes resultCh when all workers have exited.
// The caller is responsible for closing jobCh when no more jobs will be sent.
func (p *WorkerPool) Run(ctx context.Context, jobCh <-chan response.Job, resultCh chan<- *response.RawResult) {
	var wg sync.WaitGroup
	for i := 0; i < p.size; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case job, ok := <-jobCh:
					if !ok {
						return
					}
					raw := p.requester.Execute(ctx, job)
					select {
					case resultCh <- raw:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(resultCh)
	}()
}

// ProcessJobs is a convenience function that creates channels, starts the pool,
// feeds all jobs, and returns the collected raw results.
func (p *WorkerPool) ProcessJobs(ctx context.Context, jobs []response.Job) []*response.RawResult {
	jobCh := make(chan response.Job, p.size*2)
	resultCh := make(chan *response.RawResult, p.size*2)

	p.Run(ctx, jobCh, resultCh)

	go func() {
		defer close(jobCh)
		for _, job := range jobs {
			select {
			case jobCh <- job:
			case <-ctx.Done():
				return
			}
		}
	}()

	var results []*response.RawResult
	for raw := range resultCh {
		results = append(results, raw)
	}
	return results
}
