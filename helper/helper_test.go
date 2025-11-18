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

	t.Run("test file", func(t *testing.T) {
		assert.True(t, ImageExists("./helper_test.go"))
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
			name:      "WebP/AVIF/JXL Supported",
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
				"heic": true,
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

func TestHasResizeParams(t *testing.T) {
	if HasResizeParams(config.ExtraParams{}) {
		t.Fatalf("expected empty params to report no resize request")
	}

	params := config.ExtraParams{Width: 100}
	if !HasResizeParams(params) {
		t.Fatalf("expected Width to trigger resize detection")
	}

	params = config.ExtraParams{MaxHeight: 80}
	if !HasResizeParams(params) {
		t.Fatalf("expected MaxHeight to trigger resize detection")
	}
}

func TestValidateNoUpscale(t *testing.T) {
	meta := config.ImageMeta{Width: 200, Height: 100}

	tests := []struct {
		params    config.ExtraParams
		wantError bool
	}{
		// Tests for no resize params 
		{
			params:    config.ExtraParams{},
			wantError: false,
		},
		// Tests for shrink Width only
		{
			params:    config.ExtraParams{Width: 150},
			wantError: false,
		},
		// Tests for shrink Height only
		{
			params:    config.ExtraParams{Height: 80},
			wantError: false,
		},
		// Tests for shrink both Width and Height
		{
			params:    config.ExtraParams{Width: 150, Height: 80},
			wantError: false,
		},
		// Tests for upscale Width only
		{
			params:    config.ExtraParams{Width: 250},
			wantError: true,
		},
		// Tests for upscale Height only
		{
			params:    config.ExtraParams{Height: 150},
			wantError: true,
		},
		// Tests for upscale both Width and Height
		{
			params:    config.ExtraParams{Width: 300, Height: 200},
			wantError: true,
		},
		// Tests for MaxWidth and MaxHeight 
		{
			params:    config.ExtraParams{MaxWidth: 300, MaxHeight: 400},
			wantError: false,
		},
	}

	for _, tc := range tests {
		err := ValidateNoUpscale(meta, tc.params)
		if tc.wantError && err == nil {
			t.Fatalf("TestValidateNoUpscale failed with error %s", err)
		}
		if !tc.wantError && err != nil {
			t.Fatalf("TestValidateNoUpscale failed with error %s", err)
		}
	}
}