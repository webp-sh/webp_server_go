package main

import (
	"fmt"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"

	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// test all files: go test -v -cover .
// test one case: go test -v -run TestSelectFormat

func TestGetFileContentType(t *testing.T) {
	var data = []byte("remember remember the 5th of november")
	var expected = ""
	var result = getFileContentType(data)
	assert.Equalf(t, result, expected, "Result: [%s], Expected: [%s]", result, expected)
}

func TestFileCount(t *testing.T) {
	var data = "scripts"
	var expected int64 = 2
	var result = fileCount(data)
	assert.Equalf(t, result, expected, "Result: [%d], Expected: [%d]", result, expected)
}

func TestImageExists(t *testing.T) {
	var data = "./pics/empty.jpg"
	var result = imageExists(data)

	if result {
		t.Errorf("Result: [%v], Expected: [%v]", result, false)
	}
	data = ".pics/empty2.jpg"
	result = imageExists(data)

	assert.Falsef(t, result, "Result: [%v], Expected: [%v]", result, false)

}

func TestGenOptimizedAbsPath(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "test_image.*")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Set the modification time for the temporary file
	modTime := time.Now()
	if err := os.Chtimes(tempFile.Name(), modTime, modTime); err != nil {
		t.Fatalf("Failed to set modification time for the temporary file: %v", err)
	}

	rawImagePath := tempFile.Name()
	exhaustPath := "/path/to/exhaust"
	imageName := "tsuki.jpg"
	reqURI := "/path/to/tsuki.jpg"
	extraParams := ExtraParams{Width: 200, Height: 0}

	// Test if config.EnableExtraParams is false
	config.EnableExtraParams = false

	avifAbsolutePath, webpAbsolutePath := genOptimizedAbsPath(rawImagePath, exhaustPath, imageName, reqURI, extraParams)

	expectedAvifPath := path.Clean(path.Join(exhaustPath, path.Dir(reqURI), fmt.Sprintf("%s.%d.avif", imageName, modTime.Unix())))
	expectedWebpPath := path.Clean(path.Join(exhaustPath, path.Dir(reqURI), fmt.Sprintf("%s.%d.webp", imageName, modTime.Unix())))

	if avifAbsolutePath != expectedAvifPath {
		t.Errorf("Avif absolute path is incorrect. Expected: %s, Got: %s", expectedAvifPath, avifAbsolutePath)
	}
	if webpAbsolutePath != expectedWebpPath {
		t.Errorf("Webp absolute path is incorrect. Expected: %s, Got: %s", expectedWebpPath, webpAbsolutePath)
	}

	// Test if config.EnableExtraParams is true and extraParams is not 0
	config.EnableExtraParams = true

	avifAbsolutePath, webpAbsolutePath = genOptimizedAbsPath(rawImagePath, exhaustPath, imageName, reqURI, extraParams)

	expectedAvifPath = path.Clean(path.Join(exhaustPath, path.Dir(reqURI), fmt.Sprintf("%s.%d.avif_width=%d&height=%d", imageName, modTime.Unix(), extraParams.Width, extraParams.Height)))
	expectedWebpPath = path.Clean(path.Join(exhaustPath, path.Dir(reqURI), fmt.Sprintf("%s.%d.webp_width=%d&height=%d", imageName, modTime.Unix(), extraParams.Width, extraParams.Height)))

	if avifAbsolutePath != expectedAvifPath {
		t.Errorf("Avif absolute path is incorrect. Expected: %s, Got: %s", expectedAvifPath, avifAbsolutePath)
	}
	if webpAbsolutePath != expectedWebpPath {
		t.Errorf("Webp absolute path is incorrect. Expected: %s, Got: %s", expectedWebpPath, webpAbsolutePath)
	}

	// Test if config.EnableExtraParams is true and extraParams is 0
	config.EnableExtraParams = true
	extraParams = ExtraParams{Width: 200, Height: 0}

	avifAbsolutePath, webpAbsolutePath = genOptimizedAbsPath(rawImagePath, exhaustPath, imageName, reqURI, extraParams)

	expectedAvifPath = path.Clean(path.Join(exhaustPath, path.Dir(reqURI), fmt.Sprintf("%s.%d.avif_width=%d&height=%d", imageName, modTime.Unix(), extraParams.Width, extraParams.Height)))
	expectedWebpPath = path.Clean(path.Join(exhaustPath, path.Dir(reqURI), fmt.Sprintf("%s.%d.webp_width=%d&height=%d", imageName, modTime.Unix(), extraParams.Width, extraParams.Height)))

	if avifAbsolutePath != expectedAvifPath {
		t.Errorf("Avif absolute path is incorrect. Expected: %s, Got: %s", expectedAvifPath, avifAbsolutePath)
	}
	if webpAbsolutePath != expectedWebpPath {
		t.Errorf("Webp absolute path is incorrect. Expected: %s, Got: %s", expectedWebpPath, webpAbsolutePath)
	}
}

func TestSelectFormat(t *testing.T) {
	// this is a complete test case for webp compatibility
	// func goOrigin(header, ua string) bool
	// UNLESS YOU KNOW WHAT YOU ARE DOING, DO NOT CHANGE THE TEST CASE MAPPING HERE.
	var fullSupport = []string{"avif", "webp", "raw"}
	var webpSupport = []string{"webp", "raw"}
	var jpegSupport = []string{"raw"}
	var testCase = map[[2]string][]string{
		// Latest Chrome on Windows, macOS, linux, Android and iOS 13
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36"}:        fullSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36"}:  fullSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36"}:                  fullSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9", "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.60 Mobile Safari/537.36"}:           fullSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9", "Mozilla/5.0 (Linux; Android 6.0; HTC M8t) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/95.0.4638.74 Mobile Safari/537.36"}: fullSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8", "HTCM8t_LTE/1.0 Android/4.4 release/2013 Browser/WAP2.0 Profile/MIDP-2.0 Configuration/CLDC-1.1 AppleWebKit/534.30"}:                                                                      webpSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/83.0.4103.63 Mobile/15E148 Safari/604.1"}:                                                     jpegSupport,

		// macOS Catalina Safari
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.1 Safari/605.1.15"}: jpegSupport,

		// iOS14 Safari and Chrome
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.1 Mobile/15E148 Safari/604.1"}:   webpSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.77 Mobile/15E148 Safari/604.1"}: webpSupport,

		// iPadOS 14 Safari and Chrome
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPad; CPU OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.1 Mobile/15E148 Safari/604.1"}:     webpSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPad; CPU OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.77 Mobile/15E148 Safari/604.1"}: webpSupport,

		// iOS 15 Safari, Firefox and Chrome
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.1 Mobile/15E148 Safari/604.1"}:     webpSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/39.0  Mobile/15E148 Safari/605.1.15"}:   webpSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/96.0.4664.53 Mobile/15E148 Safari/604.1"}: webpSupport,

		// IE
		[2]string{"application/x-ms-application, image/jpeg, application/xaml+xml, image/gif, image/pjpeg, application/x-ms-xbap, */*", "Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; AS; rv:11.0) like Gecko"}: jpegSupport,
		// Others
		[2]string{"", "PostmanRuntime/7.26.1"}:            jpegSupport,
		[2]string{"", "curl/7.64.1"}:                      jpegSupport,
		[2]string{"image/webp", "curl/7.64.1"}:            webpSupport,
		[2]string{"image/avif,image/webp", "curl/7.64.1"}: fullSupport,

		// some weird browsers
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 15_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.16(0x18001033) NetType/WIFI Language/zh_CN"}:                                                                                                                             webpSupport,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8", "Mozilla/5.0 (Linux; Android 6.0; HTC M8t Build/MRA58K; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/45.0.2454.95 Mobile Safari/537.36 MMWEBID/4285 MicroMessenger/8.0.15.2001(0x28000F41) Process/tools WeChat/arm32 Weixin GPVersion/1 NetType/WIFI Language/zh_CN ABI/arm32"}: webpSupport,
	}
	for browser, expected := range testCase {
		var header fasthttp.RequestHeader
		header.Set("accept", browser[0])
		header.Set("user-agent", browser[1])
		guessed := guessSupportedFormat(&header)

		sort.Strings(expected)
		sort.Strings(guessed)
		log.Infof("%s expected%s --- actual %s", browser, expected, guessed)
		assert.Equal(t, expected, guessed)
	}

}

func TestGetRemoteImageInfo(t *testing.T) {
	url := "https://github.com/favicon.ico"
	statusCode, etag, length := getRemoteImageInfo(url)
	assert.NotEqual(t, "", etag)
	assert.NotEqual(t, "0", length)
	assert.Equal(t, statusCode, http.StatusOK)

	// test non-exist url
	url = "http://sdahjajda.com"
	statusCode, etag, length = getRemoteImageInfo(url)
	assert.Equal(t, "", etag)
	assert.Equal(t, "", length)
	assert.Equal(t, statusCode, http.StatusInternalServerError)
}

func TestFetchRemoteImage(t *testing.T) {
	// test the normal one
	fp := filepath.Join("./exhaust", "test.ico")

	err := fetchRemoteImage(fp, "http://github.com/favicon.ico")
	assert.Equal(t, err, nil)
	data, _ := os.ReadFile(fp)
	assert.Equal(t, "image/vnd.microsoft.icon", getFileContentType(data))

	// test can't create file
	err = fetchRemoteImage("/", "http://github.com/favicon.ico")
	assert.NotNil(t, err)

	// test bad url
	err = fetchRemoteImage(fp, "http://ahjdsgdsghja.cya")
	assert.NotNil(t, err)

}

func TestCleanProxyCache(t *testing.T) {
	// test normal situation
	fp := filepath.Join("./exhaust", "sample.png.12345.webp")
	buf := make([]byte, 0x1000)
	_ = os.WriteFile(fp, buf, 0755)
	assert.True(t, imageExists(fp))
	cleanProxyCache(fp)
	assert.False(t, imageExists(fp))

	// test bad dir
	cleanProxyCache("/aasdyg/dhj2/dagh")
}

func TestGetCompressionRate(t *testing.T) {
	pic1 := "pics/webp_server.bmp"
	pic2 := "pics/webp_server.jpg"
	var ratio string

	ratio = getCompressionRate(pic1, pic2)
	assert.Equal(t, "0.16", ratio)

	ratio = getCompressionRate(pic1, "pic2")
	assert.Equal(t, "", ratio)

	ratio = getCompressionRate("pic1", pic2)
	assert.Equal(t, "", ratio)
}
