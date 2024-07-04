// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package sysdep

import "os"

func FileInfoGetCtime(fi os.FileInfo) int64 {
	sys := fi.Sys()
	sec := fiSysGetCtime(sys)
	return sec
}

func FileIdByPath(p string) (uint64, error) {
	return fileIdByPath(p)
}
