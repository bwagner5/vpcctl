//go:build tools

package main

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/go-licenses/v2"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
