package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func walker() []string {
	var list []string
	_ = filepath.Walk("./pics", func(p string, info os.FileInfo, err error) error {
		if !info.IsDir() && !strings.HasPrefix(path.Base(p), ".") {
			list = append(list, p)
		}
		return nil
	})
	return list
}

func TestWebPEncoder(t *testing.T) {
	// Go through every files
	var dest = "/tmp/test-result"
	var target = walker()
	for _, f := range target {
		runEncoder(t, f, dest)
	}
	_ = os.Remove(dest)
}

func TestAvifEncoder(t *testing.T) {
	// Only one file: img_over_16383px.jpg might cause memory issues on CI environment
	var dest = "/tmp/test-result"
	avifEncoder("./pics/big.jpg", dest, 80)
	assertType(t, dest, "image/avif")
}

func TestNonExistImage(t *testing.T) {
	var dest = "/tmp/test-result"
	_ = webpEncoder("./pics/empty.jpg", dest, 80)
	avifEncoder("./pics/empty.jpg", dest, 80)
}

func TestConvertFail(t *testing.T) {
	var dest = "/tmp/test-result"
	_ = webpEncoder("./pics/webp_server.jpg", dest, -1)
	avifEncoder("./pics/webp_server.jpg", dest, -1)
}

func runEncoder(t *testing.T, file string, dest string) {
	if file == "pics/empty.jpg" {
		t.Log("Empty file, that's okay.")
	}
	_ = webpEncoder(file, dest, 80)
	assertType(t, dest, "image/webp")

}

func assertType(t *testing.T, dest, mime string) {
	data, _ := os.ReadFile(dest)
	types := getFileContentType(data[:512])
	assert.Equalf(t, mime, types, "File %s should be %s", dest, mime)
}
