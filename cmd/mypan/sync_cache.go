// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package main

import (
	"time"

	"mypan/pkg/store"
)

type DstCacheEntry struct {
	DstAbsPath string
	SrcMd5     string
	DstMd5     string
	Size       int64
}

func (dfc DstCacheEntry) Key() string {
	return dfc.DstAbsPath
}

func NewDstCacheEntry() store.CacheEntry {
	ce := DstCacheEntry{}
	return ce
}

type CacheEntryImplDst struct {
	fc DstCacheEntry
}

func (cei CacheEntryImplDst) DstMd5() string {
	return cei.fc.DstMd5
}
func (cei CacheEntryImplDst) SrcMd5() string {
	return cei.fc.SrcMd5
}
func (cei CacheEntryImplDst) Size() int64 {
	return cei.fc.Size
}

func NewDstCacheEntryImpl(fc DstCacheEntry) CacheEntryImplDst {
	cei := CacheEntryImplDst{
		fc: fc,
	}
	return cei
}

type SrcCacheEntry struct {
	AbsPath string
	Inode   uint64
	Size    int64
	Mtime   time.Time
	Md5     string
}

func (sce SrcCacheEntry) Key() string {
	return sce.AbsPath
}

func NewSrcCacheEntry() store.CacheEntry {
	return SrcCacheEntry{}
}

type SrcCacheEntryImpl struct {
	sce SrcCacheEntry
}

func NewSrcCacheEntryImpl(sce SrcCacheEntry) SrcCacheEntryImpl {
	scei := SrcCacheEntryImpl{
		sce: sce,
	}
	return scei
}

func (scei SrcCacheEntryImpl) Md5() string {
	return scei.sce.Md5
}
