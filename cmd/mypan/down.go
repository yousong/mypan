// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mypan/pkg/client"
	"mypan/pkg/util"

	"github.com/dustin/go-humanize"
	"github.com/golang/glog"
	"github.com/jedib0t/go-pretty/progress"
	"github.com/pkg/errors"
)

var UnitBytesIEC = progress.Units{
	Formatter: func(value int64) string {
		return humanize.IBytes(uint64(value))
	},
}

type DownMan struct {
	client client.ClientI

	cacheSetter CacheSetterI
	progress    progress.Writer
	continue_   bool
}

func NewDownMan(client client.ClientI) *DownMan {
	dm := &DownMan{
		client: client,
	}
	return dm
}

func (dm *DownMan) CacheSetter(cacheSetter CacheSetterI) *DownMan {
	dm.cacheSetter = cacheSetter
	return dm
}

func (dm *DownMan) Progress(progress progress.Writer) *DownMan {
	dm.progress = progress
	return dm
}

func (dm *DownMan) Continue(continue_ bool) *DownMan {
	dm.continue_ = continue_
	return dm
}

func (dm *DownMan) Down(
	ctx context.Context,
	relpath, outpath string,
) error {
	meta, err := dm.client.FileMetaByPath(ctx, relpath)
	if err != nil {
		return errors.Wrap(err, "meta")
	}
	if meta.IsDir == 0 {
		return dm.down(ctx, relpath, outpath, meta.DLink)
	}
	return dm.downDir(ctx, relpath, outpath)
}

func (dm *DownMan) downDir(
	ctx context.Context,
	relpath, outpath string,
) error {
	ents, err := dm.client.ListAllEx(ctx, relpath)
	if err != nil {
		return errors.Wrap(err, "list all")
	}
	abspath := dm.client.AbsPath(relpath)
	for _, ent := range ents.List {
		if ent.IsDir != 0 {
			continue
		}
		outpath := filepath.Join(outpath, strings.TrimPrefix(ent.Path, abspath))
		relpath := dm.client.RelPath(ent.Path)
		err := dm.downFileByFsId(ctx, relpath, outpath, ent.FsId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dm *DownMan) downFileByFsId(
	ctx context.Context,
	relpath string,
	outpath string,
	fsId uint64,
) error {
	meta, err := dm.client.FileMeta(ctx, fsId)
	if err != nil {
		return err
	}
	return dm.down(ctx, relpath, outpath, meta.DLink)
}

func (dm *DownMan) down(
	ctx context.Context,
	relpath string,
	outpath string,
	dlink string,
) error {
	var (
		opts    []func(*http.Request)
		w       io.Writer
		tmpname string
	)
	if outpath == "" {
		w = os.Stdout
	} else {
		dir := filepath.Dir(outpath)
		if err := util.MkdirAll(dir); err != nil {
			return err
		}
		tmpname0 := outpath + ".downloading"
		if dm.continue_ {
			fi, err := os.Stat(tmpname0)
			if err == nil {
				offset := fi.Size()
				opts = append(opts, func(req *http.Request) {
					req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
				})
				f, err := os.OpenFile(tmpname0, os.O_APPEND|os.O_WRONLY, os.FileMode(0666))
				if err != nil {
					return err
				}
				defer f.Close()
				w = f
			}
			// if it's syncdown, it's ensured by sync that we
			// should do a update (download)
			//
			// for plain download command, we expect the user is
			// responsible for viability of continue flag
		}
		if w == nil {
			f, err := os.Create(tmpname0)
			if err != nil {
				return err
			}
			defer f.Close()
			w = f
		}
		tmpname = tmpname0
	}

	if progress := dm.progress; progress != nil {
		pt := NewProgressTracker(dm.progress, relpath)
		ctx = context.WithValue(ctx, client.XloadTrackerKey, pt)
	}
	httpResp, err := dm.client.DownloadByDLink(ctx, dlink, opts...)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	if outpath != "" {
		// NOTE not sure if Content-Md5 header is reliable
		srcMd5 := httpResp.Header.Get("content-md5")
		if srcMd5 == "" {
			glog.Warningf("content-md5 header absent")
		} else {
			defer dm.callCacheSetter(ctx, relpath, srcMd5)
		}
	}

	if _, err := io.Copy(w, httpResp.Body); err != nil {
		return err
	}
	if tmpname != "" {
		err := os.Rename(tmpname, outpath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dm *DownMan) callCacheSetter(ctx context.Context, relpath, srcMd5 string) {
	if dm.cacheSetter == nil {
		return
	}
	abspath := dm.client.AbsPath(relpath)
	meta, err := dm.client.FileMetaByPath(ctx, relpath)
	if err != nil {
		glog.Warningf("filemeta %q: %v", abspath, err)
	}
	dm.cacheSetter.SetDst(
		meta.Path, meta.Md5,
		srcMd5,
		int64(meta.Size),
	)
}
