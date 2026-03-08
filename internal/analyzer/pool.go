package analyzer

import (
	"context"
	"errors"
)

var ErrQueueFull = errors.New("server busy")

type job struct {
	ctx    context.Context
	url    string
	result chan<- jobResult
}

type jobResult struct {
	result *Result
	err    error
}

type Pool struct {
	jobs chan job
}

func NewPool(a *Analyzer, workers, queueSize int) *Pool {
	p := &Pool{jobs: make(chan job, queueSize)}
	for i := 0; i < workers; i++ {
		go p.worker(a)
	}
	return p
}

func (p *Pool) worker(a *Analyzer) {
	for j := range p.jobs {
		r, err := a.Analyze(j.ctx, j.url)
		j.result <- jobResult{result: r, err: err}
	}
}

func (p *Pool) Submit(ctx context.Context, url string) (*Result, error) {
	ch := make(chan jobResult, 1)
	select {
	case p.jobs <- job{ctx: ctx, url: url, result: ch}:
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, ErrQueueFull
	}
	select {
	case res := <-ch:
		return res.result, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
