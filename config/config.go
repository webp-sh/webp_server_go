package config

import (
	"encoding/json"
	"flag"
	"os"
	"regexp"
	"runtime"
	"time"
	"fmt"

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
  "ENABLE_EXTRA_PARAMS": false,
  "LAZY_MODE": false,
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
	MaxDefaultJobs int
	MaxHeavyJobs int
	LazyMode    bool
	DumpSystemd bool
	DumpConfig  bool
	VerboseMode bool
	ShowVersion bool
	ProxyMode   bool
	Prefetch    bool
	Config      jsonFile
	Version     = "0.9.4"
	WriteLock   = cache.New(5*time.Minute, 10*time.Minute)
	ConfigFlag *flag.FlagSet
	LazyTickerPeriod = time.Second * 5
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
	// Use a flagSet to avoid issues during TestMain* tests
	ConfigFlag = flag.NewFlagSet("main", flag.ContinueOnError)
	ConfigFlag.StringVar(&ConfigPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	ConfigFlag.BoolVar(&Prefetch, "prefetch", false, "Prefetch and convert image to webp")
	ConfigFlag.IntVar(&Jobs, "jobs", runtime.NumCPU(), "Prefetch thread, default is all.")
	ConfigFlag.BoolVar(&LazyMode, "lazy", false, "Convert images in the background, asynchronously")
	ConfigFlag.IntVar(&MaxDefaultJobs, "lazy-jobs", runtime.NumCPU(), "Max parallel tasks (WebP) in lazy mode, default is all.")
	ConfigFlag.IntVar(&MaxHeavyJobs, "lazy-heavy-jobs", runtime.NumCPU(), "Max parallel heavy tasks (AVIF) in lazy mode, default is all.")
	ConfigFlag.BoolVar(&DumpConfig, "dump-config", false, "Print sample config.json")
	ConfigFlag.BoolVar(&DumpSystemd, "dump-systemd", false, "Print sample systemd service file.")
	ConfigFlag.BoolVar(&VerboseMode, "v", false, "Verbose, print out debug info.")
	ConfigFlag.BoolVar(&ShowVersion, "V", false, "Show version information.")
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

// Element is an entry in the priority queue
type Element struct {
	ImageType   string
	Raw         string
	Optimized   string
	Quality     int
	ExtraParams ExtraParams
	Priority    int
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
