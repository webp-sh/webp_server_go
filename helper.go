package main

import (
	"fmt"
	"github.com/gofiber/fiber"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func ChanErr(ccc chan int) {
	if ccc != nil {
		ccc <- 1
	}
}

func GetFileContentType(buffer []byte) string {
	// Use the net/http package's handy DectectContentType function. Always returns a valid
	// content-type by returning "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType
}

func FileCount(dir string) int {
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

func ImageExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	log.Debugf("file %s exists!", filename)
	return !info.IsDir()
}

func GenWebpAbs(RawImagePath string, ExhaustPath string, ImgFilename string, reqURI string) (string, string) {
	// get file mod time
	STAT, err := os.Stat(RawImagePath)
	if err != nil {
		log.Error(err.Error())
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

func CheckUA(c *fiber.Ctx, RawImageAbs string) (string, bool) {
	// reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent/Firefox
	// https://developer.chrome.com/multidevice/user-agent#chrome_for_ios_user_agent
	// Chrome
	// ✅ Windows: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36
	// ✅ macOS:   Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36
	// ✅ Linux:   Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36
	// ✅ iOS:     Mozilla/5.0 (iPhone; CPU iPhone OS 13_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/83.0.4103.63 Mobile/15E148 Safari/604.1
	// ✅ Android: Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.60 Mobile Safari/537.36

	// Firefox
	// ✅ Windows: Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:76.0) Gecko/20100101 Firefox/76.0
	// ✅ macOS:   Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:76.0) Gecko/20100101 Firefox/76.0
	// ✅ Linux:   Mozilla/5.0 (X11; Linux i686; rv:76.0) Gecko/20100101 Firefox/76.0
	// ✅ iOS:     Mozilla/5.0 (iPad; CPU OS 10_15_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/25.0 Mobile/15E148 Safari/605.1.15
	// ✅ Android: Mozilla/5.0 (Android 10; Mobile; rv:68.0) Gecko/68.0 Firefox/68.0

	// Safari
	// ❎ macOS: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15
	// ❎ iOS:   Mozilla/5.0 (iPad; CPU OS 13_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Mobile/15E148 Safari/604.1

	// WeChat
	// ❎ iOS:     Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_3 like Mac OS X) AppleWebKit/603.3.8 (KHTML, like Gecko) Mobile/14G60 wxwork/2.1.5 MicroMessenger/6.3.22
	// ✅ Windows: Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36 MicroMessenger/6.5.2.501 NetType/WIFI WindowsWechat QBCore/3.43.691.400 QQBrowser/9.0.2524.400
	// ✅ Android: Mozilla/5.0 (Linux; Android 7.0; LG-H831 Build/NRD90U; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/68.0.3440.91 Mobile Safari/537.36 MicroMessenger/6.6.7.1303(0x26060743) NetType/WIFI Language/zh_TW

	UA := c.Get("User-Agent")

	if strings.Contains(UA, "Firefox") || strings.Contains(UA, "Chrome") {
		// Chrome or firefox on macOS Windows
	} else if strings.Contains(UA, "Android") || strings.Contains(UA, "Windows") || strings.Contains(UA, "Linux") {
		// on Android, Windows and Linux
	} else if strings.Contains(UA, "FxiOS") || strings.Contains(UA, "CriOS") {
		//firefox and Chrome on iOS
	} else {
		log.Infof("A Safari user has arrived...%s", UA)
		c.SendFile(RawImageAbs)
		return "", true
	}
	return UA, false
}
