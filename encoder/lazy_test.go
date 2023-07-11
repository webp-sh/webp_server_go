package encoder

import (
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	// "github.com/stretchr/testify/assert"

	"webp_server_go/config"

	pond "github.com/alitto/pond"
	pq "github.com/emirpasic/gods/queues/priorityqueue"
	hs "github.com/emirpasic/gods/sets/hashset"
	"github.com/stretchr/testify/assert"
)

func LazyModeSetup(t *testing.T) func() {
	// Setup
	VipsSetupForTests(t)
	config.LazyMode = true
	config.VerboseMode = true
	DefaultWorkQueue = pq.NewWith(byPriority) // empty
	HeavyWorkQueue = pq.NewWith(byPriority)   // empty
	WorkOngoingSet = hs.New()

	// Create a buffered (non-blocking) pool that can scale up to runtime.NumCPU() workers
	// and has a buffer capacity of 1000 tasks
	DefaultWorkerPool = pond.New(runtime.NumCPU(), 1000)
	HeavyWorkerPool = pond.New(config.MaxHeavyJobs, 1000)

	exhaustPath, _ := os.MkdirTemp("", "tests-")
	config.Config.ExhaustPath = exhaustPath

	return func() {
		// Tear down
		config.LazyMode = false
		DefaultWorkerPool.StopAndWaitFor(15 * time.Second)
		HeavyWorkerPool.StopAndWaitFor(15 * time.Second)
		WorkOngoingSet.Clear()
		os.RemoveAll(config.Config.ExhaustPath)
	}
}

func TestLazyConvertImage(t *testing.T) {
	ts := LazyModeSetup(t)
	t.Cleanup(ts)

	config.Jobs = 1
	config.Config.ImgPath = "../pics"

	extraParams := config.ExtraParams{
		Width:  0,
		Height: 0,
	}

	convertImage("../pics/webp_server.png", path.Join(config.Config.ExhaustPath, "webp_server.webp"), "webp", extraParams)
	assert.Equal(t, DefaultWorkQueue.Size(), 1)
	lazyDo()
	DefaultWorkerPool.StopAndWait()
	assert.Equal(t, DefaultWorkQueue.Size(), 0)
}
