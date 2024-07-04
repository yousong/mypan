// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package main

import (
	"bytes"
	"context"
	"os/exec"

	"mypan/pkg/client"
	"mypan/pkg/util"

	"github.com/pkg/errors"
)

type WalkerStdin struct {
	Path  string `json:"path"`
	IsDir int    `json:"isdir"`

	Size        uint64 `json:"size"`
	ServerCtime uint64 `json:"server_ctime"`
	ServerMtime uint64 `json:"server_mtime"`
	LocalCtime  uint64 `json:"local_ctime"`
	LocalMtime  uint64 `json:"local_mtime"`
}

type Walker struct {
	client   client.ClientI
	execArgv []string

	parallelDo *util.ParallelDo
}

func NewWalker(
	client client.ClientI,
	execArgv []string,
) *Walker {
	w := &Walker{
		client:   client,
		execArgv: execArgv,
	}
	return w
}

func (w *Walker) Parallel(n int) {
	w.parallelDo = util.NewParallelDo(n,
		util.JoinOnCheckErr(true),
	)
}

func (w *Walker) Walk(ctx context.Context, dir string) error {
	if err := w.exec(ctx, &WalkerStdin{
		Path:  dir,
		IsDir: 1,
	}); err != nil {
		return err
	}
	resp, err := w.client.ListEx(ctx, dir)
	if err != nil {
		return errors.Wrapf(err, "list %s", dir)
	}
	for _, src := range resp.List {
		if src.IsDir != 0 {
			// we check src.Empty by ourselves
			err := w.Walk(ctx, src.Path)
			if err != nil {
				return err
			}
		} else {
			err := w.exec(ctx, &WalkerStdin{
				Path:  src.Path,
				IsDir: src.IsDir,

				Size:        src.Size,
				ServerCtime: src.ServerCtime,
				ServerMtime: src.ServerMtime,
				LocalCtime:  src.LocalCtime,
				LocalMtime:  src.LocalMtime,
			})
			if err != nil {
				return err
			}
		}
	}
	return util.TryParallelJoin(ctx, w.parallelDo)
}

func (w *Walker) exec(ctx context.Context, stdin *WalkerStdin) error {
	return util.TryParallelDo(ctx, w.parallelDo, func(ctx context.Context) error {
		return w.exec_(ctx, stdin)
	})
}

func (w *Walker) exec_(ctx context.Context, stdin *WalkerStdin) error {
	stdinData := util.MustMarshalJSON(stdin)
	stdinReader := bytes.NewBuffer(stdinData)
	cmd := exec.CommandContext(ctx, w.execArgv[0], w.execArgv[1:]...)
	cmd.Stdin = stdinReader
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "exec for %s: %s", stdin.Path, output)
	}
	return nil
}
