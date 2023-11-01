// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

type OauthDeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationUrl string `json:"verification_url"`
	QrcodeUrl       string `json:"qrcode_url"`
	ExpiresIn       int32  `json:"expires_in"`
	Interval        int32  `json:"interval"`
}

type OauthTokenResponse struct {
	ExpiresIn     int32  `json:"expires_in"`
	RefreshToken  string `json:"refresh_token"`
	AccessToken   string `json:"access_token"`
	SessionSecret string `json:"session_secret"`
	SessionKey    string `json:"session_key"`
	Scope         string `json:"scope"`
}

type OauthRefreshTokenResponse OauthTokenResponse

type FilePrecreateResponse struct {
	UploadId   string `json:"uploadid"`
	ReturnType int    `json:"return_type"`
	BlockList  []int  `json:"block_list"`
}
type Superfile2UploadResponse struct {
	Md5 string `json:"md5"`
}

const (
	ONDUP_FAIL      = "fail"
	ONDUP_OVERWRITE = "overwrite"
	ONDUP_NEWCOPY   = "newcopy"
	ONDUP_SKIP      = "skip" // method: filemanager

	RTYPE_NEWCOPY   = 1 // newcopy if path conflict
	RTYPE_NEWCOPY2  = 2 // newcopy if path conflict & blockList differ
	RTYPE_OVERWRITE = 3 // overwrite if path conflict

	CATEGORY_VIDEO   = 1
	CATEGORY_MUSIC   = 2
	CATEGORY_PICTURE = 3
	CATEGORY_DOC     = 4
	CATEGORY_APP     = 5
	CATEGORY_OTHER   = 6
	CATEGORY_SEED    = 7
)

type UploadResponse struct {
	Path  string `json:"path"`
	Size  uint64 `json:"size"`
	Ctime uint64 `json:"ctime"`
	Mtime uint64 `json:"mtime"`
	Md5   string `json:"md5"`
	FsId  uint64 `json:"fs_id"`
}

const (
	// order by file type first, then by name/time/size
	OrderByName = "name"
	OrderByTime = "time"
	OrderBySize = "size"
)

type ListResponse struct {
	Errno    int    `json:"errno"`
	GuidInfo string `json:"guid_info"`
	List     []struct {
		FsId           uint64 `json:"fs_id"`
		Path           string `json:"path"`
		ServerFilename string `json:"server_filename"`
		Size           uint64 `json:"size"`
		ServerCtime    uint64 `json:"server_ctime"`
		ServerMtime    uint64 `json:"server_mtime"`
		LocalCtime     uint64 `json:"local_ctime"`
		LocalMtime     uint64 `json:"local_mtime"`
		IsDir          int    `json:"isdir"`
		Category       int    `json:"category"`
		Md5            string `json:"md5"`
		// Empty indicates whether the directory is empty.
		//
		// NOTE It's not mentioned in the doc
		Empty *int `json:"empty,omitempty"`
		// DirEmpty indicates whether there is a subdir.
		DirEmpty *int              `json:"dir_empty,omitempty"`
		Thumbs   map[string]string `json:"thumbs"`
	} `json:"list"`
}

type ListAllResponse struct {
	HasMore int `json:"has_more"`
	Cursor  int `json:"cursor"`
	List    []struct {
		FsId           uint64            `json:"fs_id"`
		Path           string            `json:"path"`
		ServerFilename string            `json:"server_filename"`
		Size           uint64            `json:"size"`
		ServerCtime    uint64            `json:"server_ctime"`
		ServerMtime    uint64            `json:"server_mtime"`
		LocalCtime     uint64            `json:"local_ctime"`
		LocalMtime     uint64            `json:"local_mtime"`
		IsDir          int               `json:"isdir"`
		Category       int               `json:"category"`
		Md5            string            `json:"md5"`
		Thumbs         map[string]string `json:"thumbs"`
	} `json:"list"`
}

type QuotaResponse struct {
	Total int64 `json:"total"`
	Free  int64 `json:"free"`
	Used  int64 `json:"used"`
	// True if there is free space to be expired in 7 days
	Expire bool `json:"expire"`
}

type UinfoResponse struct {
	// User id
	Uk        int    `json:"uk"`
	AvatarUrl string `json:"avatar_url"`
	// Baidu account name
	BaiduName string `json:"baidu_name"`
	// Baidu netdisk account name
	NetdiskName string `json:"netdisk_name"`
	// 0 (regular user), 1 (regular paid user), 2 (premium paid user)
	VipType int `json:"vip_type"`
}

type FileManagerResponse struct {
	Info []struct {
		Errno int    `json:"errno"`
		Path  string `json:"path"`
	} `json:"info"`
	TaskId uint64 `json:"taskid"`
}

type FileMetaResponse struct {
	DLink string `json:"dlink"`

	Path        string            `json:"path"`
	Filename    string            `json:"filename"`
	Size        uint64            `json:"size"`
	IsDir       int               `json:"isdir"`
	ServerCtime uint64            `json:"server_ctime"`
	ServerMtime uint64            `json:"server_mtime"`
	Category    int               `json:"category"`
	Thumbs      map[string]string `json:"thumbs"`

	// The following fields are present and should be, though not mentioned
	// in the doc
	//
	//   FsId, Md5, LocalCtime, LocalMtime
	FsId       uint64 `json:"fs_id"`
	Md5        string `json:"md5"`
	LocalCtime uint64 `json:"local_ctime"`
	LocalMtime uint64 `json:"local_mtime"`
}

type FileMetasResponse struct {
	List []FileMetaResponse `json:"list"`
}

func DecodeResponse(r io.Reader, v interface{}) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "read")
	}
	return JSONUnmarshalResponse(data, v)
}

func JSONIsAPIError(data []byte) error {
	var apierr APIError
	if err := json.Unmarshal(data, &apierr); err != nil {
		return errors.Wrapf(err, "unmarshal response (trying error): %d: %s", len(data), data)
	}
	if err := apierr.Err(); err != nil {
		return err
	}
	return nil
}

func JSONUnmarshalResponse(data []byte, v interface{}) error {
	if apiErr := JSONIsAPIError(data); apiErr != nil {
		return apiErr
	}
	if err := json.Unmarshal(data, v); err != nil {
		return errors.Wrap(err, "unmarshal response")
	}
	return nil
}
