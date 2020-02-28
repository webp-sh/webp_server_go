package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/chai2010/webp"
	"github.com/gofiber/fiber"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	HOST         string
	PORT         string
	ImgPath      string `json:"IMG_PATH"`
	QUALITY      string
	AllowedTypes []string `json:"ALLOWED_TYPES"`
}

var configPath string
var prefetch bool

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

func webpEncoder(p1, p2 string, quality float32) (err error) {
	// if convert fails, return error; success nil
	var buf bytes.Buffer
	var img image.Image

	data, err := ioutil.ReadFile(p1)
	if err != nil {
		return
	}
	contentType := GetFileContentType(data[:512])
	if strings.Contains(contentType, "jpeg") {
		img, _ = jpeg.Decode(bytes.NewReader(data))
	} else if strings.Contains(contentType, "png") {
		img, _ = png.Decode(bytes.NewReader(data))
	}

	if img == nil {
		log.Println("Image file is corrupted or not supported!")
		err = errors.New("image file is corrupted or not supported")
		return
	}

	if err = webp.Encode(&buf, img, &webp.Options{Lossless: true, Quality: quality}); err != nil {
		log.Println(err)
		return
	}
	if err = ioutil.WriteFile(p2, buf.Bytes(), os.ModePerm); err != nil {
		log.Println(err)
		return
	}

	fmt.Printf("Save to %s ok\n", p2)
	return nil
}

func init() {
	flag.StringVar(&configPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.BoolVar(&prefetch, "prefetch", false, "Prefetch and convert image to webp")
	flag.Parse()
}

func Convert(ImgPath string, AllowedTypes []string, QUALITY string) func(c *fiber.Ctx) {
	return func(c *fiber.Ctx) {
		//basic vars
		var reqURI = c.Path()                         // mypic/123.jpg
		var RawImagePath = path.Join(ImgPath, reqURI) // /home/xxx/mypic/123.jpg
		var ImgFilename = path.Base(reqURI)           // pure filename, 123.jpg
		var finalFile string                          // We'll only need one c.sendFile()
		// Check for Safari users. If they're Safari, just simply ignore everything.
		UA := c.Get("User-Agent")
		if strings.Contains(UA, "Safari") && !strings.Contains(UA, "Chrome") &&
			!strings.Contains(UA, "Firefox") {
			finalFile = RawImagePath
			return
		}

		// check ext
		for _, ext := range AllowedTypes {
			haystack := strings.ToLower(ImgFilename)
			needle := strings.ToLower("." + ext)
			if strings.HasSuffix(haystack, needle) {
				break
			} else {
				c.Send("File extension not allowed!")
				c.SendStatus(403)
				return
			}
		}

		// Check the original image for existence,
		if !imageExists(RawImagePath) {
			c.Send("Image not found!")
			c.SendStatus(404)
			return
		}

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
		WebpAbsolutePath := path.Clean(path.Join(cwd, "exhaust", path.Dir(reqURI), WebpFilename))

		if imageExists(WebpAbsolutePath) {
			finalFile = WebpAbsolutePath
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
			_ = os.MkdirAll(path.Dir(WebpAbsolutePath), os.ModePerm)
			q, _ := strconv.ParseFloat(QUALITY, 32)
			err = webpEncoder(RawImagePath, WebpAbsolutePath, float32(q))

			if err != nil {
				fmt.Println(err)
				c.SendStatus(400)
				c.Send("Bad file!")
				return
			}
			finalFile = WebpAbsolutePath
		}
		c.SendFile(finalFile)
	}
}

func main() {

	if prefetch {
		fmt.Println(`Prefetch will convert all your images to webp, 
it may take some time and consume a lot of CPU resource. Do you want to proceed(Y/N)`)
		reader := bufio.NewReader(os.Stdin)
		char, _, _ := reader.ReadRune() //y Y ente
		if char == 121 || char == 10 || char == 89 {
			//TODO prefetch
		}
	}
	app := fiber.New()
	app.Banner = false
	app.Server = "WebP Server Go"

	config := loadConfig(configPath)

	HOST := config.HOST
	PORT := config.PORT
	ImgPath := config.ImgPath
	QUALITY := config.QUALITY
	AllowedTypes := config.AllowedTypes

	ListenAddress := HOST + ":" + PORT

	// Server Info
	ServerInfo := "WebP Server is running at " + ListenAddress
	fmt.Println(ServerInfo)

	app.Get("/*", Convert(ImgPath, AllowedTypes, QUALITY))

	app.Listen(ListenAddress)

}
