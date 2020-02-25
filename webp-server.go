package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
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
}

var configPath string

func webpEncoder(p1, p2 string, quality float32) {
	var buf bytes.Buffer
	var img image.Image

	file_data, err := os.Open(p1)
	if err != nil {
		panic(err)
	}
	img_data, _ := ioutil.ReadFile(p1)
	defer file_data.Close()

	contentType, err := GetFileContentType(file_data)
	if strings.Contains(contentType, "jpg") || strings.Contains(contentType, "jpeg") {
		img, _ = jpeg.Decode(bytes.NewReader(img_data))
	} else if strings.Contains(contentType, "png") {
		img, _ = png.Decode(bytes.NewReader(img_data))
	}

	if err := webp.Encode(&buf, img, &webp.Options{Lossless: true, Quality: quality}); err != nil {
		log.Println(err)
	}
	if err := ioutil.WriteFile(p2, buf.Bytes(), 0666); err != nil {
		log.Println(err)
	}

	fmt.Println("Save output.webp ok")
}

func init() {
	// Config Here
	flag.StringVar(&configPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.Parse()
	//flag.PrintDefaults()
}

func main() {
	app := fiber.New()
	app.Banner = false
	app.Server = "WebP Server Go"

	config := load_config(configPath)

	HOST := config.HOST
	PORT := config.PORT
	ImgPath := config.ImgPath
	QUALITY := config.QUALITY
	AllowedTypes := config.AllowedTypes

	ListenAddress := HOST + ":" + PORT

	// Server Info
	ServerInfo := "WebP Server is running at " + ListenAddress
	fmt.Println(ServerInfo)

	app.Get("/*", func(c *fiber.Ctx) {

		// /var/www/IMG_PATH/path/to/tsuki.jpg
		ImgAbsolutePath := ImgPath + c.Path()

		// /path/to/tsuki.jpg
		ImgPath := c.Path()

		// jpg
		seps := strings.Split(path.Ext(ImgPath), ".")
		var ImgExt string
		if len(seps) >= 2 {
			ImgExt = seps[1]
		} else {
			c.Send("Invalid request")
			return
		}

		// tsuki.jpg
		ImgName := path.Base(ImgPath)

		// /path/to
		DirPath := path.Dir(ImgPath)

		// /path/to/tsuki.jpg.webp
		WebpImgPath := DirPath + "/" + ImgName + ".webp"

		// /home/webp_server
		CurrentPath, err := os.Getwd()
		if err != nil {
			fmt.Println(err.Error())
		}

		// /home/webp_server/exhaust/path/to/tsuki.webp
		WebpAbsolutePath := CurrentPath + "/exhaust" + WebpImgPath

		// /home/webp_server/exhaust/path/to
		DirAbsolutePath := CurrentPath + "/exhaust" + DirPath

		// Check file extension
		_, found := Find(AllowedTypes, ImgExt)
		if !found {
			c.Send("File extension not allowed!")
			c.SendStatus(403)
			return
		}

		// Check the original image for existence
		if !imageExists(ImgAbsolutePath) {
			// The original image doesn't exist, check the webp image, delete if processed.
			if imageExists(WebpAbsolutePath) {
				os.Remove(WebpAbsolutePath)
			}
			c.Send("File not found!")
			c.SendStatus(404)
			return
		}

		// Check for Safari users
		UA := c.Get("User-Agent")
		if strings.Contains(UA, "Safari") && !strings.Contains(UA, "Chrome") && !strings.Contains(UA, "Firefox") {
			c.SendFile(ImgAbsolutePath)
			return
		}

		if imageExists(WebpAbsolutePath) {
			c.SendFile(WebpAbsolutePath)
		} else {
			// Mkdir
			os.MkdirAll(DirAbsolutePath, os.ModePerm)

			// cwebp -q 60 Cute-Baby-Girl.png -o Cute-Baby-Girl.webp

			q, _ := strconv.ParseFloat(QUALITY, 32)
			webpEncoder(ImgAbsolutePath, WebpAbsolutePath, float32(q))
			if err != nil {
				fmt.Println(err)
			}
			c.SendFile(WebpAbsolutePath)
		}
	})

	app.Listen(ListenAddress)
}

func load_config(path string) Config {
	var config Config
	jsonObject, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer jsonObject.Close()
	decoder := json.NewDecoder(jsonObject)
	decoder.Decode(&config)
	return config
}

func imageExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func GetFileContentType(out *os.File) (string, error) {

	buffer := make([]byte, 512)

	_, err := out.Read(buffer)
	if err != nil {
		return "", err
	}

	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	return contentType, nil
}
