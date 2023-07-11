package encoder

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"webp_server_go/config"
	"webp_server_go/helper"

	log "github.com/sirupsen/logrus"
)

func TestPrefetchImages(t *testing.T) {
	config.LazyMode = false
	exhaustPath, _ := os.MkdirTemp("", "tests-")
	config.Config.ExhaustPath = exhaustPath
	defer os.RemoveAll(exhaustPath)

	config.Config.ImgPath = "../pics/dir1"
	PrefetchImages()

	count := helper.FileCount(exhaustPath)
	assert.Equal(t, int64(1), count)
}

func TestPrefetchImagesLazy(t *testing.T) {
	ts := LazyModeSetup(t)
	t.Cleanup(ts)
	log.SetLevel(log.DebugLevel)
	config.Jobs = 1
	config.Config.ImgPath = "../pics/dir1"

	PrefetchImages()

	// Wait until the queues are filled
	for start := time.Now(); time.Since(start) < 15*time.Second; {
		time.Sleep(500 * time.Millisecond)
		if DefaultWorkQueue.Size() > 0 {
			break
		}
	}

	// Launch jobs
	lazyDo()

	// Wait until the work is done
	DefaultWorkerPool.StopAndWait()
	HeavyWorkerPool.StopAndWait()

	count := helper.FileCount(config.Config.ExhaustPath)
	assert.Equal(t, int64(1), count)
}

// func TestBadPrefetch(t *testing.T) {
// 	exhaustPath, _ := os.MkdirTemp("", "tests")
// 	config.Jobs = 1
// 	config.Config.ImgPath = "../pics2"
// 	config.Config.ExhaustPath = exhaustPath
// 	PrefetchImages()
// }
