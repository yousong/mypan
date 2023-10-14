// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"

	"mypan/pkg/util"

	"github.com/pkg/errors"
)

func (client *Client) FileMetaByPath(ctx context.Context, relpath string) (FileMetaResponse, error) {
	var resp FileMetaResponse

	fileMetasResponse, err := client.FileMetasByPath(ctx, []string{relpath})
	if err != nil {
		return resp, err
	}
	// just in case
	if len(fileMetasResponse.List) == 0 {
		return resp, errors.Wrap(errNotFound, relpath)
	}
	resp = fileMetasResponse.List[0]
	return resp, nil
}

func (client *Client) FileMetasByPath(ctx context.Context, relpaths []string) (FileMetasResponse, error) {
	var resp FileMetasResponse

	dirNames := map[string][]string{}
	for _, relpath := range relpaths {
		abspath := client.AbsPath(relpath)
		dir := filepath.Dir(abspath)
		name := filepath.Base(abspath)
		dirNames[dir] = append(dirNames[dir], name)
	}
	var fsIds []uint64
	for dir, names := range dirNames {
		listResp, err := client.ListEx(ctx, dir)
		if err != nil {
			return resp, errors.Wrapf(err, "list %q", dir)
		}
		for _, resp := range listResp.List {
			for _, name := range names {
				if resp.ServerFilename == name {
					fsIds = append(fsIds, resp.FsId)
					break
				}
			}
		}
	}
	if len(fsIds) != len(relpaths) {
		return resp, errors.Wrap(errNotFound, "some")
	}
	resp, err := client.FileMetas(ctx, fsIds)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (client *Client) FileMeta(ctx context.Context, fsId uint64) (FileMetaResponse, error) {
	var resp FileMetaResponse

	fileMetasResponse, err := client.FileMetas(ctx, []uint64{fsId})
	if err != nil {
		return resp, err
	}
	// just in case
	if len(fileMetasResponse.List) == 0 {
		return resp, errors.Wrap(errNotFound, fmt.Sprintf("fsId %d", fsId))
	}
	resp = fileMetasResponse.List[0]
	return resp, nil
}

func (client *Client) FileMetas(ctx context.Context, fsIds []uint64) (FileMetasResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
		fsIdsData  = string(util.MustMarshalJSON(fsIds))
		resp       FileMetasResponse
	)
	queryArgs := url.Values{}
	queryArgs.Set("method", "filemetas")
	queryArgs.Set("access_token", accessAuth.AccessToken)
	queryArgs.Set("fsids", fsIdsData)
	queryArgs.Set("dlink", strconv.Itoa(1))
	queryArgs.Set("thumb", strconv.Itoa(0))
	if err := client.doHTTPGetJSON(
		ctx,
		newMultimediaAPIURL(),
		queryArgs,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}
