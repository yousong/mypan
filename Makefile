# SPDX-License-Identifier: MIT
# Copyright (c) 2023 Yousong Zhou

binary:
	go build ./cmd/mypan

test:
	go test -v ./pkg/...

.PHONY: binary
.PHONY: test
