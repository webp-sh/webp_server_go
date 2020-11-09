package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

//go test -v -cover encoder_test.go encoder.go helper.go
func TestWebpEncoder(t *testing.T) {

	var webp = "/tmp/test-result.webp"
	var target = walker()

	for _, f := range target {
		//fmt.Println(b, c, webp)
		runEncoder(t, f, webp)
	}
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
	var err = webpEncoder(file, webp, 80, false, c)
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
