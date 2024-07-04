// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package util

import (
	"github.com/golang/glog"
)

func MustString(p *string) string {
	if p == nil {
		glog.Fatal("nil string ptr")
	}
	return *p
}
