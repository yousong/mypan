// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package main

import (
	"context"
	"path/filepath"

	"mypan/pkg/client"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type DstRemote struct {
	name    string
	abspath string
	relpath string
	size    int64
	isDir   bool

	md5  string
	fsId uint64
}

func (dr DstRemote) Name() string {
	return dr.name
}
func (dr DstRemote) RelPath() string {
	return dr.relpath
}
func (dr DstRemote) AbsPath() string {
	return dr.abspath
}
func (dr DstRemote) Size() int64 {
	return dr.size
}
func (dr DstRemote) IsDir() bool {
	return dr.isDir
}
func (dr DstRemote) Md5() string {
	return dr.md5
}

func (dr DstRemote) FsId() uint64 {
	return dr.fsId
}

type DstClientRemote struct {
	client  client.ClientI
	downMan *DownMan
}

var _ DstClient = DstClientRemote{}

func NewDstClientRemote(
	client client.ClientI,
	downMan *DownMan,
) DstClientRemote {
	dcr := DstClientRemote{
		client:  client,
		downMan: downMan,
	}
	return dcr
}

func (dcr DstClientRemote) New(ctx context.Context, path string) (Dst, error) {
	updir := filepath.Dir(path)
	name := filepath.Base(path)

	dstList, err := dcr.list(ctx, updir)
	if err != nil {
		return nil, errors.Wrapf(err, "list updir %q", updir)
	}
	for _, dst := range dstList {
		if dst.Name() == name {
			return dst, nil
		}
	}
	return nil, ErrNotExist
}

func (dcr DstClientRemote) List(ctx context.Context, dst Dst) (DstList, error) {
	abspath := dst.AbsPath()
	if !dst.IsDir() {
		return nil, errors.Wrap(ErrDirExpected, abspath)
	}
	relpath := dst.RelPath()
	dstList, err := dcr.list(ctx, relpath)
	return dstList, err
}

func (dcr DstClientRemote) list(ctx context.Context, path string) (DstList, error) {
	resp, err := dcr.client.ListEx(ctx, path)
	if err != nil {
		return nil, err
	}
	var dstList DstList
	for _, v := range resp.List {
		dr := DstRemote{
			name:    v.ServerFilename,
			size:    int64(v.Size),
			isDir:   v.IsDir != 0,
			abspath: v.Path,
			relpath: dcr.client.RelPath(v.Path),
			md5:     v.Md5,
			fsId:    v.FsId,
		}
		dstList = append(dstList, dr)
	}
	return dstList, nil
}

func (dcr DstClientRemote) Up(ctx context.Context, src Src, path string) (client.UploadResponse, error) {
	var resp client.UploadResponse

	abspath := src.AbsPath()
	if src.IsDir() {
		return resp, errors.Wrap(ErrDirUnexpected, abspath)
	}
	resp, err := dcr.client.Upload(ctx, abspath, path)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (dcr DstClientRemote) Down(ctx context.Context, dst Dst, path string) error {
	relpath := dst.RelPath()
	return dcr.downMan.Down(ctx, relpath, path)
}

func (dcr DstClientRemote) Delete(ctx context.Context, dst Dst) error {
	_, err := dcr.client.Delete(ctx, dst.RelPath())
	return err
}

type DstClientRemoteReadOnly struct {
	DstClientRemote
}

func (dcrro DstClientRemoteReadOnly) Up(ctx context.Context, src Src, path string) (client.UploadResponse, error) {
	remotePath := dcrro.client.AbsPath(path)
	glog.Infof("upload: %q to %q", src.AbsPath(), remotePath)
	return client.UploadResponse{}, nil
}

func (dcrro DstClientRemoteReadOnly) Down(ctx context.Context, dst Dst, path string) error {
	glog.Infof("download: %q to %q", dst.AbsPath(), path)
	return nil
}

func (dcrro DstClientRemoteReadOnly) Delete(ctx context.Context, dst Dst) error {
	glog.Infof("remote delete: %q", dst.AbsPath())
	return nil
}
