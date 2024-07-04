// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package sysdep

import (
	"syscall"
)

func fiSysGetCtime(sys any) int64 {
	attr, ok := sys.(*syscall.Win32FileAttributeData)
	if !ok {
		return -1
	}
	nsec := attr.CreationTime.Nanoseconds()
	sec := nsec / (1e9)
	return sec
}

func fileIdByPath(p string) (uint64, error) {
	pathp, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return 0, err
	}

	// Per https://learn.microsoft.com/en-us/windows/win32/fileio/reparse-points-and-file-operations,
	// “Applications that use the CreateFile function should specify the
	// FILE_FLAG_OPEN_REPARSE_POINT flag when opening the file if it is a reparse
	// point.”
	//
	// And per https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew,
	// “If the file is not a reparse point, then this flag is ignored.”
	//
	// So we set FILE_FLAG_OPEN_REPARSE_POINT unconditionally, since we want
	// information about the reparse point itself.
	//
	// If the file is a symlink, the symlink target should have already been
	// resolved when the fileStat was created, so we don't need to worry about
	// resolving symlink reparse points again here.
	attrs := uint32(syscall.FILE_FLAG_BACKUP_SEMANTICS | syscall.FILE_FLAG_OPEN_REPARSE_POINT)

	h, err := syscall.CreateFile(pathp, 0, 0, nil, syscall.OPEN_EXISTING, attrs, 0)
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(h)
	var i syscall.ByHandleFileInformation
	err = syscall.GetFileInformationByHandle(h, &i)
	if err != nil {
		return 0, err
	}
	fileIndex := (uint64(i.FileIndexHigh) << 32) | uint64(i.FileIndexLow)
	return fileIndex, nil
}
