package handler

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"webp_server_go/config"
	"webp_server_go/helper"

	"github.com/patrickmn/go-cache"
)

func TestFetchRemoteImgSingleflight(t *testing.T) {
	tmpDir := t.TempDir()
	oldRaw := config.Config.RemoteRawPath
	oldExhaust := config.Config.ExhaustPath
	oldMeta := config.Config.MetadataPath
	oldCache := config.RemoteCache
	oldAllowAll := config.AllowAllExtensions
	t.Cleanup(func() {
		config.Config.RemoteRawPath = oldRaw
		config.Config.ExhaustPath = oldExhaust
		config.Config.MetadataPath = oldMeta
		config.RemoteCache = oldCache
		config.AllowAllExtensions = oldAllowAll
	})

	config.Config.RemoteRawPath = filepath.Join(tmpDir, "remote-raw")
	config.Config.ExhaustPath = filepath.Join(tmpDir, "exhaust")
	config.Config.MetadataPath = filepath.Join(tmpDir, "metadata")
	config.RemoteCache = cache.New(cache.NoExpiration, 10*time.Minute)
	config.AllowAllExtensions = false

	pngBytes := tinyPNG(t)
	var gets atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			gets.Add(1)
			time.Sleep(150 * time.Millisecond)
		}
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write(pngBytes)
	}))
	t.Cleanup(srv.Close)

	imgURL := srv.URL + "/pic.png"
	const subdir = "remote.test"
	const callers = 8

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(callers)
	for range callers {
		go func() {
			defer wg.Done()
			<-start
			meta := fetchRemoteImg(imgURL, subdir)
			if meta.Id == "" {
				t.Errorf("fetchRemoteImg returned empty metadata id")
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := gets.Load(); got != 1 {
		t.Fatalf("concurrent fetches issued %d GETs, want 1", got)
	}

	localPath := filepath.Join(config.Config.RemoteRawPath, subdir, helper.HashString(imgURL)+".png")
	info, err := os.Stat(localPath)
	if err != nil {
		t.Fatalf("downloaded file missing: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("downloaded file is empty")
	}

	// A later request for the same URL should reuse the file.
	_ = fetchRemoteImg(imgURL, subdir)
	if got := gets.Load(); got != 1 {
		t.Fatalf("cached fetch issued %d GETs, want 1", got)
	}

	// A different URL is a different flight and downloads once.
	_ = fetchRemoteImg(srv.URL+"/other.png", subdir)
	if got := gets.Load(); got != 2 {
		t.Fatalf("second URL issued %d GETs, want 2", got)
	}
}

func tinyPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}
