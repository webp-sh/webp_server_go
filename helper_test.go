package main

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// test this file: go test -v -cover .
func TestGetFileContentType(t *testing.T) {
	var data = []byte("hello")
	var expected = "text/plain; charset=utf-8"
	var result = getFileContentType(data)

	assert.Equalf(t, result, expected, "Result: [%s], Expected: [%s]", result, expected)

}

func TestFileCount(t *testing.T) {
	var data = ".github"
	var expected = 2
	var result = fileCount(data)
	assert.Equalf(t, result, expected, "Result: [%d], Expected: [%d]", result, expected)

}

func TestImageExists(t *testing.T) {
	var data = "./pics/empty.jpg"
	var result = !imageExists(data)

	if result {
		t.Errorf("Result: [%v], Expected: [%v]", result, false)
	}
	data = ".pics/empty2.jpg"
	result = imageExists(data)

	assert.Falsef(t, result, "Result: [%v], Expected: [%v]", result, false)

}

func TestGenWebpAbs(t *testing.T) {
	cwd, cooked := genWebpAbs("./pics/webp_server.png", "/tmp",
		"test", "a")
	if !strings.Contains(cwd, "webp_server_go") {
		t.Logf("Result: [%v], Expected: [%v]", cwd, "webp_server_go")
	}
	var parts = strings.Split(cooked, ".")

	assert.Equalf(t, parts[0], "/tmp/test", "Result: [%v], Expected: [%v]", cooked, "/tmp/test.<ts>.webp")
	assert.Equalf(t, parts[2], "webp", "Result: [%v], Expected: [%v]", cooked, "/tmp/test.<ts>.webp")
}

func TestGenEtag(t *testing.T) {
	var data = "./pics/png.jpg"
	var expected = "W/\"1020764-262C0329\""
	var result = genEtag(data)

	assert.Equalf(t, result, expected, "Result: [%s], Expected: [%s]", result, expected)

	// proxy mode
	proxyMode = true
	config.ImgPath = "https://github.com/webp-sh/webp_server_go/raw/master/"
	remoteRaw = ""
	data = "https://github.com/webp-sh/webp_server_go/raw/master/pics/webp_server.png"
	result = genEtag(data)
	assert.Equal(t, result, "W/\"269387-6FFD6D2D\"")

}

func TestGoOrigin(t *testing.T) {
	// this is a complete test case for webp compatibility
	// func goOrigin(header, ua string) bool
	// UNLESS YOU KNOW WHAT YOU ARE DOING, DO NOT CHANGE THE TEST CASE MAPPING HERE.
	var testCase = map[[2]string]bool{
		// macOS Catalina Safari
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.1 Safari/605.1.15"}: true,
		// macOS Chrome
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.88 Safari/537.36"}: false,
		// iOS14 Safari
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.1 Mobile/15E148 Safari/604.1"}: false,
		// iOS14 Chrome
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.77 Mobile/15E148 Safari/604.1"}: false,
		// iPadOS 14 Safari
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPad; CPU OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.1 Mobile/15E148 Safari/604.1"}: false,
		// iPadOS 14 Chrome
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPad; CPU OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.77 Mobile/15E148 Safari/604.1"}: false,
		// Warning: these three are real capture headers - I don't have iOS/iPadOS prior to version 14
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15"}:                         true,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPad; CPU OS 10_15_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/25.0 Mobile/15E148 Safari/605.1.15"}:            true,
		[2]string{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Mozilla/5.0 (iPhone; CPU iPhone OS 13_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/83.0.4103.63 Mobile/15E148 Safari/604.1"}: true,
	}

	for value, is := range testCase {
		assert.Equalf(t, is, goOrigin(value[0], value[1]), "[%v]:[%s]", value, is)
	}
}

func TestUaOrigin(t *testing.T) {
	// reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent/Firefox
	// https://developer.chrome.com/multidevice/user-agent#chrome_for_ios_user_agent

	// UNLESS YOU KNOW WHAT YOU ARE DOING, DO NOT CHANGE THE TEST CASE MAPPING HERE.
	var testCase = map[string]bool{
		// Chrome on Windows, macOS, linux, iOS and Android
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36":                            false,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36":                      false,
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36":                                      false,
		"Mozilla/5.0 (iPhone; CPU iPhone OS 13_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/83.0.4103.63 Mobile/15E148 Safari/604.1": true,
		"Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.60 Mobile Safari/537.36":                               false,

		// Firefox on Windows, macOS, linux, iOS and Android
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:76.0) Gecko/20100101 Firefox/76.0":                                                     false,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:76.0) Gecko/20100101 Firefox/76.0":                                                 false,
		"Mozilla/5.0 (X11; Linux i686; rv:76.0) Gecko/20100101 Firefox/76.0":                                                                 false,
		"Mozilla/5.0 (iPad; CPU OS 10_15_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/25.0 Mobile/15E148 Safari/605.1.15": true,
		"Mozilla/5.0 (Android 10; Mobile; rv:68.0) Gecko/68.0 Firefox/68.0":                                                                  false,

		// Safari on macOS and iOS
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15":            true,
		"Mozilla/5.0 (iPad; CPU OS 13_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Mobile/15E148 Safari/604.1": true,

		// WeChat on iOS, Windows, and Android
		"Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_3 like Mac OS X) AppleWebKit/603.3.8 (KHTML, like Gecko) Mobile/14G60 wxwork/2.1.5 MicroMessenger/6.3.22":                                                                         true,
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36 MicroMessenger/6.5.2.501 NetType/WIFI WindowsWechat QBCore/3.43.691.400 QQBrowser/9.0.2524.400":              false,
		"Mozilla/5.0 (Linux; Android 7.0; LG-H831 Build/NRD90U; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/68.0.3440.91 Mobile Safari/537.36 MicroMessenger/6.6.7.1303(0x26060743) NetType/WIFI Language/zh_TW": false,

		// IE
		"Mozilla/5.0 (Windows NT 6.1; WOW64; Trident/7.0; AS; rv:11.0) like Gecko": true,
		"Mozilla/5.0 (compatible; MSIE 10.0; Windows NT 6.1; WOW64; Trident/6.0)":  true,

		// Others
		"PostmanRuntime/7.26.1": true,
		"curl/7.64.1":           true,

		// these four are captured from iOS14/iPadOS14, which supports webp
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.1 Mobile/15E148 Safari/604.1":     false,
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.77 Mobile/15E148 Safari/604.1\n": false,
		"Mozilla/5.0 (iPad; CPU OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.1 Mobile/15E148 Safari/604.1":                false,
		"Mozilla/5.0 (iPad; CPU OS 14_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/87.0.4280.77 Mobile/15E148 Safari/604.1":            false,
	}

	for browser, is := range testCase {
		assert.Equalf(t, is, uaOrigin(browser), "[%v]:[%s]", is, browser)
	}

}

func TestHeaderOrigin(t *testing.T) {
	// Chrome: Accept: image/avif,image/webp,image/apng,image/*,*/*;q=0.8
	// Safari 版本14.0.1 (15610.2.11.51.10, 15610): Accept: image/png,image/svg+xml,image/*;q=0.8,video/*;q=0.8,*/*;q=0.5
	// Safari Technology Preview Release 116 (Safari 14.1, WebKit 15611.1.5.3) Accept: image/png,image/svg+xml,image/*;q=0.8,video/*;q=0.8,*/*;q=0.5

	// UNLESS YOU KNOW WHAT YOU ARE DOING, DO NOT CHANGE THE TEST CASE MAPPING HERE.
	var testCase = map[string]bool{
		"image/avif,image/webp,image/apng,image/*,*/*;q=0.8": false,
		"*/*": true,
		"image/png,image/svg+xml,image/*;q=0.8,video/*;q=0.8,*/*;q=0.5": true,
		"I don't know what it is:-)":                                    true,
	}
	for header, is := range testCase {
		assert.Equalf(t, is, headerOrigin(header), "[%v]:[%s]", is, header)
	}
}

func TestChanErr(t *testing.T) {
	var value = 2
	var testC = make(chan int, 2)
	testC <- value
	chanErr(testC)
	value = <-testC
	assert.Equal(t, 2, value)
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
	url := "http://github.com/favicon.ico"
	err := fetchRemoteImage(fp, url)
	assert.Equal(t, err, nil)
	data, _ := ioutil.ReadFile(fp)
	assert.Equal(t, "image/x-icon", getFileContentType(data))

	// test can't create file
	err = fetchRemoteImage("/", url)
	assert.NotNil(t, err)

	// test bad url
	err = fetchRemoteImage(fp, "http://ahjdsgdsghja.cya")
	assert.NotNil(t, err)
}

func TestCleanProxyCache(t *testing.T) {
	// test normal situation
	fp := filepath.Join("./exhaust", "sample.png.12345.webp")
	_ = ioutil.WriteFile(fp, []byte("1234"), 0755)
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
