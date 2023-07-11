package encoder

import (
	"runtime"
	"time"

	"webp_server_go/config"

	log "github.com/sirupsen/logrus"

	pond "github.com/alitto/pond"
	pq "github.com/emirpasic/gods/queues/priorityqueue"
	hs "github.com/emirpasic/gods/sets/hashset"
	"github.com/emirpasic/gods/utils"
	"github.com/go-co-op/gocron"
)

var (
	DefaultWorkQueue  *pq.Queue        // Queue of pending image convertions
	HeavyWorkQueue    *pq.Queue        // Queue of pending image heavy convertions (e.g. Avif)
	WorkOngoingSet    *hs.Set          // Tracks the ongoing work to avoid queue duplicate convertions
	DefaultWorkerPool *pond.WorkerPool // Default worker pool
	HeavyWorkerPool   *pond.WorkerPool // Worker pool for heavy/long convertions (e.g. Avif)
	Beat              *gocron.Scheduler
)

func Lazy() {
	log.Info("Lazy mode enabled!")
	DefaultWorkQueue = pq.NewWith(byPriority) // Default tasks queue
	HeavyWorkQueue = pq.NewWith(byPriority)   // Heavy tasks queue
	WorkOngoingSet = hs.New()                 // In-flight operations

	// Create a buffered (non-blocking) pool that can scale up to runtime.NumCPU() workers
	// and has a buffer capacity of 1000 tasks
	DefaultWorkerPool = pond.New(runtime.NumCPU(), 1000)
	defer DefaultWorkerPool.StopAndWait()

	// Heavy tasks are the most resource intensive ones (e.g. Avif)
	HeavyWorkerPool = pond.New(config.MaxHeavyJobs, 1000)
	defer HeavyWorkerPool.StopAndWait()

	Beat = gocron.NewScheduler(time.UTC)
	Beat.SetMaxConcurrentJobs(1, gocron.RescheduleMode)
	_, err := Beat.Every(config.LazyTickerPeriod).Do(func() {
		lazyDo()
	})
	if err != nil {
		log.Panic("Error starting lazy beat", err)
	}

	defer Beat.Stop()
	Beat.StartBlocking()
}

// Create jobs from the pools
func lazyDo() {
	log.Tracef("DefaultWorkQueue size:%d", DefaultWorkQueue.Size())
	for i := 0; i < DefaultWorkQueue.Size(); i++ {
		DefaultWorkerPool.Submit(convertDefaultWork)
	}
	log.Tracef("HeavyWorkQueue size:%d", HeavyWorkQueue.Size())
	for i := 0; i < HeavyWorkQueue.Size(); i++ {
		HeavyWorkerPool.Submit(convertHeavyWork)
	}
}

// Comparator function (sort by element's priority value in descending order)
func byPriority(a, b interface{}) int {
	priorityA := a.(config.Element).Priority
	priorityB := b.(config.Element).Priority
	return -utils.IntComparator(priorityA, priorityB) // "-" descending order
}
