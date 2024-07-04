// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package util

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

type ParallelDoFunc func(ctx context.Context) error

type ParallelDoOpt func(pd *ParallelDo)

func CheckErrBeforeDo(checkErrBeforeDo bool) ParallelDoOpt {
	return func(pd *ParallelDo) {
		pd.checkErrBeforeDo = checkErrBeforeDo
	}
}
func JoinOnCheckErr(joinOnCheckErr bool) ParallelDoOpt {
	return func(pd *ParallelDo) {
		pd.joinOnCheckErr = joinOnCheckErr
	}
}

type ParallelDo struct {
	sem *semaphore.Weighted
	wg  *sync.WaitGroup

	checkErrBeforeDo bool
	joinOnCheckErr   bool

	mu   *sync.Mutex
	errs []error
}

func NewParallelDo(n int, opts ...ParallelDoOpt) *ParallelDo {
	pd := &ParallelDo{
		sem: semaphore.NewWeighted(int64(n)),
		wg:  &sync.WaitGroup{},

		checkErrBeforeDo: true,
		joinOnCheckErr:   false,

		mu: &sync.Mutex{},
	}
	for _, opt := range opts {
		opt(pd)
	}
	return pd
}

func (pd *ParallelDo) err() error {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	if len(pd.errs) > 0 {
		err := NewMultiError(pd.errs...)
		pd.errs = nil
		return err
	}
	return nil
}

func (pd *ParallelDo) appendErr(err error) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.errs = append(pd.errs, err)
}

// Do calls f with ctx.  Before calling f it will check and return an error
// should any previous calls return a non-nil error.  Otherwise the return will
// be nil.  As such callers can retry if they got a non-nil error return
func (pd *ParallelDo) Do(ctx context.Context, f ParallelDoFunc) error {
	if pd.checkErrBeforeDo {
		if err := pd.err(); err != nil {
			if pd.joinOnCheckErr {
				if joinErr := pd.Join(ctx); joinErr != nil {
					return NewMultiError(err, joinErr)
				}
			}
			return err
		}
	}

	err := pd.sem.Acquire(ctx, 1)
	if err != nil {
		return err
	}
	pd.wg.Add(1)
	go func() {
		defer pd.wg.Done()
		defer pd.sem.Release(1)
		err := f(ctx)
		if err != nil {
			pd.appendErr(err)
		}
	}()
	return nil
}

func (pd *ParallelDo) Join(ctx context.Context) error {
	pd.wg.Wait()
	return pd.err()
}

func TryParallelDo(ctx context.Context, pd *ParallelDo, f ParallelDoFunc) error {
	if pd == nil {
		return f(ctx)
	}
	return pd.Do(ctx, f)
}

func TryParallelJoin(ctx context.Context, pd *ParallelDo) error {
	if pd == nil {
		return nil
	}
	return pd.Join(ctx)
}
