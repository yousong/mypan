// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package util

import (
	"os"
)

func MkdirAll(dir string) error {
	return os.MkdirAll(dir, os.FileMode(0755))
}
