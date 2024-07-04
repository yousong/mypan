// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package client

import (
	"context"
	"net/url"
	"strconv"
)

func (client *Client) ListAll(ctx context.Context, dir string, start int) (ListAllResponse, error) {
	dir = client.AbsPath(dir)
	return client.listAll(ctx, dir, start)
}

func (client *Client) ListAllEx(ctx context.Context, dir string) (ListAllResponse, error) {
	dir = client.AbsPath(dir)

	var (
		ret    ListAllResponse
		cursor int
	)
	for {
		resp, err := client.listAll(ctx, dir, cursor)
		if err != nil {
			return ret, err
		}
		ret.List = append(ret.List, resp.List...)
		if resp.HasMore == 0 {
			break
		}
		cursor = resp.Cursor
	}
	return ret, nil
}

func (client *Client) listAll(ctx context.Context, dir string, start int) (ListAllResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
	)
	queryArgs := url.Values{}
	queryArgs.Set("method", "listall")
	queryArgs.Set("access_token", accessAuth.AccessToken)
	queryArgs.Set("path", dir)
	queryArgs.Set("recursion", strconv.Itoa(1))
	queryArgs.Set("start", strconv.Itoa(start))
	queryArgs.Set("limit", "1000")
	queryArgs.Set("web", "0") // returns no thumbnail

	var resp ListAllResponse
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
