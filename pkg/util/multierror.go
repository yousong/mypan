// SPDX-License-Identifier: MIT
// Copyright (c) 2023 Yousong Zhou
package util

type MultiError []error

func NewMultiError(errs ...error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return MultiError(errs)
}

func (me MultiError) Error() string {
	var strs []string
	for _, err := range me {
		strs = append(strs, err.Error())
	}
	return string(MustMarshalJSON(strs))
}
