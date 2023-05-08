package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalAutoUpdate(t *testing.T) {
	version = "0.0.1"
	dir := "./update"
	autoUpdate()
	assert.NotEqual(t, 0, fileCount(dir))
	_ = os.RemoveAll(dir)
}

func Test404AutoUpdate(t *testing.T) {
	version = "0.0.1"
	dir := "./update"
	releaseURL = releaseURL + "a"
	autoUpdate()
	assert.Equal(t, int64(0), fileCount(dir))
	_ = os.RemoveAll(dir)
}

func TestNoNeedAutoUpdate(t *testing.T) {
	version = "99.99"
	dir := "./update"
	autoUpdate()
	info, _ := os.Stat(dir)
	assert.Nil(t, info)
}
