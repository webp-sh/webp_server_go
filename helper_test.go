package main

import (
	"strings"
	"testing"
)

// test this file: go test -v -cover helper_test.go helper.go
// test one function: go test -run TestGetFileContentType helper_test.go helper.go -v
func TestGetFileContentType(t *testing.T) {
	var data = []byte("hello")
	var expected = "text/plain; charset=utf-8"
	var result = GetFileContentType(data)

	if result != expected {
		t.Errorf("Result: [%s], Expected: [%s]", result, expected)
	}

}

// TODO: make a universal logging function
func TestFileCount(t *testing.T) {
	var data = ".github"
	var expected = 2
	var result = FileCount(data)

	if result != expected {
		t.Errorf("Result: [%d], Expected: [%d]", result, expected)
	}
}

func TestImageExists(t *testing.T) {
	var data = "./pics/empty.jpg"
	var result = !ImageExists(data)

	if result {
		t.Errorf("Result: [%v], Expected: [%v]", result, false)
	}
	data = ".pics/empty2.jpg"
	result = ImageExists(data)

	if result {
		t.Errorf("Result: [%v], Expected: [%v]", result, false)
	}

}

func TestGenWebpAbs(t *testing.T) {
	cwd, cooked := GenWebpAbs("./pics/webp_server.png", "/tmp",
		"test", "a")
	if !strings.Contains(cwd, "webp_server_go") {
		t.Logf("Result: [%v], Expected: [%v]", cwd, "webp_server_go")
	}
	var parts = strings.Split(cooked, ".")
	if parts[0] != "/tmp/test" || parts[2] != "webp" {
		t.Errorf("Result: [%v], Expected: [%v]", cooked, "/tmp/test.<ts>.webp")

	}
}

func TestGenEtag(t *testing.T) {
	var data = "./pics/png.jpg"
	var expected = "W/\"1020764-262C0329\""
	var result = GenEtag(data)

	if result != expected {
		t.Errorf("Result: [%s], Expected: [%s]", result, expected)
	}
}


func TestGoOrigin(t *testing.T) {
	// reference: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent/Firefox
	// https://developer.chrome.com/multidevice/user-agent#chrome_for_ios_user_agent

	var testCase = map[string]bool{
		// Chrome on Windows, macOS, linux, iOS and Android
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36":                            false,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36":                      false,
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.61 Safari/537.36":                                      false,
		"Mozilla/5.0 (iPhone; CPU iPhone OS 13_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/83.0.4103.63 Mobile/15E148 Safari/604.1": false,
		"Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.60 Mobile Safari/537.36":                               false,

		// Firefox on Windows, macOS, linux, iOS and Android
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:76.0) Gecko/20100101 Firefox/76.0":                                                     false,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:76.0) Gecko/20100101 Firefox/76.0":                                                 false,
		"Mozilla/5.0 (X11; Linux i686; rv:76.0) Gecko/20100101 Firefox/76.0":                                                                 false,
		"Mozilla/5.0 (iPad; CPU OS 10_15_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/25.0 Mobile/15E148 Safari/605.1.15": false,
		"Mozilla/5.0 (Android 10; Mobile; rv:68.0) Gecko/68.0 Firefox/68.0":                                                                  false,

		// Safari on macOS and iOS
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1 Safari/605.1.15":            true,
		"Mozilla/5.0 (iPad; CPU OS 13_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Mobile/15E148 Safari/604.1": true,

		// WeChat on iOS, Windows, and Android
		"Mozilla/5.0 (iPhone; CPU iPhone OS 10_3_3 like Mac OS X) AppleWebKit/603.3.8 (KHTML, like Gecko) Mobile/14G60 wxwork/2.1.5 MicroMessenger/6.3.22":                                                                         true,
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36 MicroMessenger/6.5.2.501 NetType/WIFI WindowsWechat QBCore/3.43.691.400 QQBrowser/9.0.2524.400":              false,
		"Mozilla/5.0 (Linux; Android 7.0; LG-H831 Build/NRD90U; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/68.0.3440.91 Mobile Safari/537.36 MicroMessenger/6.6.7.1303(0x26060743) NetType/WIFI Language/zh_TW": false,
	}

	for browser, is := range testCase {

		if is != goOrigin(browser) {
			t.Errorf("[%v]:[%s]", is, browser)
		}
	}

}
