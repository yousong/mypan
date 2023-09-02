// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package store

import (
	"io/fs"
	"io/ioutil"
	"path"

	"mypan/pkg/util"

	"github.com/pkg/errors"
)

type StoreI interface {
	Set(key string, data []byte) error
	Get(key string) ([]byte, error)
}

type DirStore struct {
	dir  string
	mode fs.FileMode
}

func NewDirStore(dir string) (*DirStore, error) {
	if err := util.MkdirAll(dir); err != nil {
		return nil, errors.Wrapf(err, "new dir store")
	}
	store := &DirStore{
		dir:  dir,
		mode: fs.FileMode(0644),
	}
	return store, nil
}

func (ds *DirStore) Set(key string, data []byte) error {
	filename := path.Join(ds.dir, key)
	err := ioutil.WriteFile(filename, data, ds.mode)
	return err
}

func (ds *DirStore) Get(key string) ([]byte, error) {
	filename := path.Join(ds.dir, key)
	data, err := ioutil.ReadFile(filename)
	return data, err
}
