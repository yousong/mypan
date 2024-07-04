// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package util

import (
	"encoding/json"

	"github.com/golang/glog"
)

func MustMarshalJSON(v interface{}) []byte {
	d, err := json.Marshal(v)
	if err != nil {
		glog.Fatalf("must marshal %#v: %v", v, err)
	}
	return d
}
