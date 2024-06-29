package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	ConfigPath = "../config.json"
	m.Run()
	ConfigPath = "config.json"
	Config.ImgPath = "./pics"
}

func TestLoadConfig(t *testing.T) {
	LoadConfig()
	assert.Equal(t, Config.Host, "127.0.0.1")
	assert.Equal(t, Config.Port, "3333")
	assert.Equal(t, Config.Quality, 80)
	assert.Equal(t, Config.ImgPath, "./pics")
	assert.Equal(t, Config.ImageMap, map[string]string{})
	assert.Equal(t, Config.ExhaustPath, "./exhaust")
	assert.Equal(t, Config.CacheTTL, 259200)
	assert.Equal(t, Config.MaxCacheSize, 0)
}

func TestSwitchProxyMode(t *testing.T) {
	switchProxyMode()
	assert.False(t, ProxyMode)
	Config.ImgPath = "https://picsum.photos"
	switchProxyMode()
	assert.True(t, ProxyMode)
}

func TestParseImgMap(t *testing.T) {
	empty := map[string]string{}
	good := map[string]string{
		"/1":                  "../pics/dir1",
		"http://example.com":  "../pics",
		"https://example.com": "../pics",
	}
	bad := map[string]string{
		"1":                   "../pics/dir1",
		"httpx://example.com": "../pics",
		"ftp://example.com":   "../pics",
	}

	assert.Equal(t, empty, parseImgMap(empty))
	assert.Equal(t, empty, parseImgMap(bad))
	assert.Equal(t, good, parseImgMap(good))

	for k, v := range good {
		bad[k] = v
	}
	assert.Equal(t, good, parseImgMap(bad))
}
