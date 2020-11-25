package main

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

//go test -v -cover .
func TestWebpEncoder(t *testing.T) {
	var webp = "/tmp/test-result.webp"
	var target = walker()

	for _, f := range target {
		//fmt.Println(b, c, webp)
		runEncoder(t, f, webp)
	}
	_ = os.Remove(webp)

	// test error
	err := webpEncoder("./pics/empty.jpg", webp, 80, true, nil)
	assert.NotNil(t, err)
	_ = os.Remove(webp)
}

func TestNonImage(t *testing.T) {
	var webp = "/tmp/test-result.webp"
	// test error
	var err = webpEncoder("./pics/empty.jpg", webp, 80, true, nil)
	assert.NotNil(t, err)
	_ = os.Remove(webp)

}

func walker() []string {
	var list []string
	_ = filepath.Walk("./pics", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			list = append(list, path)
		}
		return nil
	})
	return list
}

func runEncoder(t *testing.T, file string, webp string) {
	var c chan int
	//t.Logf("convert from %s to %s", file, webp)
	var err = webpEncoder(file, webp, 80, true, c)
	if file == "pics/empty.jpg" && err != nil {
		t.Log("Empty file, that's okay.")
	} else if err != nil {
		t.Fatalf("Fatal, convert failed for %s: %v ", file, err)
	}

	data, _ := ioutil.ReadFile(webp)
	types := getFileContentType(data[:512])
	if types != "image/webp" {
		t.Fatal("Fatal, file type is wrong!")
	}

}
