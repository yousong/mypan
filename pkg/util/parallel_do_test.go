// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package util

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type TestSuiteParallelDo struct {
	suite.Suite
}

func (s *TestSuiteParallelDo) SetupTest() {
}

func (s *TestSuiteParallelDo) doFunc(
	dur time.Duration,
	run *bool,
	err error,
) ParallelDoFunc {
	return func(ctx context.Context) error {
		if run != nil {
			*run = true
		}
		time.Sleep(dur)
		return err
	}
}

func (s *TestSuiteParallelDo) doFuncSleep(
	dur time.Duration,
	run *bool,
) ParallelDoFunc {
	return s.doFunc(dur, nil, nil)
}

func (s *TestSuiteParallelDo) Test_DoErr() {
	ctx := context.Background()
	pd := NewParallelDo(2)
	wantErr := fmt.Errorf("an error")
	wg := &sync.WaitGroup{}
	wg.Add(1)
	s.Nil(pd.Do(ctx, func(ctx context.Context) error {
		defer wg.Done()
		return wantErr
	}))
	wg.Wait()

	run := false
	gotErr := pd.Do(ctx, func(ctx context.Context) error {
		run = true
		return nil
	})
	s.Equal(wantErr, gotErr)
	s.Nil(pd.Join(ctx))
	s.False(run)
}

func (s *TestSuiteParallelDo) Test_JoinErr() {
	ctx := context.Background()
	pd := NewParallelDo(2)
	for i := 0; i < 3; i++ {
		s.Nil(pd.Do(ctx, func(ctx context.Context) error {
			time.Sleep(time.Second)
			return fmt.Errorf("error %d", i)
		}))
	}

	run := false
	gotErr0 := pd.Do(ctx, func(ctx context.Context) error {
		run = true
		return nil
	})
	s.NotNil(gotErr0)

	gotErr1 := pd.Join(ctx)
	s.False(run)
	s.NotNil(gotErr1)
	multiErr0, ok0 := gotErr0.(MultiError)
	multiErr1, ok1 := gotErr1.(MultiError)
	if ok0 {
		s.False(ok1)
		s.Equal(2, len(multiErr0))
	} else {
		s.False(ok0)
		s.Equal(2, len(multiErr1))
	}
}

func TestParallelDo(t *testing.T) {
	suite.Run(t, &TestSuiteParallelDo{})
}
