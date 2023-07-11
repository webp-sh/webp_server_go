package helper

import (
	"testing"
	"webp_server_go/config"

	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, int64(5), count)
}

func TestImageExists(t *testing.T) {
	t.Run("file not exists", func(t *testing.T) {
		assert.False(t, ImageExists("dgyuaikdsa"))
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
		assert.False(t, CheckAllowedType("./helper_test.go"))
	})

	t.Run("allowed type", func(t *testing.T) {
		assert.True(t, CheckAllowedType("test.jpg"))
	})
}
