package main

import (
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/stretchr/testify/assert"

	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

var dest = "/tmp/test-result"

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

func TestResizeImage(t *testing.T) {
	// Create a test image with specific dimensions
	testImage, _ := vips.NewImageFromFile("./pics/png.jpg")

	// Test case 1: Both width and height are greater than 0
	params1 := ExtraParams{
		Width:  100,
		Height: 100,
	}
	err := resizeImage(testImage, params1)
	if err != nil {
		t.Errorf("Error occurred while resizing image: %v", err)
	}

	// Assert the resized image has the expected dimensions
	resizedWidth1 := testImage.Width()
	resizedHeight1 := testImage.Height()
	expectedWidth1 := params1.Width
	expectedHeight1 := params1.Height
	// If both width and height are provided, follow Width and keep aspect ratio
	if resizedWidth1 != expectedWidth1 {
		t.Errorf("Resized image dimensions do not match. Expected: %dx%d, Actual: %dx%d",
			expectedWidth1, expectedHeight1, resizedWidth1, resizedHeight1)
	}

	// Test case 2: Only width is greater than 0
	params2 := ExtraParams{
		Width:  100,
		Height: 0,
	}
	err = resizeImage(testImage, params2)
	if err != nil {
		t.Errorf("Error occurred while resizing image: %v", err)
	}

	// Assert the resized image has the expected width
	resizedWidth2 := testImage.Width()
	expectedWidth2 := params2.Width
	if resizedWidth2 != expectedWidth2 {
		t.Errorf("Resized image width does not match. Expected: %d, Actual: %d",
			expectedWidth2, resizedWidth2)
	}

	// Test case 3: Only height is greater than 0
	params3 := ExtraParams{
		Width:  0,
		Height: 100,
	}
	err = resizeImage(testImage, params3)
	if err != nil {
		t.Errorf("Error occurred while resizing image: %v", err)
	}

	// Assert the resized image has the expected height
	resizedHeight3 := testImage.Height()
	expectedHeight3 := params3.Height
	if resizedHeight3 != expectedHeight3 {
		t.Errorf("Resized image height does not match. Expected: %d, Actual: %d",
			expectedHeight3, resizedHeight3)
	}

}

func TestWebPEncoder(t *testing.T) {
	// Go through every files
	var target = walker()
	for _, f := range target {
		runEncoder(t, f, dest)
	}
	_ = os.Remove(dest)
}

func TestAnimatedGIFWithWebPEncoder(t *testing.T) {
	runEncoder(t, "./pics/gif-animated.gif", dest)
	_ = os.Remove(dest)
}

func TestAvifEncoder(t *testing.T) {
	// Only one file: img_over_16383px.jpg might cause memory issues on CI environment
	assert.Nil(t, avifEncoder("./pics/big.jpg", dest, 80, ExtraParams{Width: 0, Height: 0}))
	assertType(t, dest, "image/avif")
}

func TestNonExistImage(t *testing.T) {
	assert.NotNil(t, webpEncoder("./pics/empty.jpg", dest, 80, ExtraParams{Width: 0, Height: 0}))
	assert.NotNil(t, avifEncoder("./pics/empty.jpg", dest, 80, ExtraParams{Width: 0, Height: 0}))
}

func TestHighResolutionImage(t *testing.T) {
	assert.NotNil(t, webpEncoder("./pics/img_over_16383px.jpg", dest, 80, ExtraParams{Width: 0, Height: 0}))
	assert.Nil(t, avifEncoder("./pics/img_over_16383px.jpg", dest, 80, ExtraParams{Width: 0, Height: 0}))
}

func runEncoder(t *testing.T, file string, dest string) {
	if file == "pics/empty.jpg" {
		t.Log("Empty file, that's okay.")
	}
	_ = webpEncoder(file, dest, 80, ExtraParams{Width: 0, Height: 0})
	assertType(t, dest, "image/webp")

}

func assertType(t *testing.T, dest, mime string) {
	data, _ := os.ReadFile(dest)
	types := getFileContentType(data[:512])
	assert.Equalf(t, mime, types, "File %s should be %s", dest, mime)
}
