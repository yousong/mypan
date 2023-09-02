// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package client

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type APIError struct {
	CodeInt   int
	CodeStr   string
	Message   string
	RequestId string

	IsError  bool
	Original string
}

func (err *APIError) Err() error {
	if err.IsError {
		return err
	}
	return nil
}

func (err *APIError) Error() string {
	return err.Original
}

func (err *APIError) UnmarshalJSON(data []byte) error {
	type Error struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`

		Errno  int    `json:"errno"`
		Errmsg string `json:"errmsg"`

		ErrorCode int         `json:"error_code"`
		ErrorMsg  string      `json:"error_msg"`
		RequestId interface{} `json:"request_id,omitempty"`
	}

	var e Error
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}

	if code := e.Errno; code != 0 {
		err.CodeInt = code
	} else if code := e.ErrorCode; code != 0 {
		err.CodeInt = code
	} else if e.Error != "" {
		err.CodeStr = e.Error
	}
	isError := err.CodeInt != 0 || err.CodeStr != ""

	if msg := e.ErrorDescription; msg != "" {
		err.Message = msg
	} else if msg := e.Errmsg; msg != "" {
		err.Message = msg
	}

	switch reqId := e.RequestId.(type) {
	case float64:
		err.RequestId = fmt.Sprintf("%d", int64(reqId))
	case string:
		err.RequestId = reqId
	}

	if isError {
		err.IsError = true
		err.Original = string(data)
	}
	return nil
}

var errNotFound = fmt.Errorf("not found")

func ErrIsNotExist(err error) bool {
	cause := errors.Cause(err)
	if aee, ok := cause.(*APIError); ok {
		// method=list
		//
		// 	{"errno":-9,"request_id":165361853887776084}
		if aee.CodeInt == -9 {
			return true
		}
		// method=listall
		//
		// 	{"errmsg":"file does not exist","errno":31066,"request_id":"346199097945924836"}
		if aee.CodeInt == 31066 {
			return true
		}
	}
	if cause == errNotFound {
		return true
	}
	return false
}
