// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

//go:build unix


package sysdep

import (
	"fmt"
	"os"
	"syscall"
)

func fileIdByPath(p string) (uint64, error) {
	fi, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	sys := fi.Sys()
	if stat, ok := sys.(*syscall.Stat_t); ok {
		return stat.Ino, nil
	} else {
		return 0, fmt.Errorf("unknown stat type: %T", sys)
	}
}
