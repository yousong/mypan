// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"sync"

	"mypan/pkg/config"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type Config struct {
	AppID     int64
	AppKey    string
	SecretKey string

	AccessAuth AccessAuth

	AppBaseDir string
}

type Client struct {
	cfg Config

	httpclient *http.Client

	mu         *sync.Mutex
	accessAuth AccessAuth
}

func New(cfg Config) *Client {
	if !strings.HasSuffix(cfg.AppBaseDir, "/") {
		cfg.AppBaseDir += "/"
	}

	client := &Client{
		cfg: cfg,

		accessAuth: cfg.AccessAuth,
		httpclient: &http.Client{},

		mu: &sync.Mutex{},
	}
	return client
}

func (client *Client) AbsPath(relpath string) string {
	if relpath == "" || relpath[0] != '/' {
		return path.Join(client.cfg.AppBaseDir, relpath)
	}
	return relpath
}

func (client *Client) RelPath(abspath string) string {
	return strings.TrimPrefix(abspath, client.cfg.AppBaseDir)
}

func (client *Client) doHTTPReqJSON(ctx context.Context, req *http.Request, v interface{}) error {
	resp, err := client.doHTTPReq(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := DecodeResponse(resp.Body, v); err != nil {
		return err
	}
	return nil
}

func (client *Client) doHTTPReqBytes(ctx context.Context, req *http.Request) ([]byte, error) {
	resp, err := client.doHTTPReq(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return data, err
	}
	if apiErr := JSONIsAPIError(data); apiErr != nil {
		return data, apiErr
	}
	return data, nil
}

func (client *Client) doHTTPGetJSON(
	ctx context.Context,
	apiURL *url.URL,
	queryArgs url.Values,
	v interface{},
) error {
	return client.doHTTPJSON(ctx, http.MethodGet, apiURL, queryArgs, nil, nil, v)
}

func (client *Client) doHTTPPostFormJSON(
	ctx context.Context,
	apiURL *url.URL,
	queryArgs url.Values,
	body io.Reader,
	v interface{},
) error {
	return client.doHTTPPostJSON(ctx, apiURL, queryArgs, body, ContentTypeFormUrlEncoded, v)
}
func (client *Client) doHTTPPostJSON(
	ctx context.Context,
	apiURL *url.URL,
	queryArgs url.Values,
	body io.Reader,
	contentType string,
	v interface{},
) error {
	reqOpt := func(req *http.Request) {
		req.Header.Set("Content-Type", contentType)
	}
	return client.doHTTPJSON(ctx, http.MethodPost, apiURL, queryArgs, body, reqOpt, v)
}

func (client *Client) doHTTPJSON(
	ctx context.Context,
	method string,
	apiURL *url.URL,
	queryArgs url.Values,
	body io.Reader,
	reqOpt func(req *http.Request),
	v interface{},
) error {
	if queryArgs != nil {
		apiURL.RawQuery = queryArgs.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, apiURL.String(), body)
	if err != nil {
		return err
	}
	if req.ContentLength <= 0 {
		if l, ok := body.(lenI); ok {
			req.ContentLength = int64(l.Len())
		}
	}
	if reqOpt != nil {
		reqOpt(req)
	}
	if err := client.doHTTPReqJSON(ctx, req, v); err != nil {
		return err
	}
	return nil
}

func (client *Client) doHTTPReq(ctx context.Context, req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", APP_USER_AGENT)
	}
	if glog.V(config.VerboseHTTP) {
		dumpBody := bool(glog.V(config.VerboseHTTPBody))
		d, err := httputil.DumpRequestOut(req, dumpBody)
		if err == nil {
			glog.Infof("%s", d)
		} else {
			return nil, errors.Wrap(err, "dump request")
		}
	}
	resp, err := client.httpclient.Do(req)
	if err != nil {
		return resp, err
	}
	if glog.V(config.VerboseHTTP) {
		dumpBody := bool(glog.V(config.VerboseHTTPBody))
		d, err := httputil.DumpResponse(resp, dumpBody)
		if err == nil {
			glog.Infof("%s", d)
		} else {
			return nil, errors.Wrap(err, "dump response")
		}
	}
	return resp, err
}

func (client *Client) vlog() glog.Verbose {
	return glog.V(config.VerboseOn)
}

type ClientI interface {
	AbsPath(relpath string) string
	RelPath(abspath string) string

	// Check and auth
	OauthGetDeviceCode(ctx context.Context) (OauthDeviceCodeResponse, error)
	OauthGetToken(ctx context.Context, deviceCode string) (OauthTokenResponse, error)
	OauthRefreshToken(ctx context.Context) (OauthRefreshTokenResponse, error)
	GetAccessAuth() AccessAuth
	SetAccessAuth(accessAuth AccessAuth)
	CheckAccessAuth(ctx context.Context) error

	UInfo(ctx context.Context) (UinfoResponse, error)
	Quota(ctx context.Context) (QuotaResponse, error)

	FileMetaByPath(ctx context.Context, relpath string) (FileMetaResponse, error)
	FileMetasByPath(ctx context.Context, relpaths []string) (FileMetasResponse, error)
	FileMeta(ctx context.Context, fsId uint64) (FileMetaResponse, error)
	FileMetas(ctx context.Context, fsIds []uint64) (FileMetasResponse, error)

	Download(ctx context.Context, relpath string) (*http.Response, error)
	DownloadByMeta(ctx context.Context, meta FileMetaResponse) (*http.Response, error)
	DownloadByFsId(ctx context.Context, fsId uint64) (*http.Response, error)
	DownloadByDLink(ctx context.Context, dlink string, opts ...func(*http.Request)) (*http.Response, error)

	Upload(
		ctx context.Context,
		src, dst string,
	) (UploadResponse, error)

	List(ctx context.Context, dir string, start int) (ListResponse, error)
	ListEx(ctx context.Context, dir string) (ListResponse, error)
	ListAll(ctx context.Context, dir string, start int) (ListAllResponse, error)
	ListAllEx(ctx context.Context, dir string) (ListAllResponse, error)

	Delete(
		ctx context.Context,
		file string,
	) (FileManagerResponse, error)
	DeleteMulti(
		ctx context.Context,
		fileList []string,
	) (FileManagerResponse, error)
	Rename(
		ctx context.Context,
		path, newname string,
	) (FileManagerResponse, error)
	Copy(
		ctx context.Context,
		path, dest string,
	) (FileManagerResponse, error)
	Move(
		ctx context.Context,
		path, dest string,
	) (FileManagerResponse, error)
}
