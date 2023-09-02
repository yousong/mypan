// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"context"

	"mypan/pkg/config"

	"github.com/golang/glog"
)

type ReadOnlyClient struct {
	ClientI
}

func NewReadOnly(client ClientI) ClientI {
	roc := &ReadOnlyClient{
		ClientI: client,
	}
	return roc
}

func (roc *ReadOnlyClient) log(fmtStr string, args ...interface{}) {
	glog.V(config.VerboseOn).Infof(fmtStr, args...)
}

func (roc *ReadOnlyClient) Upload(
	ctx context.Context,
	src, dst string,
) (UploadResponse, error) {
	roc.log("skip: upload: src %q, dst %q", src, dst)
	return UploadResponse{}, nil
}

func (roc *ReadOnlyClient) Delete(
	ctx context.Context,
	file string,
) (FileManagerResponse, error) {
	roc.log("skip: delete: file %q", file)
	return FileManagerResponse{}, nil
}

func (roc *ReadOnlyClient) DeleteMulti(
	ctx context.Context,
	fileList []string,
) (FileManagerResponse, error) {
	roc.log("skip: delete multi: file list %q", fileList)
	return FileManagerResponse{}, nil
}
