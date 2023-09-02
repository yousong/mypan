// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
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
