package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/gofiber/fiber"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	HOST         string
	PORT         string
	ImgPath      string `json:"IMG_PATH"`
	QUALITY      string
	AllowedTypes []string `json:"ALLOWED_TYPES"`
	ExhaustPath  string   `json:"EXHAUST_PATH"`
}

const version = "0.1.1"

var configPath string
var prefetch bool
var jobs int
var dumpConfig bool
var dumpSystemd bool
var verboseMode bool

const sampleConfig = `
{
	"HOST": "127.0.0.1",
	"PORT": "3333",
	"QUALITY": "80",
	"IMG_PATH": "/path/to/pics",
	"EXHAUST_PATH": "",
	"ALLOWED_TYPES": ["jpg","png","jpeg","bmp"]
}`
const sampleSystemd = `
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

func loadConfig(path string) Config {
	var config Config
	jsonObject, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonObject.Close()
	decoder := json.NewDecoder(jsonObject)
	_ = decoder.Decode(&config)
	_, err = os.Stat(config.ImgPath)
	if err != nil {
		log.Fatalf("Your image path %s is incorrect.Please check and confirm.", config.ImgPath)
	}

	return config
}

func init() {
	flag.StringVar(&configPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.BoolVar(&prefetch, "prefetch", false, "Prefetch and convert image to webp")
	flag.IntVar(&jobs, "jobs", runtime.NumCPU(), "Prefetch thread, default is all.")
	flag.BoolVar(&dumpConfig, "dump-config", false, "Print sample config.json")
	flag.BoolVar(&dumpSystemd, "dump-systemd", false, "Print sample systemd service file.")
	flag.BoolVar(&verboseMode, "v", false, "Verbose, print out debug info.")
	flag.Parse()
	// Logrus
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)
	Formatter := &log.TextFormatter{
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		TimestampFormat:           "2006-01-02 15:04:05",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return fmt.Sprintf("[%s()]", f.Function), ""
		},
	}
	log.SetFormatter(Formatter)

	if verboseMode {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug mode is enable!")
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func main() {
	// process cli params
	if dumpConfig {
		fmt.Println(sampleConfig)
		os.Exit(0)
	}
	if dumpSystemd {
		fmt.Println(sampleSystemd)
		os.Exit(0)
	}

	go autoUpdate()
	config := loadConfig(configPath)

	HOST := config.HOST
	PORT := config.PORT
	confImgPath := path.Clean(config.ImgPath)
	QUALITY := config.QUALITY
	AllowedTypes := config.AllowedTypes
	var ExhaustPath string
	if len(config.ExhaustPath) == 0 {
		ExhaustPath = "./exhaust"
	} else {
		ExhaustPath = config.ExhaustPath
	}

	if prefetch {
		go PrefetchImages(confImgPath, ExhaustPath, QUALITY)
	}

	app := fiber.New()
	app.Banner = false
	app.Server = "WebP Server Go"

	ListenAddress := HOST + ":" + PORT

	// Server Info
	log.Infof("WebP Server %s %s", version, ListenAddress)

	app.Get("/*", Convert(confImgPath, ExhaustPath, AllowedTypes, QUALITY))
	app.Listen(ListenAddress)

}
