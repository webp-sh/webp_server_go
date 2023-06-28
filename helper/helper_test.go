// webp_server_go - helper_test.go
// 2023-06-28 19:22
// Benny <benny.think@gmail.com>

package helper

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"webp_server_go/config"
)

func TestMain(m *testing.M) {
	config.ConfigPath = "../config.json"
	config.LoadConfig()
	m.Run()
	config.ConfigPath = "config.json"

}

func TestFileCount(t *testing.T) {
	// test helper dir
	count := FileCount("./")
	assert.Equal(t, int64(3), count)
}

func TestImageExists(t *testing.T) {
	t.Run("file not exists", func(t *testing.T) {
		assert.False(t, ImageExists("dgyuaikdsa"))
	})

	t.Run("file size incorrect", func(t *testing.T) {
		assert.False(t, ImageExists("test.txt"))
	})

	// TODO: how to test lock?

	t.Run("test dir", func(t *testing.T) {
		assert.False(t, ImageExists("/tmp"))
	})

	t.Run("test file", func(t *testing.T) {
		assert.True(t, ImageExists("./helper_test.go"))
	})
}

func TestCheckAllowedType(t *testing.T) {
	t.Run("not allowed type", func(t *testing.T) {
		assert.False(t, CheckAllowedType("test.txt"))
	})

	t.Run("allowed type", func(t *testing.T) {
		assert.True(t, CheckAllowedType("test.jpg"))
	})
}
