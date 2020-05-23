package main

import (
	"strings"
	"testing"
)

// test this file: go test helper_test.go helper.go -v
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
		t.Errorf("Result: [%v], Expected: [%v]", result, true)
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
