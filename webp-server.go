package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Host         string   `json:"HOST"`
	Port         string   `json:"Port"`
	ImgPath      string   `json:"IMG_PATH"`
	Quality      string   `json:"Quality"`
	AllowedTypes []string `json:"ALLOWED_TYPES"`
	ExhaustPath  string   `json:"EXHAUST_PATH"`
}

var (
	configPath                                     string
	jobs                                           int
	dumpConfig, dumpSystemd, verboseMode, prefetch bool

	proxyMode bool
	config    Config
)

const (
	version      = "0.2.1"
	sampleConfig = `
{
	"Host": "127.0.0.1",
	"Port": "3333",
	"quality": "80",
	"IMG_PATH": "/path/to/pics",
	"EXHAUST_PATH": "",
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

func loadConfig(path string) Config {
	jsonObject, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(jsonObject)
	_ = decoder.Decode(&config)
	_ = jsonObject.Close()
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
	config = loadConfig(configPath)

	// Check for remote address
	matched, _ := regexp.MatchString(`^https?://`, config.ImgPath)
	proxyMode = false
	if matched {
		proxyMode = true

	} else {
		_, err := os.Stat(config.ImgPath)
		if err != nil {
			log.Fatalf("Your image path %s is incorrect.Please check and confirm.", config.ImgPath)
		}
	}
	if len(config.ExhaustPath) == 0 {
		config.ExhaustPath = "./exhaust"
	}

	if prefetch {
		go prefetchImages(config.ImgPath, config.ExhaustPath, config.Quality)
	}

	app := fiber.New()
	ListenAddress := config.Host + ":" + config.Port

	// Server Info
	log.Infof("WebP Server %s %s", version, ListenAddress)

	app.Get("/*", convert)
	_ = app.Listen(ListenAddress)

}
