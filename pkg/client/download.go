// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

func (client *Client) Download(ctx context.Context, relpath string) (*http.Response, error) {
	metaResp, err := client.FileMetaByPath(ctx, relpath)
	if err != nil {
		return nil, errors.Wrapf(err, "get meta by relpath %q", relpath)
	}
	return client.DownloadByMeta(ctx, metaResp)
}

func (client *Client) DownloadByFsId(ctx context.Context, fsId uint64) (*http.Response, error) {
	metaResp, err := client.FileMeta(ctx, fsId)
	if err != nil {
		return nil, errors.Wrapf(err, "get meta by fsId %d", fsId)
	}
	return client.DownloadByMeta(ctx, metaResp)
}

func (client *Client) DownloadByMeta(ctx context.Context, meta FileMetaResponse) (*http.Response, error) {
	dlink := meta.DLink
	if dlink == "" {
		return nil, errors.New("empty dlink")
	}
	return client.DownloadByDLink(ctx, dlink)
}

func (client *Client) getReqByDLink(ctx context.Context, method, dlink string) (*http.Request, error) {
	var (
		accessAuth = client.GetAccessAuth()
	)
	dlinkUrl, err := url.Parse(dlink)
	if err != nil {
		return nil, errors.Wrapf(err, "bad dlink %s", dlink)
	}
	dlinkQueryArgs := dlinkUrl.Query()
	dlinkQueryArgs.Set("access_token", accessAuth.AccessToken)
	dlinkUrl.RawQuery = dlinkQueryArgs.Encode()
	dlink = dlinkUrl.String()
	req, err := http.NewRequestWithContext(ctx, method, dlink, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "new http request %s", dlink)
	}
	// The doc mandates value of User-Agent header, but it seems it works without it
	req.Header.Set("User-Agent", "pan.baidu.com")
	return req, nil
}

func (client *Client) DownloadByDLink(ctx context.Context, dlink string, opts ...func(*http.Request)) (*http.Response, error) {
	httpReq, err := client.getReqByDLink(ctx, http.MethodGet, dlink)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(httpReq)
	}
	httpResp, err := client.doHTTPReq(ctx, httpReq)
	if err != nil {
		return nil, errors.Wrapf(err, "http get %s", dlink)
	}
	if xlt := xloadTracker(ctx); xlt != nil {
		var (
			contentLen = httpResp.ContentLength
			total      = contentLen
			done       = int64(0)
		)
		if ranges := httpResp.Header.Get("content-range"); ranges != "" {
			var (
				unit  string
				start int64
				end   int64
			)
			n, err := fmt.Sscanf(ranges, "%s %d-%d/%d", &unit, &start, &end, &total)
			if err != nil || n != 4 || unit != "bytes" || (end-start+1) != contentLen || total < contentLen {
				httpResp.Body.Close()
				return nil, fmt.Errorf("parse content-range %q: %v", ranges, err)
			}
			done = total - contentLen
		}
		httpResp.Body = newReadCloseTrackerWithCtx(ctx, httpResp.Body, total, done)
	}
	return httpResp, nil
}

func (client *Client) HeadByDLink(ctx context.Context, dlink string) (*http.Response, error) {
	httpReq, err := client.getReqByDLink(ctx, http.MethodHead, dlink)
	if err != nil {
		return nil, err
	}
	httpResp, err := client.doHTTPReq(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	return httpResp, nil
}
