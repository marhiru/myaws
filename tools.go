// +build tools

package tools

import (
	_ "github.com/golang/lint/golint"      // executable dependency for development
	_ "github.com/gordonklaus/ineffassign" // executable dependency for development
)
