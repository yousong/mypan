// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou

package main

import (
	"context"
	"fmt"
	"time"

	"mypan/pkg/client"
	"mypan/pkg/config"
	"mypan/pkg/store"
	"mypan/pkg/util"

	"github.com/golang/glog"
	"github.com/pkg/errors"
)

type AuthMan struct {
	client client.ClientI
	store  store.StoreSerdeI
}

func NewAuthMan(client client.ClientI, store store.StoreSerdeI) *AuthMan {
	am := &AuthMan{
		client: client,
		store:  store,
	}
	return am
}

func (am *AuthMan) Auth(ctx context.Context) error {
	accessAuth, err := am.getAccessAuth()
	if err != nil {
		glog.V(config.VerboseOn).Infof("store get: %v", err)
	}
	if accessAuth.AccessToken == "" {
		deviceCode, interval, deadline, err := am.promptDeviceCodeAuth(ctx)
		if err != nil {
			return errors.Wrapf(err, "prepare device code auth")
		}
		accessAuth, err := am.checkDeviceCodeAuthComplete(ctx, deviceCode, interval, deadline)
		if err != nil {
			return errors.Wrapf(err, "check device code auth")
		}
		if err := am.setAccessAuth(accessAuth); err != nil {
			return errors.Wrapf(err, "store set")
		}
	}
	if err := am.client.CheckAccessAuth(ctx); err != nil {
		return errors.Wrap(err, "check access auth")
	}
	glog.Infof("auth ok")
	return nil
}

func (am *AuthMan) Refresh(ctx context.Context) error {
	resp, err := am.client.OauthRefreshToken(ctx)
	if err != nil {
		return err
	}
	expires := time.Duration(resp.ExpiresIn) * time.Second
	accessAuth := client.AccessAuth{
		AccessToken:           resp.AccessToken,
		AccessTokenExpireTime: time.Now().Add(expires),
		RefreshToken:          resp.RefreshToken,
	}
	if err := am.setAccessAuth(accessAuth); err != nil {
		return err
	}
	glog.Infof("auth refresh ok")
	return nil
}

func (am *AuthMan) RefreshAccessTokenLoop(ctx context.Context) {
	for {
		var (
			accessAuth = am.client.GetAccessAuth()
			expireTime = accessAuth.AccessTokenExpireTime
			now        = time.Now()
		)
		if now.After(expireTime) {
			glog.Errorf("refresh token: already expired (%s)", expireTime)
			return
		}
		var (
			timeAvail     = expireTime.Sub(now)
			nextCheckWait time.Duration
		)
		if timeAvail < 24*time.Hour {
			nextCheckWait = 7 * time.Minute
		} else if timeAvail < 3*24*time.Hour {
			nextCheckWait = 60 * time.Minute
		} else {
			nextCheckWait = 24 * time.Hour
		}
		glog.Infof("refresh token: next refresh after %s", nextCheckWait)
		select {
		case <-time.After(nextCheckWait):
		case <-ctx.Done():
			glog.Errorf("refresh token: ctx done: %v", ctx.Err())
			return
		}

		if err := am.Refresh(ctx); err != nil {
			glog.Errorf("refresh token: %v", err)
		}
	}
}

func (am *AuthMan) promptDeviceCodeAuth(ctx context.Context) (
	deviceCode string,
	interval time.Duration,
	deadline time.Time,
	err error,
) {
	resp, err := am.client.OauthGetDeviceCode(ctx)
	if err != nil {
		return
	}
	deviceCode = resp.DeviceCode
	if deviceCode == "" {
		err = fmt.Errorf("device code is empty")
		return
	}

	fmt.Printf("Device code auth: %s\n", util.MustMarshalJSON(resp))
	fmt.Printf("Open this URL and scan qrcode there: %s\n", resp.QrcodeUrl)
	fmt.Printf(" or, open this URL: %s\n", resp.VerificationUrl)
	fmt.Printf("     fill in code: %s\n", resp.UserCode)

	interval = 17 * time.Second
	if resp.Interval != 0 {
		interval = time.Duration(resp.Interval) * time.Second
	}
	expires := 3 * time.Minute
	if resp.ExpiresIn != 0 {
		expires = time.Duration(resp.ExpiresIn) * time.Second
	}
	deadline = time.Now().Add(expires)
	glog.Infof("expires in %s (%s)", expires, deadline)
	glog.Infof("check interval %s", interval)
	return deviceCode, interval, deadline, nil
}

func (am *AuthMan) checkDeviceCodeAuthComplete(
	ctx context.Context,
	deviceCode string,
	interval time.Duration,
	deadline time.Time,
) (client.AccessAuth, error) {
	for {
		if time.Now().After(deadline) {
			return client.AccessAuth{}, fmt.Errorf("device code expired")
		}

		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return client.AccessAuth{}, ctx.Err()
		}
		resp, err := am.client.OauthGetToken(ctx, deviceCode)
		if err != nil {
			glog.Warningf("get access token with device code: %v", err)
			continue
		}

		accessToken := resp.AccessToken
		refreshToken := resp.RefreshToken
		expires := time.Duration(resp.ExpiresIn) * time.Second

		glog.V(config.VerboseOn).Infof("get access token: %s\n", util.MustMarshalJSON(resp))
		glog.V(config.VerboseOn).Infof("    access token: %s\n", accessToken)
		glog.V(config.VerboseOn).Infof("    expires in %s\n", expires)
		glog.V(config.VerboseOn).Infof("    refresh token: %s\n", refreshToken)
		accessAuth := client.AccessAuth{
			AccessToken:           accessToken,
			AccessTokenExpireTime: time.Now().Add(expires),
			RefreshToken:          refreshToken,
		}
		return accessAuth, nil
	}
}

func (am *AuthMan) getAccessAuth() (accessAuth client.AccessAuth, err error) {
	err = am.store.Get(config.StoreKeyAccessAuth, &accessAuth)
	return
}

func (am *AuthMan) setAccessAuth(accessAuth client.AccessAuth) (err error) {
	err = am.store.Set(config.StoreKeyAccessAuth, accessAuth)
	if err == nil {
		am.client.SetAccessAuth(accessAuth)
	}
	return
}
