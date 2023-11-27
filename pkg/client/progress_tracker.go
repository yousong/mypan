// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"context"
	"io"
)

type XloadTrackerI interface {
	Start(total int64)
	Increment(n int64)
	Done()
}

type xloadTrackerKeyType struct{}

var XloadTrackerKey xloadTrackerKeyType

func xloadTracker(ctx context.Context) XloadTrackerI {
	v := ctx.Value(XloadTrackerKey)
	if xlt, ok := v.(XloadTrackerI); ok {
		return xlt
	}
	return nil
}

type lenI interface {
	Len() int
}

type readTracker struct {
	reader  io.Reader
	tracker XloadTrackerI
}

func newReadTracker(reader io.Reader, tracker XloadTrackerI) readTracker {
	rt := readTracker{
		reader:  reader,
		tracker: tracker,
	}
	return rt
}

func newReadTrackerWithCtx(ctx context.Context, r io.Reader) (io.Reader, XloadTrackerI) {
	if xlt := xloadTracker(ctx); xlt != nil {
		rt := newReadTracker(r, xlt)
		xlt.Start(int64(rt.Len()))
		return rt, xlt
	}
	return r, nil
}

func (rt readTracker) Read(p []byte) (int, error) {
	n, err := rt.reader.Read(p)
	rt.tracker.Increment(int64(n))
	return n, err
}

func (rt readTracker) Len() int {
	if l, ok := rt.reader.(lenI); ok {
		return l.Len()
	}
	return -1
}

type readCloseTracker struct {
	readCloser io.ReadCloser
	tracker    XloadTrackerI
}

func newReadCloseTracker(readCloser io.ReadCloser, tracker XloadTrackerI) readCloseTracker {
	rct := readCloseTracker{
		readCloser: readCloser,
		tracker:    tracker,
	}
	return rct
}

func newReadCloseTrackerWithCtx(ctx context.Context, rc io.ReadCloser, total, done int64) io.ReadCloser {
	if xlt := xloadTracker(ctx); xlt != nil {
		rct := newReadCloseTracker(rc, xlt)
		xlt.Start(total)
		xlt.Increment(done)
		return rct
	}
	return rc
}

func (rct readCloseTracker) Read(p []byte) (int, error) {
	n, err := rct.readCloser.Read(p)
	rct.tracker.Increment(int64(n))
	return n, err
}

func (rct readCloseTracker) Close() error {
	defer rct.tracker.Done()
	return rct.readCloser.Close()
}
