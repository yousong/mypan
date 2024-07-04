// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package sysdep

import "syscall"

func fiSysGetCtime(sys any) int64 {
	st, ok := sys.(*syscall.Stat_t)
	if !ok {
		return -1
	}
	if st.Birthtimespec.Sec != 0 || st.Birthtimespec.Nsec != 0 {
		return st.Birthtimespec.Sec
	}
	return st.Ctimespec.Sec
}
