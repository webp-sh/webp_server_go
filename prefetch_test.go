// webp_server_go - prefetch_test.go
// 2020-11-10 09:27
// Benny <benny.think@gmail.com>

package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestPrefetchImages(t *testing.T) {
	fp := "./prefetch"
	_ = os.Mkdir(fp, 0755)
	prefetchImages("./pics/dir1/", "./prefetch")
	count := fileCount("./prefetch")
	assert.Equal(t, int64(2), count)
	_ = os.RemoveAll(fp)
}

func TestBadPrefetch(t *testing.T) {
	jobs = 1
	prefetchImages("./pics2", "./prefetch")
}
