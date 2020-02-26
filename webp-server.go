package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

	"github.com/chai2010/webp"
)

type Config struct {
	HOST         string
	PORT         string
	ImgPath      string `json:"IMG_PATH"`
	QUALITY      string
	AllowedTypes []string `json:"ALLOWED_TYPES"`
}

var configPath string

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

func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
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
	if err = ioutil.WriteFile(p2, buf.Bytes(), 0666); err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Save to webp ok")
	return nil
}

func init() {
	flag.StringVar(&configPath, "config", "config.json", "/path/to/config.json. (Default: ./config.json)")
	flag.Parse()
}

func main() {
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

		// Check the original image for existence
		OriginalImgExists := imageExists(ImgAbsolutePath)
		if !OriginalImgExists {
			c.Send("File not found!")
			c.SendStatus(404)
			return
		}

		// 1582558990
		STAT, err := os.Stat(ImgAbsolutePath)
		if err != nil {
			fmt.Println(err.Error())
		}
		ModifiedTime := STAT.ModTime().Unix()

		// /path/to/tsuki.jpg.1582558990.webp
		WebpImgPath := fmt.Sprintf("%s/%s.%d.webp", DirPath, ImgName, ModifiedTime)

		// /home/webp_server
		CurrentPath, err := os.Getwd()
		if err != nil {
			fmt.Println(err.Error())
		}

		// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
		WebpAbsolutePath := path.Clean(CurrentPath + "/exhaust" + WebpImgPath)

		// /home/webp_server/exhaust/path/to
		DirAbsolutePath := path.Clean(CurrentPath + "/exhaust" + DirPath)

		// Check file extension
		_, found := Find(AllowedTypes, ImgExt)
		if !found {
			c.Send("File extension not allowed!")
			c.SendStatus(403)
			return
		}

		// Check the original image for existence
		if !OriginalImgExists {
			// The original image doesn't exist, check the webp image, delete if processed.
			if imageExists(WebpAbsolutePath) {
				_ = os.Remove(WebpAbsolutePath)
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
			_ = os.MkdirAll(DirAbsolutePath, os.ModePerm)

			// cwebp -q 60 Cute-Baby-Girl.png -o Cute-Baby-Girl.webp
			q, _ := strconv.ParseFloat(QUALITY, 32)
			err = webpEncoder(ImgAbsolutePath, WebpAbsolutePath, float32(q))

			if err != nil {
				fmt.Println(err)
				c.SendStatus(400)
				c.Send("Bad file!")
				return
			}

			ImgNameCopy := string([]byte(ImgName))
			c.SendFile(WebpAbsolutePath)

			// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558100.webp <- older ones will be removed
			// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp <- keep the latest one
			WebpCachedImgPath := path.Clean(fmt.Sprintf("%s/exhaust%s/%s.*.webp", CurrentPath, DirPath, ImgNameCopy))
			matches, err := filepath.Glob(WebpCachedImgPath)
			if err != nil {
				fmt.Println(err.Error())
			} else {
				for _, p := range matches {
					if strings.Compare(WebpAbsolutePath, p) != 0 {
						_ = os.Remove(p)
					}
				}
			}
		}
	})

	app.Listen(ListenAddress)

}
