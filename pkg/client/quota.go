// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package client

import (
	"context"
	"net/url"
)

func (client *Client) Quota(ctx context.Context) (QuotaResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
		resp       QuotaResponse
	)
	queryArgs := url.Values{}
	queryArgs.Set("access_token", accessAuth.AccessToken)
	queryArgs.Set("checkfree", "1")
	queryArgs.Set("checkexpire", "1")
	if err := client.doHTTPGetJSON(
		ctx,
		newQuotaAPIURL(),
		queryArgs,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}
