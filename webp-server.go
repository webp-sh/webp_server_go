package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/image/bmp"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber"
)

type Config struct {
	HOST         string
	PORT         string
	ImgPath      string `json:"IMG_PATH"`
	QUALITY      string
	AllowedTypes []string `json:"ALLOWED_TYPES"`
	ExhaustPath  string   `json:"EXHAUST_PATH"`
}

const version = "0.0.3"

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
  "IMG_PATH": "/Users/benny/goLandProject/webp_server_go/pics",
  "EXHAUST_PATH": "",
  "ALLOWED_TYPES": ["jpg", "png", "jpeg", "bmp", "gif"]
}
`
const sampleSystemd = `
[Unit]
Description=WebP Server
Documentation=https://github.com/n0vad3v/webp_server_go
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
WantedBy=multi-user.target
`

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

func imageExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func GetFileContentType(buffer []byte) string {
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType
}

func chanErr(ccc chan int) {
	if ccc != nil {
		ccc <- 1
	}
}
func webpEncoder(p1, p2 string, quality float32, Log bool, c chan int) (err error) {
	// if convert fails, return error; success nil
	var buf bytes.Buffer
	var img image.Image

	data, err := ioutil.ReadFile(p1)
	if err != nil {
		chanErr(c)
		return
	}

	contentType := GetFileContentType(data[:512])
	if strings.Contains(contentType, "jpeg") {
		img, _ = jpeg.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "png") {
		img, _ = png.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "bmp") {
		img, _ = bmp.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "gif") {
		// TODO: need to support animated webp
		img, _ = gif.Decode(bytes.NewReader(data))
	}

	if img == nil {
		msg := "image file " + path.Base(p1) + " is corrupted or not supported"
		log.Println(msg)
		err = errors.New(msg)
		chanErr(c)
		return
	}

	if err = webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: quality}); err != nil {
		log.Println(err)
		chanErr(c)
		return
	}
	if err = ioutil.WriteFile(p2, buf.Bytes(), 0755); err != nil {
		log.Println(err)
		chanErr(c)
		return
	}

	if Log {
		fmt.Printf("Save to %s ok\n", p2)
	}

	chanErr(c)

	return nil
}

func init() {
	flag.StringVar(&configPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.BoolVar(&prefetch, "prefetch", false, "Prefetch and convert image to webp")
	flag.IntVar(&jobs, "jobs", runtime.NumCPU(), "Prefetch thread, default is all.")
	flag.BoolVar(&dumpConfig, "dump-config", false, "Print sample config.json")
	flag.BoolVar(&dumpSystemd, "dump-systemd", false, "Print sample systemd service file.")
	flag.Parse()
}

func Convert(ImgPath string, ExhaustPath string, AllowedTypes []string, QUALITY string) func(c *fiber.Ctx) {
	return func(c *fiber.Ctx) {
		//basic vars
		var reqURI = c.Path()                        // mypic/123.jpg
		var RawImageAbs = path.Join(ImgPath, reqURI) // /home/xxx/mypic/123.jpg
		var ImgFilename = path.Base(reqURI)          // pure filename, 123.jpg
		var finalFile string                         // We'll only need one c.sendFile()
		// Check for Safari users. If they're Safari, just simply ignore everything.
		UA := c.Get("User-Agent")
		if strings.Contains(UA, "Safari") && !strings.Contains(UA, "Chrome") &&
			!strings.Contains(UA, "Firefox") {
			c.SendFile(RawImageAbs)
			return
		}

		// check ext
		// TODO: may remove this function. Check in Nginx.
		var allowed = false
		for _, ext := range AllowedTypes {
			haystack := strings.ToLower(ImgFilename)
			needle := strings.ToLower("." + ext)
			if strings.HasSuffix(haystack, needle) {
				allowed = true
				break
			} else {
				allowed = false
			}
		}
		if !allowed {
			c.Send("File extension not allowed!")
			c.SendStatus(403)
			return
		}

		// Check the original image for existence,
		if !imageExists(RawImageAbs) {
			c.Send("Image not found!")
			c.SendStatus(404)
			return
		}

		cwd, WebpAbsPath := genWebpAbs(RawImageAbs, ExhaustPath, ImgFilename, reqURI)

		if imageExists(WebpAbsPath) {
			finalFile = WebpAbsPath
		} else {
			// we don't have abc.jpg.png1582558990.webp
			// delete the old pic and convert a new one.
			// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
			destHalfFile := path.Clean(path.Join(cwd, "exhaust", path.Dir(reqURI), ImgFilename))
			matches, err := filepath.Glob(destHalfFile + "*")
			if err != nil {
				fmt.Println(err.Error())
			} else {
				// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558100.webp <- older ones will be removed
				// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp <- keep the latest one
				for _, p := range matches {
					if strings.Compare(destHalfFile, p) != 0 {
						_ = os.Remove(p)
					}
				}
			}

			//for webp, we need to create dir first
			_ = os.MkdirAll(path.Dir(WebpAbsPath), 0755)
			q, _ := strconv.ParseFloat(QUALITY, 32)
			err = webpEncoder(RawImageAbs, WebpAbsPath, float32(q), true, nil)

			if err != nil {
				fmt.Println(err)
				c.SendStatus(400)
				c.Send("Bad file!")
				return
			}
			finalFile = WebpAbsPath
		}
		c.SendFile(finalFile)
	}
}

func fileCount(dir string) int {
	count := 0
	_ = filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				count += 1
			}
			return nil
		})
	return count
}

func genWebpAbs(RawImagePath string, ExhaustPath string, ImgFilename string, reqURI string) (string, string) {
	// get file mod time
	STAT, err := os.Stat(RawImagePath)
	if err != nil {
		fmt.Println(err.Error())
	}
	ModifiedTime := STAT.ModTime().Unix()
	// webpFilename: abc.jpg.png -> abc.jpg.png1582558990.webp
	var WebpFilename = fmt.Sprintf("%s.%d.webp", ImgFilename, ModifiedTime)
	cwd, _ := os.Getwd()

	// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
	// Custom Exhaust: /path/to/exhaust/web_path/web_to/tsuki.jpg.1582558990.webp
	WebpAbsolutePath := path.Clean(path.Join(ExhaustPath, path.Dir(reqURI), WebpFilename))
	return cwd, WebpAbsolutePath
}

func prefetchImages(confImgPath string, ExhaustPath string, QUALITY string) {
	fmt.Println(`Prefetch will convert all your images to webp, it may take some time and consume a lot of CPU resource. Do you want to proceed(Y/n)`)
	reader := bufio.NewReader(os.Stdin)
	char, _, _ := reader.ReadRune() //y Y enter
	// maximum ongoing prefetch is depending on your core of CPU
	log.Printf("Prefetching using %d cores", jobs)
	var finishChan = make(chan int, jobs)
	for i := 0; i < jobs; i++ {
		finishChan <- 0
	}
	if char == 121 || char == 10 || char == 89 {
		//prefetch, recursive through the dir
		all := fileCount(confImgPath)
		count := 0
		err := filepath.Walk(confImgPath,
			func(picAbsPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				// RawImagePath string, ImgFilename string, reqURI string
				proposedURI := strings.Replace(picAbsPath, confImgPath, "", 1)
				_, p2 := genWebpAbs(picAbsPath, ExhaustPath, info.Name(), proposedURI)
				q, _ := strconv.ParseFloat(QUALITY, 32)
				_ = os.MkdirAll(path.Dir(p2), 0755)
				go webpEncoder(picAbsPath, p2, float32(q), false, finishChan)
				count += <-finishChan
				//progress bar
				_, _ = fmt.Fprintf(os.Stdout, "[Webp Server started] - convert in progress: %d/%d\r", count, all)
				return nil
			})
		if err != nil {
			log.Println(err)
		}
	}

	_, _ = fmt.Fprintf(os.Stdout, "Prefetch completeY(^_^)Y\n\n")

}
func autoUpdate() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Download error.", err)
		}
	}()

	var api = "https://api.github.com/repos/webp-sh/webp_server_go/releases/latest"
	type Result struct {
		TagName string `json:"tag_name"`
	}
	var res Result
	resp1, _ := http.Get(api)
	data1, _ := ioutil.ReadAll(resp1.Body)
	_ = json.Unmarshal(data1, &res)

	var gitVersion = res.TagName

	if gitVersion > version {
		log.Printf("Time to update! New version %s found!", gitVersion)
	} else {
		log.Println("No new version found.")
		return
	}

	var filename = fmt.Sprintf("webp-server-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOARCH == "windows" {
		filename += ".exe"
	}
	var releaseUrl = "https://github.com/webp-sh/webp_server_go/releases/latest/download/" + filename
	log.Println("Downloading binary...")
	resp, _ := http.Get(releaseUrl)
	if resp.StatusCode != 200 {
		log.Printf("%s-%s not found on release. "+
			"Contact developers to supply your version", runtime.GOOS, runtime.GOARCH)
		return
	}
	data, _ := ioutil.ReadAll(resp.Body)
	_ = os.Mkdir("update", 0755)
	err := ioutil.WriteFile(path.Join("update", filename), data, 0755)

	if err == nil {
		log.Println("Update complete. Please find your binary from update directory.")
	}
	_ = resp.Body.Close()
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
		go prefetchImages(confImgPath, ExhaustPath, QUALITY)
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
