package main

import "fmt"

type Config struct {
	Host              string   `json:"HOST"`
	Port              string   `json:"PORT"`
	ImgPath           string   `json:"IMG_PATH"`
	Quality           int      `json:"QUALITY,string"`
	AllowedTypes      []string `json:"ALLOWED_TYPES"`
	ExhaustPath       string   `json:"EXHAUST_PATH"`
	EnableAVIF        bool     `json:"ENABLE_AVIF"`
	EnableExtraParams bool     `json:"ENABLE_EXTRA_PARAMS"`
}

type ExtraParams struct {
	Width  int // in px
	Height int // in px
}

// String : convert ExtraParams to string, used to generate cache path
func (e *ExtraParams) String() string {
	return fmt.Sprintf("_width=%d&height=%d", e.Width, e.Height)
}

var (
	configPath               string
	jobs                     int
	dumpConfig, dumpSystemd  bool
	verboseMode, showVersion bool
	prefetch, proxyMode      bool
	remoteRaw                = "remote-raw"
	config                   Config
	version                  = "0.8.0"
)

const (
	sampleConfig = `
{
  "HOST": "127.0.0.1",
  "PORT": "3333",
  "QUALITY": "80",
  "IMG_PATH": "./pics",
  "EXHAUST_PATH": "./exhaust",
  "ALLOWED_TYPES": ["jpg","png","jpeg","bmp"],
  "ENABLE_AVIF": false,
  "ENABLE_EXTRA_PARAMS": false
}`

	sampleSystemd = `
[Unit]
Description=WebP Server Go
Documentation=https://github.com/webp-sh/webp_server_go
After=nginx.target

[Service]
Type=simple
StandardError=journal
WorkingDirectory=/opt/webps
ExecStart=/opt/webps/webp-server --config /opt/webps/config.json
Restart=always
RestartSec=3s

[Install]
WantedBy=multi-user.target`
)

const (
	webpMax = 16383
	avifMax = 65536
)
