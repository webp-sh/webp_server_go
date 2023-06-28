package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
	assert.Equal(t, Config.ExhaustPath, "./exhaust")
}

func TestExtraParamsString(t *testing.T) {
	param := ExtraParams{
		Width:  100,
		Height: 100,
	}
	assert.Equal(t, param.String(), "_width=100&height=100")

}

func TestSwitchProxyMode(t *testing.T) {
	switchProxyMode()
	assert.False(t, ProxyMode)
	Config.ImgPath = "https://picsum.photos"
	switchProxyMode()
	assert.True(t, ProxyMode)
}
