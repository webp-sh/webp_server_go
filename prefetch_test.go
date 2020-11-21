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
	// single thread
	fp := "./prefetch"
	_ = os.Mkdir(fp, 0755)
	prefetchImages("./pics", "./prefetch", "80")
	count := fileCount("./prefetch")
	assert.Equal(t, 6, count)
	_ = os.RemoveAll(fp)

	// concurrency
	jobs = 2
	_ = os.Mkdir(fp, 0755)
	prefetchImages("./pics", "./prefetch", "80")
	count = fileCount("./prefetch")
	assert.Equal(t, 4, count)
	_ = os.RemoveAll(fp)
}
