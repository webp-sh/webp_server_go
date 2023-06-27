package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

const (
	TimeDateFormat = "2006-01-02 15:04:05"
	FiberLogFormat = "${ip} - [${time}] ${method} ${url} ${status} ${referer} ${ua}\n"
	WebpMax        = 16383
	AvifMax        = 65536
	RemoteRaw      = "remote-raw"

	SampleConfig = `
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

	SampleSystemd = `
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

var (
	configPath  string
	Jobs        int
	DumpSystemd bool
	DumpConfig  bool
	ShowVersion bool
	ProxyMode   bool
	Prefetch    bool
	Config      jsonFile
	Version     = "0.9.0"
	WriteLock   = cache.New(5*time.Minute, 10*time.Minute)
)

type jsonFile struct {
	Host              string   `json:"HOST"`
	Port              string   `json:"PORT"`
	ImgPath           string   `json:"IMG_PATH"`
	Quality           int      `json:"QUALITY,string"`
	AllowedTypes      []string `json:"ALLOWED_TYPES"`
	ExhaustPath       string   `json:"EXHAUST_PATH"`
	EnableAVIF        bool     `json:"ENABLE_AVIF"`
	EnableExtraParams bool     `json:"ENABLE_EXTRA_PARAMS"`
}

func init() {
	flag.StringVar(&configPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.BoolVar(&Prefetch, "prefetch", false, "Prefetch and convert image to webp")
	flag.IntVar(&Jobs, "jobs", runtime.NumCPU(), "Prefetch thread, default is all.")
	flag.BoolVar(&DumpConfig, "dump-config", false, "Print sample config.json")
	flag.BoolVar(&DumpSystemd, "dump-systemd", false, "Print sample systemd service file.")
	flag.BoolVar(&ShowVersion, "V", false, "Show version information.")
	flag.Parse()
	Config = loadConfig()
	switchProxyMode()
}

func loadConfig() (config jsonFile) {
	jsonObject, err := os.Open(configPath)
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(jsonObject)
	_ = decoder.Decode(&config)
	_ = jsonObject.Close()
	return config
}

type ExtraParams struct {
	Width  int // in px
	Height int // in px
}

// String : convert ExtraParams to string, used to generate cache path
func (e *ExtraParams) String() string {
	return fmt.Sprintf("_width=%d&height=%d", e.Width, e.Height)
}

func switchProxyMode() {
	matched, _ := regexp.MatchString(`^https?://`, Config.ImgPath)
	if matched {
		ProxyMode = true
	}
}
