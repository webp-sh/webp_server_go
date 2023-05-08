package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrefetchImages(t *testing.T) {
	fp := "./prefetch"
	_ = os.Mkdir(fp, 0755)
	prefetchImages("./pics/dir1/", "./prefetch")
	count := fileCount("./prefetch")
	assert.Equal(t, int64(1), count)
	_ = os.RemoveAll(fp)
}

func TestBadPrefetch(t *testing.T) {
	jobs = 1
	prefetchImages("./pics2", "./prefetch")
}
