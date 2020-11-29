// webp_server_go - config
// 2020-11-27 13:05
// Benny <benny.think@gmail.com>

package main

type Config struct {
	Host         string   `json:"HOST"`
	Port         string   `json:"PORT"`
	ImgPath      string   `json:"IMG_PATH"`
	Quality      string   `json:"QUALITY"`
	AllowedTypes []string `json:"ALLOWED_TYPES"`
	ExhaustPath  string   `json:"EXHAUST_PATH"`
}

var (
	configPath               string
	jobs                     int
	dumpConfig, dumpSystemd  bool
	verboseMode, showVersion bool
	prefetch, proxyMode      bool

	config  Config
	version = "0.3.0"
)

const (
	NotCompressed = "not_compressed"
	WebpBigger    = "webp_bigger"
)

const (
	sampleConfig = `
{
  "HOST": "127.0.0.1",
  "PORT": "3333",
  "QUALITY": "80",
  "IMG_PATH": "./pics",
  "EXHAUST_PATH": "./exhaust",
  "ALLOWED_TYPES": ["jpg","png","jpeg","bmp"]
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
