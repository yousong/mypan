// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package store

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type StoreSerdeI interface {
	Set(key string, val interface{}) error
	Get(key string, val interface{}) error
}

type JSONStore struct {
	store StoreI

	indent string
}

func NewJSONStore(store StoreI) *JSONStore {
	js := &JSONStore{
		store:  store,
		indent: "  ",
	}
	return js
}

func (js *JSONStore) Indent(indent string) *JSONStore {
	js.indent = indent
	return js
}

func (js *JSONStore) Set(key string, val interface{}) error {
	const (
		marshalPrefix = ""
	)
	var (
		data []byte
		err  error
	)
	if indent := js.indent; indent != "" {
		data, err = json.MarshalIndent(val, marshalPrefix, js.indent)
	} else {
		data, err = json.Marshal(val)
	}
	if err != nil {
		return errors.Wrapf(err, "json store set marshal (%s)", key)
	}
	if err := js.store.Set(key, data); err != nil {
		return errors.Wrapf(err, "json store set %s", key)
	}
	return nil
}

func (js *JSONStore) Get(key string, val interface{}) error {
	data, err := js.store.Get(key)
	if err != nil {
		return errors.Wrapf(err, "json store get %s", key)
	}
	if err := json.Unmarshal(data, val); err != nil {
		return errors.Wrapf(err, "json store get unmarshal (%s)", key)
	}
	return nil
}
