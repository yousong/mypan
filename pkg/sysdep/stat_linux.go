// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

//go:build unix


package sysdep

import (
	"syscall"
)

func fiSysGetCtime(sys any) int64 {
	st, ok := sys.(*syscall.Stat_t)
	if !ok {
		return -1
	}
	return int64(st.Ctim.Sec)
}
