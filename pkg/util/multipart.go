// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package util

import (
	"bytes"
	"io"
	"mime/multipart"
)

type FormField struct {
	Name   string
	Reader io.Reader
}

type FormFile struct {
	Name     string
	Filename string
	Reader   io.Reader
}

type MultipartFormBodyChunked struct {
	pr  io.ReadCloser
	mpw *multipart.Writer
}

func NewMultipartFormFilesBodyChunked(
	formFiles ...FormFile,
) *MultipartFormBodyChunked {
	return NewMultipartFormBodyChunked(nil, formFiles)
}

func NewMultipartFormBodyChunked(
	formFields []FormField,
	formFiles []FormFile,
) *MultipartFormBodyChunked {
	pr, pw := io.Pipe()
	mpw := multipart.NewWriter(pw)
	mfb := &MultipartFormBodyChunked{
		pr:  pr,
		mpw: mpw,
	}
	go func() {
		var closeError error
		defer func() {
			pw.CloseWithError(closeError)
		}()

		for _, formField := range formFields {
			w, err := mpw.CreateFormField(formField.Name)
			if err != nil {
				closeError = err
				return
			}
			if _, err := io.Copy(w, formField.Reader); err != nil {
				closeError = err
				return
			}
		}
		for _, formFile := range formFiles {
			w, err := mpw.CreateFormFile(formFile.Name, formFile.Filename)
			if err != nil {
				closeError = err
				return
			}
			if _, err := io.Copy(w, formFile.Reader); err != nil {
				closeError = err
				return
			}
		}
		if err := mpw.Close(); err != nil {
			closeError = err
			return
		}
	}()
	return mfb
}

func (mfb *MultipartFormBodyChunked) FormDataContentType() string {
	return mfb.mpw.FormDataContentType()
}

func (mfb *MultipartFormBodyChunked) ReadCloser() io.ReadCloser {
	return mfb.pr
}

func (mfb *MultipartFormBodyChunked) Read(p []byte) (int, error) {
	return mfb.pr.Read(p)
}

func (mfb *MultipartFormBodyChunked) Close() error {
	return mfb.pr.Close()
}

type MultipartFormBody struct {
	r   io.Reader
	mpw *multipart.Writer
}

func NewMultipartFormFilesBody(
	formFiles ...FormFile,
) (*MultipartFormBody, error) {
	return NewMultipartFormBody(nil, formFiles)
}

func NewMultipartFormBody(
	formFields []FormField,
	formFiles []FormFile,
) (*MultipartFormBody, error) {
	buf := &bytes.Buffer{}
	mpw := multipart.NewWriter(buf)
	mfb := &MultipartFormBody{
		r:   buf,
		mpw: mpw,
	}
	for _, formField := range formFields {
		w, err := mpw.CreateFormField(formField.Name)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(w, formField.Reader); err != nil {
			return nil, err
		}
	}
	for _, formFile := range formFiles {
		w, err := mpw.CreateFormFile(formFile.Name, formFile.Filename)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(w, formFile.Reader); err != nil {
			return nil, err
		}
	}
	if err := mpw.Close(); err != nil {
		return nil, err
	}
	return mfb, nil
}

func (mfb *MultipartFormBody) FormDataContentType() string {
	return mfb.mpw.FormDataContentType()
}

func (mfb *MultipartFormBody) Reader() io.Reader {
	return mfb.r
}

func (mfb *MultipartFormBody) Read(p []byte) (int, error) {
	return mfb.r.Read(p)
}
