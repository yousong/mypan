// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"os"
	"strconv"
	"syscall"

	"mypan/pkg/util"

	"github.com/pkg/errors"
)

const (
	// API limit on single upload
	MAX_SIZE_SINGLE_UPLOAD = 2 * GiB

	// Client limit
	MIN_SIZE_MULTIPART_UPLOAD = 5 * MiB
)

type statOpt struct {
	Size  int64
	Ctime int64
	Mtime int64
}

func seekStart(f *os.File) error {
	_, err := f.Seek(0, 0)
	return err
}

func (client *Client) Upload(
	ctx context.Context,
	src, dst string,
) (UploadResponse, error) {
	var resp UploadResponse

	dst = client.AbsPath(dst)
	f, err := os.Open(src)
	if err != nil {
		return resp, err
	}
	fi, err := f.Stat()
	if err != nil {
		return resp, err
	}
	statopt := statOpt{
		Size:  fi.Size(),
		Mtime: fi.ModTime().Unix(),
	}
	switch fiSys := fi.Sys().(type) {
	case syscall.Stat_t:
		statopt.Ctime = fiSys.Ctim.Sec
	default:
		client.vlog().Infof("unexpected stat type: %T", fiSys)
	}
	if statopt.Size < MIN_SIZE_MULTIPART_UPLOAD {
		return client.uploadSingle(ctx, f, dst)
	} else {
		return client.uploadMultipart(ctx, f, statopt, dst)
	}
}

func (client *Client) uploadSingle(
	ctx context.Context,
	f *os.File,
	dst string,
) (UploadResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
		resp       UploadResponse
	)
	queryArgs := url.Values{}
	queryArgs.Set("method", "upload")
	queryArgs.Set("path", dst)
	queryArgs.Set("ondup", ONDUP_OVERWRITE)
	queryArgs.Set("access_token", accessAuth.AccessToken)

	body, _ := util.NewMultipartFormFilesBody(util.FormFile{
		Name:     "file",
		Filename: f.Name(),
		Reader:   f,
	})
	bodyReader := body.Reader()
	bodyReader, xlt := newReadTrackerWithCtx(ctx, bodyReader)
	if xlt != nil {
		defer xlt.Done()
	}
	//body := util.NewMultipartFormFilesBodyChunked(util.FormFile{
	//        Name:     "file",
	//        Filename: f.Name(),
	//        Reader:   f,
	//})
	//bodyReader := body
	if err := client.doHTTPPostJSON(
		ctx,
		newSingleUploadAPIURL(),
		queryArgs,
		bodyReader,
		body.FormDataContentType(),
		&resp,
	); err != nil {
		return resp, err
	}
	client.vlog().Infof("single upload %v", resp)
	return resp, nil
}

func (client *Client) uploadMultipart(
	ctx context.Context,
	f *os.File,
	statopt statOpt,
	dst string,
) (UploadResponse, error) {
	var ret UploadResponse

	// make md5 blockList
	blockList, err := computeReaderBlockList(f)
	if err != nil {
		return ret, errors.Wrap(err, "compute block list")
	}
	if err := seekStart(f); err != nil {
		return ret, errors.Wrap(err, "file seek")
	}
	blockListData := string(util.MustMarshalJSON(blockList))

	// precreate
	precreateResp, err := client.uploadPrecreate(ctx, dst, statopt, blockListData)
	if err != nil {
		return ret, errors.Wrapf(err, "precreate %q", dst)
	}

	// upload parts
	var (
		uploadId        = precreateResp.UploadId
		blockListIndice = precreateResp.BlockList
	)
	for _, partSeq := range blockListIndice {
		f.Seek(UPLOAD_API_BLOCK_SIZE*int64(partSeq), 0)
		r := io.LimitReader(f, UPLOAD_API_BLOCK_SIZE)
		mffb, _ := util.NewMultipartFormFilesBody(util.FormFile{
			Name:     "file",
			Filename: f.Name(),
			Reader:   r,
		})
		bodyReader := mffb.Reader()
		bodyReader, xlt := newReadTrackerWithCtx(ctx, bodyReader)
		if xlt != nil {
			defer xlt.Done()
		}
		contentType := mffb.FormDataContentType()
		resp, err := client.uploadSuperfile2(ctx, dst, uploadId, partSeq, bodyReader, contentType)
		if err != nil {
			return ret, errors.Wrapf(err, "upload %q (%d)", dst, partSeq)
		}
		client.vlog().Infof("upload %s (%d): %s", dst, partSeq, util.MustMarshalJSON(resp))
	}

	// combine parts
	ret, err = client.uploadCreate(ctx, dst, statopt, uploadId, blockListData)
	if err != nil {
		return ret, errors.Wrapf(err, "file create %q", dst)
	}
	return ret, nil
}

func (client *Client) uploadPrecreate(
	ctx context.Context,
	dst string,
	statopt statOpt,
	blockListData string,
) (FilePrecreateResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
		resp       FilePrecreateResponse
	)

	queryArgs := url.Values{}
	queryArgs.Set("method", "precreate")
	queryArgs.Set("access_token", accessAuth.AccessToken)

	bodyArgs := url.Values{}
	bodyArgs.Set("path", dst)
	bodyArgs.Set("isdir", "0")
	bodyArgs.Set("autoinit", "1")
	bodyArgs.Set("size", strconv.FormatInt(statopt.Size, 10))
	bodyArgs.Set("block_list", blockListData)
	bodyArgs.Set("rtype", strconv.Itoa(RTYPE_OVERWRITE))
	body := bytes.NewBufferString(bodyArgs.Encode())
	if err := client.doHTTPPostFormJSON(
		ctx,
		newFileAPIURL(),
		queryArgs,
		body,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}

func (client *Client) uploadSuperfile2(
	ctx context.Context,
	dst string,
	uploadId string,
	partSeq int,
	body io.Reader,
	contentType string,
) (Superfile2UploadResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
		resp       Superfile2UploadResponse
	)

	queryArgs := url.Values{}
	queryArgs.Set("method", "upload")
	queryArgs.Set("access_token", accessAuth.AccessToken)
	queryArgs.Set("partseq", strconv.Itoa(partSeq))
	queryArgs.Set("uploadid", uploadId)
	queryArgs.Set("path", dst)
	queryArgs.Set("type", "tmpfile")

	if err := client.doHTTPPostJSON(
		ctx,
		newSuperfile2APIURL(),
		queryArgs,
		body,
		contentType,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}

func (client *Client) uploadCreate(
	ctx context.Context,
	dst string,
	statopt statOpt,
	uploadId string,
	blockListData string,
) (UploadResponse, error) {
	var (
		accessAuth = client.GetAccessAuth()
		resp       UploadResponse
	)

	queryArgs := url.Values{}
	queryArgs.Set("method", "create")
	queryArgs.Set("access_token", accessAuth.AccessToken)

	bodyArgs := url.Values{}
	bodyArgs.Set("path", dst)
	bodyArgs.Set("isdir", "0")
	bodyArgs.Set("uploadid", uploadId)
	bodyArgs.Set("block_list", blockListData)
	bodyArgs.Set("rtype", strconv.Itoa(RTYPE_OVERWRITE))
	bodyArgs.Set("size", strconv.FormatInt(statopt.Size, 10))
	bodyArgs.Set("local_ctime", strconv.FormatInt(statopt.Ctime, 10))
	bodyArgs.Set("local_mtime", strconv.FormatInt(statopt.Mtime, 10))
	body := bytes.NewBufferString(bodyArgs.Encode())
	if err := client.doHTTPPostFormJSON(
		ctx,
		newFileAPIURL(),
		queryArgs,
		body,
		&resp,
	); err != nil {
		return resp, err
	}
	return resp, nil
}
