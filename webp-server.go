package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gofiber/fiber"
	"log"
	"os"
	"path"
	"runtime"
)

type Config struct {
	HOST         string
	PORT         string
	ImgPath      string `json:"IMG_PATH"`
	QUALITY      string
	AllowedTypes []string `json:"ALLOWED_TYPES"`
	ExhaustPath  string   `json:"EXHAUST_PATH"`
}

const version = "0.0.4"

var configPath string
var prefetch bool
var jobs int
var dumpConfig bool
var dumpSystemd bool

const sampleConfig = `
{
	"HOST": "127.0.0.1",
	"PORT": "3333",
	"QUALITY": "80",
	"IMG_PATH": "/path/to/pics",
	"EXHAUST_PATH": "",
	"ALLOWED_TYPES": ["jpg","png","jpeg","bmp","gif"]
}`
const sampleSystemd = `
[Unit]
Description=WebP Server
Documentation=https://github.com/webp-sh/webp_server_go
After=nginx.target

[Service]
Type=simple
StandardError=journal
AmbientCapabilities=CAP_NET_BIND_SERVICE
WorkingDirectory=/opt/webps
ExecStart=/opt/webps/webp-server --config /opt/webps/config.json
ExecReload=/bin/kill -HUP $MAINPID
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
	return config
}

func init() {
	flag.StringVar(&configPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.BoolVar(&prefetch, "prefetch", false, "Prefetch and convert image to webp")
	flag.IntVar(&jobs, "jobs", runtime.NumCPU(), "Prefetch thread, default is all.")
	flag.BoolVar(&dumpConfig, "dump-config", false, "Print sample config.json")
	flag.BoolVar(&dumpSystemd, "dump-systemd", false, "Print sample systemd service file.")
	flag.Parse()
}

func main() {
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

	// process cli params
	if dumpConfig {
		fmt.Println(sampleConfig)
		os.Exit(0)
	}
	if dumpSystemd {
		fmt.Println(sampleSystemd)
		os.Exit(0)
	}

	if prefetch {
		go PrefetchImages(confImgPath, ExhaustPath, QUALITY)
	}

	app := fiber.New()
	app.Banner = false
	app.Server = "WebP Server Go"

	ListenAddress := HOST + ":" + PORT

	// Server Info
	ServerInfo := "WebP Server " + version + " is running at " + ListenAddress
	fmt.Println(ServerInfo)

	app.Get("/*", Convert(confImgPath, ExhaustPath, AllowedTypes, QUALITY))
	app.Listen(ListenAddress)

}
