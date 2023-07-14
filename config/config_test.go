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
}

func TestSwitchProxyMode(t *testing.T) {
	switchProxyMode()
	assert.False(t, ProxyMode)
	Config.ImgPath = "https://picsum.photos"
	switchProxyMode()
	assert.True(t, ProxyMode)
}

// func TestSwitchProxyModeImgMap(t *testing.T) {
// 	switchProxyMode()
// 	assert.False(t, ProxyMode)
// 	Config.ImageMap = map[string]string{
// 		"https://picsum.photos": "https://picsum.photos",
// 		} 
// 	switchProxyMode()
// 	assert.True(t, ProxyMode)
// }