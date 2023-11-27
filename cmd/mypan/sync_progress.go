// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package main

import (
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/progress"
)

type ProgressTracker struct {
	progress progress.Writer
	message  string

	tracker *progress.Tracker
}

func NewProgressTracker(progress progress.Writer, message string) *ProgressTracker {
	pt := &ProgressTracker{
		progress: progress,
		message:  message,
	}
	return pt
}

func (pt *ProgressTracker) Start(total int64) {
	pt.tracker = &progress.Tracker{
		Message: fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), pt.message),
		Units:   UnitBytesIEC,
		Total:   total,
	}
	pt.progress.AppendTracker(pt.tracker)
}

func (pt *ProgressTracker) Increment(n int64) {
	pt.tracker.Increment(n)
}

func (pt *ProgressTracker) Done() {
	pt.tracker.MarkAsDone()
}
