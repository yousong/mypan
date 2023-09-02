// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import "net/url"

const (
	SchemeHTTPS         = "https"
	HostOpenAPIBaiduCom = "openapi.baidu.com"
	HostPanBaiduCom     = "pan.baidu.com"
	HostDPcsBaiduCom    = "d.pcs.baidu.com"
)

func newSingleUploadAPIURL() *url.URL {
	u := &url.URL{
		Scheme: SchemeHTTPS,
		Host:   HostDPcsBaiduCom,
		Path:   "/rest/2.0/pcs/file",
	}
	return u
}

func newSuperfile2APIURL() *url.URL {
	u := &url.URL{
		Scheme: SchemeHTTPS,
		Host:   HostDPcsBaiduCom,
		Path:   "/rest/2.0/pcs/superfile2",
	}
	return u
}

func newFileAPIURL() *url.URL {
	u := &url.URL{
		Scheme: SchemeHTTPS,
		Host:   HostPanBaiduCom,
		Path:   "/rest/2.0/xpan/file",
	}
	return u
}

func newMultimediaAPIURL() *url.URL {
	u := &url.URL{
		Scheme: SchemeHTTPS,
		Host:   HostPanBaiduCom,
		Path:   "/rest/2.0/xpan/multimedia",
	}
	return u
}

func newQuotaAPIURL() *url.URL {
	u := &url.URL{
		Scheme: SchemeHTTPS,
		Host:   HostPanBaiduCom,
		Path:   "/api/quota",
	}
	return u
}

func newUinfoAPIURL() *url.URL {
	u := &url.URL{
		Scheme: SchemeHTTPS,
		Host:   HostPanBaiduCom,
		Path:   "/rest/2.0/xpan/nas",
	}
	return u
}

func newAuthDeviceCodeAPIURL() *url.URL {
	u := &url.URL{
		Scheme: SchemeHTTPS,
		Host:   HostOpenAPIBaiduCom,
		Path:   "/oauth/2.0/device/code",
	}
	return u
}

func newAuthTokenAPIURL() *url.URL {
	u := &url.URL{
		Scheme: SchemeHTTPS,
		Host:   HostOpenAPIBaiduCom,
		Path:   "/oauth/2.0/token",
	}
	return u
}
