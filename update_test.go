// webp_server_go - update_test
// 2020-11-10 09:36
// Benny <benny.think@gmail.com>

package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestNormalAutoUpdate(t *testing.T) {
	version = "0.0.1"
	dir := "./update"
	autoUpdate()
	assert.NotEqual(t, 0, fileCount(dir))
	_ = os.RemoveAll(dir)
}

func TestNoNeedAutoUpdate(t *testing.T) {
	version = "99.99"
	dir := "./update"
	autoUpdate()
	info, _ := os.Stat(dir)
	assert.Nil(t, info)
}
