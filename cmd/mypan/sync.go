// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"mypan/pkg/client"
	"mypan/pkg/config"
	"mypan/pkg/store"

	"github.com/golang/glog"
	"github.com/jedib0t/go-pretty/progress"
	"github.com/pkg/errors"
)

var (
	ErrDirUnexpected = fmt.Errorf("not expecting a dir")
	ErrDirExpected   = fmt.Errorf("not a dir")
	ErrNotExist      = fmt.Errorf("not exist")
)

func ErrIsNotExist(err error) bool {
	cause := errors.Cause(err)
	if cause == ErrNotExist {
		return true
	}
	if client.ErrIsNotExist(err) {
		return true
	}
	if os.IsNotExist(err) {
		return true
	}
	return false
}

type CacheSetterI interface {
	SetDst(dstAbsPath, dstMd5, srcMd5 string, size int64)
}

type Src interface {
	Name() string
	RelPath() string
	AbsPath() string
	Size() int64
	IsDir() bool
}

type Dst interface {
	Name() string
	RelPath() string
	AbsPath() string
	Size() int64
	IsDir() bool

	Md5() string
	FsId() uint64
}

type OrderI interface {
	Name() string
	IsDir() bool
}

func Less(i, j OrderI) bool {
	if i.IsDir() == j.IsDir() {
		namei := i.Name()
		namej := j.Name()
		return namei < namej
	} else if i.IsDir() {
		return false
	} else {
		return true
	}
}

type SrcList []Src
type DstList []Dst

func (srcList SrcList) Len() int {
	return len(srcList)
}
func (srcList SrcList) Less(i, j int) bool {
	srci := srcList[i]
	srcj := srcList[j]
	return Less(srci, srcj)
}
func (srcList SrcList) Swap(i, j int) {
	srcList[i], srcList[j] = srcList[j], srcList[i]
}
func (dstList DstList) Len() int {
	return len(dstList)
}
func (dstList DstList) Less(i, j int) bool {
	dsti := dstList[i]
	dstj := dstList[j]
	return Less(dsti, dstj)
}
func (dstList DstList) Swap(i, j int) {
	dstList[i], dstList[j] = dstList[j], dstList[i]
}

type SrcClient interface {
	New(ctx context.Context, path string) (Src, error)
	List(ctx context.Context, src Src) (SrcList, error)
	Delete(ctx context.Context, src Src) error
}

type SrcCacheEntryI interface {
	Md5() string
}

type DstCacheEntryI interface {
	DstMd5() string
	SrcMd5() string
	Size() int64
}

type DstClient interface {
	New(ctx context.Context, path string) (Dst, error)
	List(ctx context.Context, dst Dst) (DstList, error)
	Up(ctx context.Context, src Src, path string) (client.UploadResponse, error)
	Down(ctx context.Context, dst Dst, path string) error
	Delete(ctx context.Context, dst Dst) error
}

type Sync struct {
	client client.ClientI

	src       string
	dst       string
	srcClient SrcClient
	dstClient DstClient

	srcCacheStore *store.FileCacheStore
	dstCacheStore *store.FileCacheStore
	cacheSetter   CacheSetterI

	progress progress.Writer

	up        bool
	dryrun    bool
	nodelete  bool
	continue_ bool
}

func NewSyncUp(
	src, dst string,
	client client.ClientI,
	srcCacheStore *store.FileCacheStore,
	dstCacheStore *store.FileCacheStore,
	opts ...SyncOpt,
) *Sync {
	su := newSync(
		src,
		dst,
		client,
		srcCacheStore,
		dstCacheStore,
		opts...,
	)
	su.up = true
	return su
}

func NewSyncDown(
	src, dst string,
	client client.ClientI,
	srcCacheStore *store.FileCacheStore,
	dstCacheStore *store.FileCacheStore,
	opts ...SyncOpt,
) *Sync {
	su := newSync(
		src,
		dst,
		client,
		srcCacheStore,
		dstCacheStore,
		opts...,
	)
	su.up = false
	return su
}

func newSync(
	src, dst string,
	client client.ClientI,
	srcCacheStore *store.FileCacheStore,
	dstCacheStore *store.FileCacheStore,
	opts ...SyncOpt,
) *Sync {
	cacheSetter := NewCacheSetter(dstCacheStore)
	downMan := NewDownMan(client).CacheSetter(cacheSetter)

	srcClient := SrcClientLocal{}
	dstClient := NewDstClientRemote(client, downMan)
	su := &Sync{
		client: client,

		src:       src,
		dst:       dst,
		srcClient: srcClient,
		dstClient: dstClient,

		srcCacheStore: srcCacheStore,
		dstCacheStore: dstCacheStore,
		cacheSetter:   cacheSetter,
	}
	for _, opt := range opts {
		opt(su)
	}
	if su.dryrun {
		su.srcClient = SrcClientLocalReadOnly{srcClient}
		su.dstClient = DstClientRemoteReadOnly{dstClient}
	}
	downMan.Continue(su.continue_)
	return su
}

type SyncOpt func(*Sync)

func DryRun(dryrun bool) SyncOpt {
	return func(su *Sync) {
		su.dryrun = dryrun
	}
}

func NoDelete(nodelete bool) SyncOpt {
	return func(su *Sync) {
		su.nodelete = nodelete
	}
}

func Continue() SyncOpt {
	return func(su *Sync) {
		su.continue_ = true
	}
}

func Progress(progress progress.Writer) SyncOpt {
	return func(su *Sync) {
		su.progress = progress
	}
}

func (su *Sync) Do(ctx context.Context) error {
	var (
		src     Src
		dst     Dst
		srcList SrcList
		dstList DstList
		err     error
	)
	// Check src and get srcList if available
	src, err = su.srcClient.New(ctx, su.src)
	if err != nil && !ErrIsNotExist(err) {
		return err
	}
	if src != nil {
		srcList, err = su.srcClient.List(ctx, src)
		if err != nil {
			return err
		}
	}

	// Check dst if available
	dst, err = su.dstClient.New(ctx, su.dst)
	if err != nil && !ErrIsNotExist(err) {
		return err
	}

	// type must match
	if src != nil && dst != nil {
		srcIsDir := src.IsDir()
		dstIsDir := dst.IsDir()
		if srcIsDir != dstIsDir {
			return fmt.Errorf("src, dst isdir attr do not match: %v vs. %v", srcIsDir, dstIsDir)
		}
	}
	// get dstList if available
	if dst != nil {
		dstList, err = su.dstClient.List(ctx, dst)
		if err != nil {
			return err
		}
	}
	return su.sync(ctx, srcList, dstList)
}

func (su *Sync) sync(
	ctx context.Context,
	srcList SrcList,
	dstList DstList,
) error {
	sort.Sort(srcList)
	sort.Sort(dstList)
	i, j := 0, 0
	srcsz, dstsz := len(srcList), len(dstList)
	for {
		if i >= srcsz {
			return su.actionDstMore(ctx, dstList[j:]...)
		} else if j >= dstsz {
			return su.actionSrcMore(ctx, srcList[i:]...)
		}
		src1, dst1 := srcList[i], dstList[j]
		srcIsDir := src1.IsDir()
		dstIsDir := dst1.IsDir()
		namei := src1.Name()
		namej := dst1.Name()
		if srcIsDir {
			if dstIsDir {
				if namei == namej {
					srclist1, err := su.srcClient.List(ctx, src1)
					if err != nil {
						return err
					}
					dstlist1, err := su.dstClient.List(ctx, dst1)
					if err != nil {
						return err
					}
					if err := su.sync(ctx, srclist1, dstlist1); err != nil {
						return errors.Wrapf(err, "cmp %q, %q", namei, namej)
					}
					i += 1
					j += 1
				} else {
					if err := su.nameCmpAction(ctx, src1, dst1, &i, &j); err != nil {
						return err
					}
				}
			} else {
				if err := su.nameCmpAction(ctx, src1, dst1, &i, &j); err != nil {
					return err
				}
			}
		} else {
			if dstIsDir {
				if err := su.nameCmpAction(ctx, src1, dst1, &i, &j); err != nil {
					return err
				}
			} else {
				// cmp
				if namei == namej {
					updateCause := ""
					ent := su.getOrSetDstCacheEntry(ctx, dst1.AbsPath())
					if ent == nil {
						updateCause = "no cache"
					} else {
						if src1.Size() != dst1.Size() {
							updateCause = "local size != remote's"
						}
						if src1.Size() != ent.Size() {
							updateCause = "local size != cache's"
						}
						if dst1.Md5() != ent.DstMd5() {
							updateCause = "remote md5 != cache's"
						}
						sce := su.getOrSetSrcCacheEntry(ctx, src1.AbsPath())
						if sce == nil || sce.Md5() != ent.SrcMd5() {
							updateCause = "local md5 != cache's"
						}
					}
					if updateCause != "" {
						glog.V(config.VerboseOn).Infof("update %s: %s", src1.AbsPath(), updateCause)
						if err := su.actionUpdate(ctx, src1, dst1); err != nil {
							return err
						}
					}
					i += 1
					j += 1
				} else {
					if err := su.nameCmpAction(ctx, src1, dst1, &i, &j); err != nil {
						return err
					}
				}
			}
		}
	}
}

func (su *Sync) nameCmpAction(
	ctx context.Context,
	src Src,
	dst Dst,
	i *int,
	j *int,
) error {
	if Less(src, dst) {
		if err := su.actionSrcMore(ctx, src); err != nil {
			return err
		}
		*i = *i + 1
	} else {
		if err := su.actionDstMore(ctx, dst); err != nil {
			return err
		}
		*j = *j + 1
	}
	return nil
}

func (su *Sync) actionSrcMore(
	ctx context.Context,
	srcs ...Src,
) error {
	var action func(context.Context, Src) error
	if su.up {
		action = su.upSrc
	} else {
		action = su.delSrc
	}
	for _, src := range srcs {
		err := action(ctx, src)
		if err != nil {
			return err
		}
	}
	return nil
}

func (su *Sync) actionDstMore(
	ctx context.Context,
	dsts ...Dst,
) error {
	var action func(context.Context, Dst) error
	if su.up {
		action = su.delDst
	} else {
		action = su.downDst
	}
	for _, dst := range dsts {
		err := action(ctx, dst)
		if err != nil {
			return err
		}
	}
	return nil
}

func (su *Sync) actionUpdate(
	ctx context.Context,
	src Src,
	dst Dst,
) error {
	if su.up {
		return su.upSrc(ctx, src)
	} else {
		return su.downDst(ctx, dst)
	}
}

func (su *Sync) upSrc(
	ctx context.Context,
	src Src,
) error {
	if src.IsDir() {
		srcList, err := su.srcClient.List(ctx, src)
		if err != nil {
			return err
		}
		return su.upSrcList(ctx, srcList)
	} else {
		path := su.upRemotePath(src)
		if su.progress != nil {
			message := src.AbsPath()
			pt := NewProgressTracker(su.progress, message)
			ctx = context.WithValue(ctx, client.XloadTrackerKey, pt)
		}
		resp, err := su.dstClient.Up(ctx, src, path)
		if err != nil {
			return err
		}
		sce := su.getOrSetSrcCacheEntry(ctx, src.AbsPath())
		if sce != nil && resp.Md5 != "" {
			su.cacheSetter.SetDst(
				resp.Path,
				resp.Md5,
				sce.Md5(),
				int64(resp.Size),
			)
		}
		return nil
	}
}

func (su *Sync) delSrc(
	ctx context.Context,
	src Src,
) error {
	if su.nodelete {
		glog.Infof("skip deleting local %q", src.AbsPath())
		return nil
	} else {
		return su.srcClient.Delete(ctx, src)
	}
}

func (su *Sync) downDst(
	ctx context.Context,
	dst Dst,
) error {
	path := su.downLocalPath(dst)
	return su.dstClient.Down(ctx, dst, path)
}

func (su *Sync) upRemotePath(src Src) string {
	return filepath.Join(su.dst, src.RelPath())
}

func (su *Sync) downLocalPath(dst Dst) string {
	relpath := dst.RelPath()
	outsub := strings.TrimPrefix(relpath, su.dst)
	abspath := filepath.Join(su.src, outsub)
	return abspath
}

func (su *Sync) delDst(
	ctx context.Context,
	dst Dst,
) error {
	if su.nodelete {
		glog.Infof("skip deleting remote %q", dst.AbsPath())
		return nil
	} else {
		return su.dstClient.Delete(ctx, dst)
	}
}

func (su *Sync) upSrcList(
	ctx context.Context,
	srcList SrcList,
) error {
	for _, src := range srcList {
		err := su.upSrc(ctx, src)
		if err != nil {
			return err
		}
	}
	return nil
}

func (su *Sync) getOrSetDstCacheEntry(ctx context.Context, dstAbsPath string) DstCacheEntryI {
	var (
		v  store.CacheEntry
		ok bool
	)
	v, ok = su.dstCacheStore.Get(dstAbsPath)
	if !ok {
		relpath := su.client.RelPath(dstAbsPath)
		meta, err := su.client.FileMetaByPath(ctx, relpath)
		if err != nil {
			return nil
		}
		httpResp, err := su.client.DownloadByDLink(ctx, meta.DLink)
		if err != nil {
			return nil
		}
		httpResp.Body.Close()
		srcMd5 := httpResp.Header.Get("content-md5")
		if srcMd5 == "" {
			return nil
		}
		su.cacheSetter.SetDst(dstAbsPath, meta.Md5, srcMd5, int64(meta.Size))

		v, ok = su.dstCacheStore.Get(dstAbsPath)
		if !ok {
			return nil
		}
	}
	dce := v.(DstCacheEntry)
	return NewDstCacheEntryImpl(dce)
}

func (su *Sync) getOrSetSrcCacheEntry(ctx context.Context, srcAbsPath string) SrcCacheEntryI {
	// stat call
	fi, err := os.Stat(srcAbsPath)
	if err != nil {
		return nil
	}
	var ino uint64
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		ino = stat.Ino
	}

	// cache hit?
	ce, ok := su.srcCacheStore.Get(srcAbsPath)
	if ok {
		sce := ce.(SrcCacheEntry)
		if sce.Inode == ino && sce.Size == fi.Size() && sce.Mtime.Equal(fi.ModTime()) {
			return NewSrcCacheEntryImpl(sce)
		}
		glog.V(config.VerboseOn).Infof("src cache miss: %s, ino %v %d %d, size %v %d %d, mtime %v %s %s",
			srcAbsPath,
			sce.Inode == ino, sce.Inode, ino,
			sce.Size == fi.Size(), sce.Size, fi.Size(),
			sce.Mtime.Equal(fi.ModTime()), sce.Mtime, fi.ModTime(),
		)
	}

	// cache miss
	// calculate hash
	f, err := os.Open(srcAbsPath)
	if err != nil {
		return nil
	}
	defer f.Close()
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil
	}
	data := h.Sum(nil)
	hashStr := hex.EncodeToString(data)

	// new cache entry
	sce := SrcCacheEntry{
		AbsPath: srcAbsPath,
		Inode:   ino,
		Size:    fi.Size(),
		Mtime:   fi.ModTime(),
		Md5:     hashStr,
	}
	if err := su.srcCacheStore.Set(sce); err != nil {
		glog.Warningf("set src file cache (%s): %v", srcAbsPath, err)
	}
	return NewSrcCacheEntryImpl(sce)
}

type CacheSetter struct {
	dstCacheStore *store.FileCacheStore
}

func NewCacheSetter(dstCacheStore *store.FileCacheStore) *CacheSetter {
	cs := &CacheSetter{
		dstCacheStore: dstCacheStore,
	}
	return cs
}

func (cs *CacheSetter) SetDst(dstAbsPath, dstMd5, srcMd5 string, size int64) {
	if err := cs.dstCacheStore.Set(DstCacheEntry{
		DstAbsPath: dstAbsPath,
		DstMd5:     dstMd5,
		SrcMd5:     srcMd5,
		Size:       size,
	}); err != nil {
		glog.Warningf("set dst file cache (%s): %v", dstAbsPath, err)
	}
}
