//go:build tools

/*
Copyright (c) Edgeless Systems GmbH

SPDX-License-Identifier: AGPL-3.0-only
*/

package main

import (
	_ "github.com/katexochen/sh/v3/cmd/shfmt"
	_ "github.com/rhysd/actionlint/cmd/actionlint"
	_ "mvdan.cc/gofumpt"
)
