// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package client

import (
	"bytes"
	"testing"
)

func TestComputeBlockList(t *testing.T) {
	for _, c := range []struct {
		Name      string
		DataLen   int
		BlockList []string
	}{
		{
			Name: "empty",
		}, {
			Name:    "one-byte",
			DataLen: 1,
			BlockList: []string{
				"7fc56270e7a70fa81a5935b72eacbe29",
			},
		}, {
			Name:    "blockSize",
			DataLen: blockSize,
			BlockList: []string{
				"4a31ac3594cb245c08e134ec06b3057e",
			},
		}, {
			Name:    "blockSize+1",
			DataLen: blockSize + 1,
			BlockList: []string{
				"4a31ac3594cb245c08e134ec06b3057e",
				"7fc56270e7a70fa81a5935b72eacbe29",
			},
		},
	} {
		t.Run(c.Name, func(t *testing.T) {
			r := &bytes.Buffer{}
			for i := 0; i < c.DataLen; i++ {
				r.WriteByte('A')
			}
			blockList, err := computeReaderBlockList(r)
			if err != nil {
				t.Errorf("%s: compute error: %v", c.Name, err)
			}
			if len(blockList) != len(c.BlockList) {
				t.Errorf("%s: block list size: want %d, got %d",
					c.Name, len(c.BlockList), len(blockList))
			}
			for i := range blockList {
				if blockList[i] != c.BlockList[i] {
					t.Errorf("%s: block list value not equal: %d: want %s, got %s",
						c.Name, i, c.BlockList[i], blockList[i])
				}
			}
		})
	}
}
