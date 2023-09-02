// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package store

import (
	"os"
	"reflect"

	"github.com/pkg/errors"
)

type CacheEntry interface {
	Key() string
}

type NewCacheEntryFunc func() CacheEntry

type FileCacheStore struct {
	fileKey   string
	jsonStore StoreSerdeI
	newFunc   NewCacheEntryFunc

	m map[string]CacheEntry
}

func NewFileCacheStore(
	fileKey string,
	jsonStore StoreSerdeI,
	newFunc NewCacheEntryFunc,
) (*FileCacheStore, error) {
	// TODO lock
	fcs := &FileCacheStore{
		fileKey:   fileKey,
		jsonStore: jsonStore,
		newFunc:   newFunc,

		m: map[string]CacheEntry{},
	}
	if err := fcs.load(); err != nil {
		cause := errors.Cause(err)
		if !os.IsNotExist(cause) {
			return nil, err
		}
	}
	return fcs, nil
}

func (fcs *FileCacheStore) Get(key string) (CacheEntry, bool) {
	ce, ok := fcs.m[key]
	return ce, ok
}

func (fcs *FileCacheStore) Set(ce CacheEntry) error {
	fcs.m[ce.Key()] = ce
	return fcs.dump()
}

func (fcs *FileCacheStore) load() error {
	var (
		ce      = fcs.newFunc()
		strType = reflect.TypeOf("")
		ceType  = reflect.TypeOf(ce)
		mType   = reflect.MapOf(strType, ceType)
		mVal    = reflect.New(mType)
		m       = mVal.Interface()
	)

	err := fcs.jsonStore.Get(fcs.fileKey, m)
	if err != nil {
		return err
	}
	for iter := mVal.Elem().MapRange(); iter.Next(); {
		k := iter.Key().Interface().(string)
		v := iter.Value().Interface().(CacheEntry)
		fcs.m[k] = v
	}
	return err
}

func (fcs *FileCacheStore) dump() error {
	err := fcs.jsonStore.Set(fcs.fileKey, fcs.m)
	return err
}
