package helper

import (
	"testing"
	"webp_server_go/config"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestMain(m *testing.M) {
	config.ConfigPath = "../config.json"
	config.LoadConfig()
	m.Run()
	config.ConfigPath = "config.json"
}

func TestFileCount(t *testing.T) {
	// test helper dir
	count := FileCount("./")
	assert.Equal(t, int64(4), count)
}

func TestImageExists(t *testing.T) {
	t.Run("file not exists", func(t *testing.T) {
		assert.False(t, ImageExists("dgyuaikdsa"))
	})

	// TODO: how to test lock?

	t.Run("test dir", func(t *testing.T) {
		assert.False(t, ImageExists("/tmp"))
	})

	t.Run("test non image file", func(t *testing.T) {
		assert.False(t, ImageExists("./helper_test.go"))
	})

	t.Run("test image file", func(t *testing.T) {
		assert.True(t, ImageExists("../pics/big.jpg"))
	})

	t.Run("test empty image file", func(t *testing.T) {
		assert.False(t, ImageExists("../pics/empty.jpg"))
	})

	t.Run("test broken image file", func(t *testing.T) {
		assert.True(t, ImageExists("../pics/invalid.png"))
	})

	t.Run("test heic image file", func(t *testing.T) {
		assert.True(t, ImageExists("../pics/sample3.heic"))
	})
}

func TestCheckAllowedExtension(t *testing.T) {
	t.Run("not allowed type", func(t *testing.T) {
		assert.False(t, CheckAllowedExtension("./helper_test.go"))
	})

	t.Run("allowed type", func(t *testing.T) {
		assert.True(t, CheckAllowedExtension("test.jpg"))
	})
}

func TestGuessSupportedFormat(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		accept    string
		expected  map[string]bool
	}{
		{
			name:      "WebP and AVIF explicitly accepted",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15", // iPad
			accept:    "image/webp, image/avif",
			expected: map[string]bool{
				"jpg":  true,
				"jpeg": true,
				"png":  true,
				"gif":  true,
				"svg":  true,
				"bmp":  true,
				"webp": true,
				"avif": true,
				"jxl":  false,
				"nef":  false,
				"heic": false,
			},
		},
		{
			name:      "WebP Supported",
			userAgent: "iPhone OS 15",
			accept:    "image/webp, image/png",
			expected: map[string]bool{
				"jpg":  true,
				"jpeg": true,
				"png":  true,
				"gif":  true,
				"svg":  true,
				"bmp":  true,
				"webp": true,
				"avif": false,
				"jxl":  false,
				"nef":  false,
				"heic": false,
			},
		},
		{
			name:      "WebP/AVIF Supported",
			userAgent: "iPhone OS 16",
			accept:    "image/webp, image/png",
			expected: map[string]bool{
				"jpg":  true,
				"jpeg": true,
				"png":  true,
				"gif":  true,
				"svg":  true,
				"bmp":  true,
				"webp": true,
				"avif": false,
				"jxl":  false,
				"nef":  false,
				"heic": false,
			},
		},
		{
			name:      "Both Supported",
			userAgent: "iPhone OS 16",
			accept:    "image/webp, image/avif",
			expected: map[string]bool{
				"jpg":  true,
				"jpeg": true,
				"png":  true,
				"gif":  true,
				"svg":  true,
				"bmp":  true,
				"webp": true,
				"avif": true,
				"jxl":  false,
				"nef":  false,
				"heic": false,
			},
		},
		{
			name:      "No Supported Formats",
			userAgent: "Unknown OS",
			accept:    "image/jpeg, image/gif",
			expected: map[string]bool{
				"jpg":  true,
				"jpeg": true,
				"png":  true,
				"gif":  true,
				"svg":  true,
				"bmp":  true,
				"webp": false,
				"avif": false,
				"jxl":  false,
				"nef":  false,
				"heic": false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header := &fasthttp.RequestHeader{}
			header.Set("user-agent", test.userAgent)
			header.Set("accept", test.accept)

			result := GuessSupportedFormat(header)

			if len(result) != len(test.expected) {
				t.Errorf("Expected %v, but got %v", test.expected, result)
			}

			for k, v := range test.expected {
				assert.Equal(t, v, result[k])
			}
		})
	}
}

func TestGuessSupportedFormatAcceptNegotiation(t *testing.T) {
	tests := []struct {
		name     string
		accept   string
		expected map[string]bool
	}{
		{
			name:   "explicit modern formats",
			accept: "image/webp, image/avif;q=0.8, image/jxl;q=0.5",
			expected: map[string]bool{
				"webp": true,
				"avif": true,
				"jxl":  true,
			},
		},
		{
			name:   "zero quality means not acceptable",
			accept: "image/webp;q=0, image/avif;q=0.0, image/jxl;q=0",
			expected: map[string]bool{
				"webp": false,
				"avif": false,
				"jxl":  false,
			},
		},
		{
			name:   "wildcards do not advertise modern formats",
			accept: "image/*;q=0.9, */*;q=0.8, image/jxl;q=0",
			expected: map[string]bool{
				"webp": false,
				"avif": false,
				"jxl":  false,
			},
		},
		{
			name:   "media types are case insensitive",
			accept: "IMAGE/WEBP;Q=0.4, IMAGE/JXL",
			expected: map[string]bool{
				"webp": true,
				"avif": false,
				"jxl":  true,
			},
		},
		{
			name:   "invalid quality is ignored",
			accept: "image/webp;q=2, image/avif;q=invalid, image/jxl;q=-1",
			expected: map[string]bool{
				"webp": false,
				"avif": false,
				"jxl":  false,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			header := &fasthttp.RequestHeader{}
			header.Set("Accept", test.accept)

			result := GuessSupportedFormat(header)
			for format, expected := range test.expected {
				assert.Equal(t, expected, result[format], format)
			}
		})
	}
}

func TestGuessSupportedFormatIgnoresUserAgent(t *testing.T) {
	header := &fasthttp.RequestHeader{}
	header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4 like Mac OS X) Version/17.4 Mobile/15E148 Safari/604.1 Firefox/133.0")
	header.Set("Accept", "image/jpeg,image/png,image/*;q=0.8,*/*;q=0.5")

	result := GuessSupportedFormat(header)

	assert.False(t, result["webp"])
	assert.False(t, result["avif"])
	assert.False(t, result["jxl"])
	assert.False(t, result["heic"])
}
