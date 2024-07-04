// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
)

const (
	opCopy   = "copy"
	opRename = "rename"
	opMove   = "move"
	opDelete = "delete"
)

const (
	asyncNo   = 0 // 同步
	asyncAuto = 1 // 自适应
	asyncYes  = 2 // 异步
)

func (client *Client) Delete(
	ctx context.Context,
	file string,
) (FileManagerResponse, error) {
	resp, err := client.DeleteMulti(ctx, []string{file})
	return resp, err
}

func (client *Client) DeleteMulti(
	ctx context.Context,
	fileList []string,
) (FileManagerResponse, error) {
	fileList1 := make([]string, len(fileList))
	for i, file := range fileList {
		fileList1[i] = client.AbsPath(file)
	}
	resp, err := client.doFileManager(ctx, opDelete, fileList1)
	return resp, err
}

func (client *Client) Rename(
	ctx context.Context,
	path, newname string,
) (FileManagerResponse, error) {
	resp, err := client.renameMulti(ctx, [2]string{path, newname})
	return resp, err
}

func (client *Client) renameMulti(
	ctx context.Context,
	pairs ...[2]string,
) (FileManagerResponse, error) {
	var filelist []map[string]string
	for _, p := range pairs {
		filelist = append(filelist, map[string]string{
			"path":    client.AbsPath(p[0]),
			"newname": p[1],
		})
	}
	resp, err := client.doFileManager(ctx, opRename, filelist)
	return resp, err
}

func (client *Client) Copy(
	ctx context.Context,
	path, dest string,
) (FileManagerResponse, error) {
	resp, err := client.copyMoveMulti(ctx, opCopy, [2]string{path, dest})
	return resp, err
}

func (client *Client) Move(
	ctx context.Context,
	path, dest string,
) (FileManagerResponse, error) {
	resp, err := client.copyMoveMulti(ctx, opMove, [2]string{path, dest})
	return resp, err
}

func (client *Client) copyMoveMulti(
	ctx context.Context,
	op string,
	pairs ...[2]string,
) (FileManagerResponse, error) {
	var filelist []map[string]string
	for _, p := range pairs {
		dest := client.AbsPath(p[1])
		newname := filepath.Base(dest)
		dest = filepath.Dir(dest)
		filelist = append(filelist, map[string]string{
			"path":    client.AbsPath(p[0]),
			"dest":    dest,
			"newname": newname,
		})
	}
	resp, err := client.doFileManager(ctx, op, filelist)
	return resp, err
}

func (client *Client) doFileManager(
	ctx context.Context,
	op string,
	filelist interface{},
) (FileManagerResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
		resp       FileManagerResponse
	)
	filelistData, err := json.Marshal(filelist)
	if err != nil {
		return resp, errors.Wrap(err, "marshal filelist")
	}

	queryArgs := url.Values{}
	queryArgs.Set("method", "filemanager")
	queryArgs.Set("access_token", accessAuth.AccessToken)
	queryArgs.Set("opera", op)

	bodyArgs := url.Values{}
	bodyArgs.Set("async", strconv.Itoa(asyncAuto))
	bodyArgs.Set("filelist", string(filelistData))
	if op != opDelete {
		bodyArgs.Set("ondup", ONDUP_OVERWRITE)
	}
	bodyStr := bodyArgs.Encode()
	if err := client.doHTTPPostFormJSON(
		ctx,
		newFileAPIURL(),
		queryArgs,
		bytes.NewBufferString(bodyStr),
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}
