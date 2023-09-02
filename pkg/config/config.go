// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package config

import (
	"os"
	"path"
)

type Config struct {
	AppID      int64
	AppKey     string
	SecretKey  string
	AppBaseDir string

	RunDir string
}

var Global Config

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		os.Exit(1)
	}
	runDir := path.Join(homeDir, ".mypan")

	Global = Config{
		RunDir: runDir,

		AppID:      AppID,
		AppKey:     AppKey,
		SecretKey:  SecretKey,
		AppBaseDir: AppBaseDir,
	}
}

const (
	AppID      = 40079350
	AppKey     = "4uwf4wql9Gtg3Dr79r6sKRgrac4M9uc1"
	SecretKey  = "1mBQ9NOpW33EjLcYGzWQxTGUSNteZSfX"
	AppBaseDir = "/apps/mypan"
)

const (
	StoreKeyAccessAuth    = "accessAuth.json"
	StoreKeyDstCacheEntry = "dst_filecache.json"
	StoreKeySrcCacheEntry = "src_filecache.json"
)

const (
	VerboseOff = iota
	// - debug message
	Verbose1
	// - http req/resp header
	Verbose2
	// - http req/resp body
	Verbose3

	VerboseOn       = Verbose1
	VerboseHTTP     = Verbose2
	VerboseHTTPBody = Verbose3
)
