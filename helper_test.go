package main

import (
	"testing"
)

// test this file: go test helper_test.go helper.go -v
// test one function: go test -run TestGetFileContentType helper_test.go helper.go -v
func TestGetFileContentType(t *testing.T) {
	var zero = []byte("hello")
	r := GetFileContentType(zero)
	if r != "text/plain; charset=utf-8" {
		t.Errorf("Test error for %s", t.Name())
	}

}
