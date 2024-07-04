// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package client

import (
	"context"
	"net/url"
)

func (client *Client) UInfo(ctx context.Context) (UinfoResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
		resp       UinfoResponse
	)
	queryArgs := url.Values{}
	queryArgs.Set("method", "uinfo")
	queryArgs.Set("access_token", accessAuth.AccessToken)
	if err := client.doHTTPGetJSON(
		ctx,
		newUinfoAPIURL(),
		queryArgs,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}
