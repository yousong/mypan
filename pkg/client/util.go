// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

const (
	blockSize = UPLOAD_API_BLOCK_SIZE
)

func computeFileBlockList(name string) ([]string, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return computeReaderBlockList(f)
}

func computeReaderBlockList(r io.Reader) ([]string, error) {
	var (
		blockList []string
		csum      = md5.New()
	)
	for {
		n, err := io.CopyN(csum, r, blockSize)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n > 0 {
			v := make([]byte, 0, csum.Size())
			v = csum.Sum(v)
			blockList = append(blockList, hex.EncodeToString(v))
			csum.Reset()
		}
		if err == io.EOF {
			break
		}
	}
	return blockList, nil
}
