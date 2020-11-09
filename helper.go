package main

import (
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"strings"

	log "github.com/sirupsen/logrus"
)

func chanErr(ccc chan int) {
	if ccc != nil {
		ccc <- 1
	}
}

func getFileContentType(buffer []byte) string {
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType
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

func imageExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	log.Debugf("file %s exists!", filename)
	return !info.IsDir()
}

// Check for remote filepath, e.g: https://test.webp.sh/node.png
// return StatusCode, etagValue
func getRemoteImageInfo(fileUrl string) (int, string) {
	res, err := http.Head(fileUrl)
	if err != nil {
		log.Fatal("Connection to remote error!")
		return http.StatusInternalServerError, ""
	}
	if res.StatusCode != 404 {
		etagValue := res.Header.Get("etag")
		if etagValue == "" {
			log.Info("Remote didn't return etag in header, please check.")
		} else {
			return 200, etagValue
		}
	}
	return res.StatusCode, ""
}

func fetchRemoteImage(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_ = os.MkdirAll(path.Dir(filepath), 0755)
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Given /path/to/node.png
// Delete /path/to/node.png*
func cleanProxyCache(cacheImagePath string) {
	// Delete /node.png*
	files, err := filepath.Glob(cacheImagePath + "*")
	if err != nil {
		fmt.Println(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Info(err)
		}
	}
}

func genWebpAbs(RawImagePath string, ExhaustPath string, ImgFilename string, reqURI string) (string, string) {
	// get file mod time
	STAT, err := os.Stat(RawImagePath)
	if err != nil {
		log.Error(err.Error())
		return "", ""
	}
	ModifiedTime := STAT.ModTime().Unix()
	// webpFilename: abc.jpg.png -> abc.jpg.png1582558990.webp
	WebpFilename := fmt.Sprintf("%s.%d.webp", ImgFilename, ModifiedTime)
	cwd, _ := os.Getwd()

	// /home/webp_server/exhaust/path/to/tsuki.jpg.1582558990.webp
	// Custom Exhaust: /path/to/exhaust/web_path/web_to/tsuki.jpg.1582558990.webp
	WebpAbsolutePath := path.Clean(path.Join(ExhaustPath, path.Dir(reqURI), WebpFilename))
	return cwd, WebpAbsolutePath
}

func genEtag(ImgAbsPath string) string {
	data, err := ioutil.ReadFile(ImgAbsPath)
	if err != nil {
		log.Info(err)
	}
	crc := crc32.ChecksumIEEE(data)
	return fmt.Sprintf(`W/"%d-%08X"`, len(data), crc)
}

func goOrigin(UA string) bool {
	// for more information, please check test case
	if strings.Contains(UA, "Firefox") || strings.Contains(UA, "Chrome") {
		// Chrome or firefox on macOS Windows
	} else if strings.Contains(UA, "Android") || strings.Contains(UA, "Linux") {
		// on Android and Linux
	} else if strings.Contains(UA, "FxiOS") || strings.Contains(UA, "CriOS") {
		//firefox and Chrome on iOS
		return true
	} else {
		return true
	}
	return false
}
