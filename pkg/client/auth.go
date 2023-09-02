// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"context"
	"net/url"
	"time"

	"mypan/pkg/config"

	"github.com/golang/glog"
)

type AccessAuth struct {
	AccessToken           string
	AccessTokenExpireTime time.Time
	RefreshToken          string
}

func (client *Client) OauthGetDeviceCode(ctx context.Context) (OauthDeviceCodeResponse, error) {
	var (
		cfg  = client.cfg
		resp OauthDeviceCodeResponse
	)
	queryArgs := url.Values{}
	queryArgs.Set("response_type", "device_code")
	queryArgs.Set("client_id", cfg.AppKey)
	queryArgs.Set("scope", SCOPE_BASIC_NETDISK)
	if err := client.doHTTPGetJSON(
		ctx,
		newAuthDeviceCodeAPIURL(),
		queryArgs,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}

func (client *Client) OauthGetToken(ctx context.Context, deviceCode string) (OauthTokenResponse, error) {
	var (
		cfg  = client.cfg
		resp OauthTokenResponse
	)
	queryArgs := url.Values{}
	queryArgs.Set("grant_type", "device_token")
	queryArgs.Set("client_id", cfg.AppKey)
	queryArgs.Set("client_secret", cfg.SecretKey)
	queryArgs.Set("code", deviceCode)
	if err := client.doHTTPGetJSON(
		ctx,
		newAuthTokenAPIURL(),
		queryArgs,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}

func (client *Client) OauthRefreshToken(ctx context.Context) (OauthRefreshTokenResponse, error) {
	var (
		cfg        = client.cfg
		accessAuth = client.GetAccessAuth()
		resp       OauthRefreshTokenResponse
	)
	queryArgs := url.Values{}
	queryArgs.Set("grant_type", "refresh_token")
	queryArgs.Set("client_id", cfg.AppKey)
	queryArgs.Set("client_secret", cfg.SecretKey)
	queryArgs.Set("refresh_token", accessAuth.RefreshToken)
	if err := client.doHTTPGetJSON(
		ctx,
		newAuthTokenAPIURL(),
		queryArgs,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}

func (client *Client) CheckAccessAuth(ctx context.Context) error {
	resp, err := client.Quota(ctx)
	glog.V(config.VerboseOn).Infof("check access auth with quota: %v", resp)
	return err
}

func (client *Client) GetAccessAuth() AccessAuth {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.accessAuth
}

func (client *Client) SetAccessAuth(accessAuth AccessAuth) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.accessAuth = accessAuth
}
