// webp_server_go - webp-server_test
// 2020-11-10 09:41
// Benny <benny.think@gmail.com>

package main

import (
	"github.com/stretchr/testify/assert"
	"net"
	"runtime"
	"testing"
	"time"
)

// due to test limit, we can't test for cli param part.

func TestLoadConfig(t *testing.T) {
	c := loadConfig("./config.json")
	assert.Equal(t, "./exhaust", c.ExhaustPath)
	assert.Equal(t, "127.0.0.1", c.Host)
	assert.Equal(t, "3333", c.Port)
	assert.Equal(t, "80", c.Quality)
	assert.Equal(t, "./pics", c.ImgPath)
	assert.Equal(t, []string{"jpg", "png", "jpeg", "bmp"}, c.AllowedTypes)
}

func TestDeferInit(t *testing.T) {
	// test initial value
	assert.Equal(t, "", configPath)
	assert.False(t, prefetch)
	assert.Equal(t, false, dumpSystemd)
	assert.Equal(t, false, dumpConfig)
	assert.False(t, verboseMode)
}

func TestMainFunction(t *testing.T) {
	go main()
	time.Sleep(time.Second * 2)
	// test read config value
	assert.Equal(t, "config.json", configPath)
	assert.False(t, prefetch)
	assert.Equal(t, runtime.NumCPU(), jobs)
	assert.Equal(t, false, dumpSystemd)
	assert.Equal(t, false, dumpConfig)
	assert.False(t, verboseMode)
	// test port
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", "3333"), time.Second*2)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}
