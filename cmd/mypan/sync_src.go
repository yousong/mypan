// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mypan/pkg/config"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type SrcLocal struct {
	name    string
	abspath string
	relpath string
	size    int64
	isDir   bool
}

func (sl SrcLocal) Name() string {
	return sl.name
}
func (sl SrcLocal) RelPath() string {
	return sl.relpath
}
func (sl SrcLocal) AbsPath() string {
	return sl.abspath
}
func (sl SrcLocal) Size() int64 {
	return sl.size
}
func (sl SrcLocal) IsDir() bool {
	return sl.isDir
}

type SrcClientLocal struct{}

var _ SrcClient = SrcClientLocal{}

func (scl SrcClientLocal) New(ctx context.Context, path string) (Src, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if err := scl.checkFileInfo(fi); err != nil {
		return nil, errors.Wrap(err, path)
	}
	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	var relpath string
	if fi.IsDir() {
		relpath = ""
	} else {
		relpath = filepath.Base(abspath)
	}
	sl := SrcLocal{
		name:    fi.Name(),
		size:    fi.Size(),
		isDir:   fi.IsDir(),
		abspath: abspath,
		relpath: relpath,
	}
	return sl, nil
}

func (scl SrcClientLocal) List(ctx context.Context, src Src) (SrcList, error) {
	if !src.IsDir() {
		return nil, errors.Wrap(ErrDirExpected, src.AbsPath())
	}
	basedir := src.AbsPath()
	des, err := os.ReadDir(basedir)
	if err != nil {
		return nil, err
	}
	sclAbspath := strings.TrimSuffix(basedir, src.RelPath())
	var srclist SrcList
	for _, de := range des {
		fi, err := de.Info()
		if err != nil {
			return nil, err
		}
		name := de.Name()
		abspath := filepath.Join(basedir, name)
		if err := scl.checkFileInfo(fi); err != nil {
			glog.V(config.VerboseOn).Infof("skipping %q: %v", abspath, err)
			continue
		}
		isDir := fi.IsDir()
		var size int64
		if !isDir {
			size = fi.Size()
		}
		srclist = append(srclist, SrcLocal{
			name:    name,
			size:    size,
			isDir:   isDir,
			abspath: abspath,
			relpath: strings.TrimPrefix(abspath, sclAbspath),
		})
	}
	return srclist, nil
}

func (scl SrcClientLocal) Delete(ctx context.Context, src Src) error {
	abspath := src.AbsPath()
	return os.RemoveAll(abspath)
}

func (scl SrcClientLocal) checkFileInfo(fi os.FileInfo) error {
	mode := fi.Mode() & os.ModeType
	if (mode & (^os.ModeDir)) != 0 {
		return fmt.Errorf("only file/dir allowed (mode %s)", mode)
	}
	return nil
}

type SrcClientLocalReadOnly struct {
	SrcClientLocal
}

func (sclro SrcClientLocalReadOnly) Delete(ctx context.Context, src Src) error {
	glog.Infof("local delete: %q", src.AbsPath())
	return nil
}
