// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"context"
	"net/url"
	"strconv"
)

func (client *Client) List(ctx context.Context, dir string, start int) (ListResponse, error) {
	dir = client.AbsPath(dir)
	resp, _, err := client.list(ctx, dir, start)
	return resp, err
}

func (client *Client) ListEx(ctx context.Context, dir string) (ListResponse, error) {
	dir = client.AbsPath(dir)

	var (
		ret   ListResponse
		start int
	)
	for {
		resp, hasMore, err := client.list(ctx, dir, start)
		if err != nil {
			return ret, err
		}
		ret.List = append(ret.List, resp.List...)
		if !hasMore {
			break
		}
		start += len(resp.List)
	}
	return ret, nil
}

func (client *Client) list(ctx context.Context, dir string, start int) (ListResponse, bool, error) {
	const limit = 1000
	var (
		accessAuth = client.GetAccessAuth()
	)
	queryArgs := url.Values{}
	queryArgs.Set("method", "list")
	queryArgs.Set("access_token", accessAuth.AccessToken)
	queryArgs.Set("dir", dir)
	queryArgs.Set("start", strconv.Itoa(start))
	queryArgs.Set("limit", strconv.Itoa(limit))
	queryArgs.Set("showempty", "1") // returns dir_empty attr
	queryArgs.Set("folder", "0")    // not only folder
	queryArgs.Set("web", "0")       // returns no thumbnail

	var resp ListResponse
	if err := client.doHTTPGetJSON(
		ctx,
		newFileAPIURL(),
		queryArgs,
		&resp,
	); err != nil {
		return resp, false, err
	}
	hasMore := len(resp.List) >= limit
	return resp, hasMore, nil
}
