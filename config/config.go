package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

const (
	TimeDateFormat = "2006-01-02 15:04:05"
	FiberLogFormat = "${ip} - [${time}] ${method} ${url} ${status} ${referer} ${ua}\n"
	WebpMax        = 16383
	AvifMax        = 65536
	HttpRegexp     = `^https?://`
	SampleConfig   = `
{
  "HOST": "127.0.0.1",
  "PORT": "3333",
  "QUALITY": "80",
  "IMG_PATH": "./pics",
  "EXHAUST_PATH": "./exhaust",
  "IMG_MAP": {},
  "ALLOWED_TYPES": ["jpg","png","jpeg","bmp","svg"],
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
	ConfigPath     string
	Jobs           int
	DumpSystemd    bool
	DumpConfig     bool
	ShowVersion    bool
	ProxyMode      bool
	Prefetch       bool
	Config         = NewWebPConfig()
	Version        = "0.10.1"
	WriteLock      = cache.New(5*time.Minute, 10*time.Minute)
	RemoteRaw      = "./remote-raw"
	Metadata       = "./metadata"
	LocalHostAlias = "local"
)

type MetaFile struct {
	Id       string `json:"id"`       // hash of below path️, also json file name id.webp
	Path     string `json:"path"`     // local: path with width and height, proxy: full url
	Checksum string `json:"checksum"` // hash of original file or hash(etag). Use this to identify changes
}

type webpConfig struct {
	Host              string            `json:"HOST"`
	Port              string            `json:"PORT"`
	ImgPath           string            `json:"IMG_PATH"`
	Quality           int               `json:"QUALITY,string"`
	AllowedTypes      []string          `json:"ALLOWED_TYPES"`
	ImageMap          map[string]string `json:"IMG_MAP"`
	ExhaustPath       string            `json:"EXHAUST_PATH"`
	EnableAVIF        bool              `json:"ENABLE_AVIF"`
	EnableExtraParams bool              `json:"ENABLE_EXTRA_PARAMS"`
}

func NewWebPConfig() *webpConfig {
	return &webpConfig{
		Host:              "0.0.0.0",
		Port:              "3333",
		ImgPath:           "./pics",
		Quality:           80,
		AllowedTypes:      []string{"jpg", "png", "jpeg", "bmp", "svg"},
		ImageMap:          map[string]string{},
		ExhaustPath:       "./exhaust",
		EnableAVIF:        false,
		EnableExtraParams: false,
	}
}

func init() {
	flag.StringVar(&ConfigPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.BoolVar(&Prefetch, "prefetch", false, "Prefetch and convert image to WebP format.")
	flag.IntVar(&Jobs, "jobs", runtime.NumCPU(), "Prefetch thread, default is all.")
	flag.BoolVar(&DumpConfig, "dump-config", false, "Print sample config.json.")
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
	Config.ImageMap = parseImgMap(Config.ImageMap)

	// Read from ENV for override
	if os.Getenv("WEBP_HOST") != "" {
		Config.Host = os.Getenv("WEBP_HOST")
	}
	if os.Getenv("WEBP_PORT") != "" {
		Config.Port = os.Getenv("WEBP_PORT")
	}
	if os.Getenv("WEBP_IMG_PATH") != "" {
		Config.ImgPath = os.Getenv("WEBP_IMG_PATH")
	}
	if os.Getenv("WEBP_EXHAUST_PATH") != "" {
		Config.ExhaustPath = os.Getenv("WEBP_EXHAUST_PATH")
	}
	if os.Getenv("WEBP_QUALITY") != "" {
		quality, err := strconv.Atoi(os.Getenv("WEBP_QUALITY"))
		if err != nil {
			log.Warnf("WEBP_QUALITY is not a valid integer, using value in config.json %d", Config.Quality)
		} else {
			Config.Quality = quality
		}
	}
	if os.Getenv("WEBP_ALLOWED_TYPES") != "" {
		Config.AllowedTypes = strings.Split(os.Getenv("WEBP_ALLOWED_TYPES"), ",")
	}
	if os.Getenv("WEBP_ENABLE_AVIF") != "" {
		enableAVIF := os.Getenv("WEBP_ENABLE_AVIF")
		if enableAVIF == "true" {
			Config.EnableAVIF = true
		} else if enableAVIF == "false" {
			Config.EnableAVIF = false
		} else {
			log.Warnf("WEBP_ENABLE_AVIF is not a valid boolean, using value in config.json %t", Config.EnableAVIF)
		}
	}
	if os.Getenv("WEBP_ENABLE_EXTRA_PARAMS") != "" {
		enableExtraParams := os.Getenv("WEBP_ENABLE_EXTRA_PARAMS")
		if enableExtraParams == "true" {
			Config.EnableExtraParams = true
		} else if enableExtraParams == "false" {
			Config.EnableExtraParams = false
		} else {
			log.Warnf("WEBP_ENABLE_EXTRA_PARAMS is not a valid boolean, using value in config.json %t", Config.EnableExtraParams)
		}
	}
	if os.Getenv("WEBP_IMG_MAP") != "" {
		// TODO
	}

	fmt.Println("Config init complete")
	fmt.Println("Config", Config)

}

func parseImgMap(imgMap map[string]string) map[string]string {
	var parsedImgMap = map[string]string{}
	httpRegexpMatcher := regexp.MustCompile(HttpRegexp)
	for uriMap, uriMapTarget := range imgMap {
		if httpRegexpMatcher.Match([]byte(uriMap)) || strings.HasPrefix(uriMap, "/") {
			// Valid
			parsedImgMap[uriMap] = uriMapTarget
		} else {
			// Invalid
			log.Warnf("IMG_MAP key '%s' does matches '%s' or starts with '/' - skipped", uriMap, HttpRegexp)
		}
	}
	return parsedImgMap
}

type ExtraParams struct {
	Width  int // in px
	Height int // in px
}

func switchProxyMode() {
	matched, _ := regexp.MatchString(HttpRegexp, Config.ImgPath)
	if matched {
		// Enable proxy based on ImgPath should be deprecated in future versions
		log.Warn("Enable proxy based on ImgPath will be deprecated in future versions. Use IMG_MAP config options instead")
		ProxyMode = true
	}
}
