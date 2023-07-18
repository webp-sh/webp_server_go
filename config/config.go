package config

import (
	"encoding/json"
	"flag"
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
	ConfigPath  string
	Jobs        int
	DumpSystemd bool
	DumpConfig  bool
	ShowVersion bool
	ProxyMode   bool
	Prefetch    bool
	Config      jsonFile
	Version     = "0.9.6"
	WriteLock   = cache.New(5*time.Minute, 10*time.Minute)
)

const Metadata = "metadata"

type MetaFile struct {
	Id       string `json:"id"`       // hash of below path️, also json file name id.webp
	Path     string `json:"path"`     // local: path with width and height, proxy: full url
	Checksum string `json:"checksum"` // hash of original file or hash(etag). Use this to identify changes
}

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
	flag.StringVar(&ConfigPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.BoolVar(&Prefetch, "prefetch", false, "Prefetch and convert image to webp")
	flag.IntVar(&Jobs, "jobs", runtime.NumCPU(), "Prefetch thread, default is all.")
	flag.BoolVar(&DumpConfig, "dump-config", false, "Print sample config.json")
	flag.BoolVar(&DumpSystemd, "dump-systemd", false, "Print sample systemd service file.")
	flag.BoolVar(&ShowVersion, "V", false, "Show version information.")
}

func LoadConfig() {
	jsonObject, err := os.Open(ConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(jsonObject)
	_ = decoder.Decode(&Config)
	_ = jsonObject.Close()
	switchProxyMode()
}

type ExtraParams struct {
	Width  int // in px
	Height int // in px
}

func switchProxyMode() {
	matched, _ := regexp.MatchString(`^https?://`, Config.ImgPath)
	if matched {
		ProxyMode = true
	}
}
