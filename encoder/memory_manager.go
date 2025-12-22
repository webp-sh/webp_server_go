package encoder

import (
	"runtime"
	"sync"
	"time"
	"webp_server_go/config"

	log "github.com/sirupsen/logrus"
)

// MemoryManager 管理内存和并发限制
type MemoryManager struct {
	maxConcurrency int
	currentJobs    int
	jobQueue       chan *ConversionJob
	mu             sync.RWMutex
	semaphore      chan struct{}
	memoryLimitMB  int64
	currentMemory  int64
	memoryMu       sync.RWMutex
}

// ConversionJob 转换任务
type ConversionJob struct {
	RawPath       string
	JxlPath       string
	AvifPath      string
	WebpPath      string
	ExtraParams   ExtraParams
	SupportedFormats map[string]bool
	Chan          chan int
}

var memManager *MemoryManager
var once sync.Once

// GetMemoryManager 获取内存管理器单例
func GetMemoryManager() *MemoryManager {
	once.Do(func() {
		// 使用配置中的参数
		maxConc := config.Config.MaxConcurrentConversions
		memLimit := int64(config.Config.MemoryLimitMB)

		// 确保参数在合理范围内
		if maxConc <= 0 {
			maxConc = runtime.NumCPU() * 2 // 默认每个CPU核心2个并发
		}
		if maxConc > 16 {
			maxConc = 16 // 最大16个并发，防止内存爆炸
		}
		if maxConc < 2 {
			maxConc = 2 // 最少2个并发
		}

		if memLimit <= 0 {
			memLimit = 200 // 默认200MB
		}

		memManager = &MemoryManager{
			maxConcurrency: maxConc,
			jobQueue:       make(chan *ConversionJob, maxConc*2), // 队列大小是并发数的2倍
			semaphore:      make(chan struct{}, maxConc),
			memoryLimitMB:  memLimit,
		}

		// 启动工作池
		for i := 0; i < maxConc; i++ {
			go memManager.worker()
		}

		log.Infof("MemoryManager initialized: max_concurrency=%d, memory_limit=%dMB", maxConc, memLimit)
	})
	return memManager
}

// worker 工作协程
func (m *MemoryManager) worker() {
	for job := range m.jobQueue {
		// 获取信号量
		m.semaphore <- struct{}{}
		
		// 检查内存限制
		if !m.checkMemoryLimit() {
			log.Warn("Memory limit reached, waiting...")
			// 等待内存释放
			for !m.checkMemoryLimit() {
				time.Sleep(100 * time.Millisecond)
				runtime.GC() // 强制垃圾回收
			}
		}

		// 记录开始
		m.mu.Lock()
		m.currentJobs++
		m.mu.Unlock()

		// 估算内存使用并记录
		estimatedMemory := m.estimateMemoryUsage(job.RawPath)
		m.memoryMu.Lock()
		m.currentMemory += estimatedMemory
		m.memoryMu.Unlock()

		// 执行转换
		m.processJob(job)

		// 释放信号量和内存计数
		<-m.semaphore
		
		m.mu.Lock()
		m.currentJobs--
		m.mu.Unlock()

		m.memoryMu.Lock()
		m.currentMemory -= estimatedMemory
		m.memoryMu.Unlock()

		log.Debugf("Job completed. Current jobs: %d, Memory usage: %dMB", 
			m.currentJobs, m.currentMemory)
	}
}

// checkMemoryLimit 检查内存使用是否超限
func (m *MemoryManager) checkMemoryLimit() bool {
	m.memoryMu.RLock()
	defer m.memoryMu.RUnlock()
	return m.currentMemory < m.memoryLimitMB
}

// estimateMemoryUsage 估算转换任务的内存使用量(MB)
func (m *MemoryManager) estimateMemoryUsage(filepath string) int64 {
	// 基于文件大小的简单估算
	// 假设转换过程中内存使用是文件大小的5-10倍
	// 这里使用保守估计：20MB
	return 20
}

// processJob 处理单个转换任务
func (m *MemoryManager) processJob(job *ConversionJob) {
	// 执行实际的转换逻辑
	convertSync(job.RawPath, job.JxlPath, job.AvifPath, job.WebpPath, 
		job.ExtraParams, job.SupportedFormats, job.Chan)
}

// SubmitJob 提交转换任务
func (m *MemoryManager) SubmitJob(job *ConversionJob) {
	select {
	case m.jobQueue <- job:
		log.Debugf("Job submitted to queue: %s", job.RawPath)
	default:
		// 队列满，直接返回错误
		log.Warn("Job queue is full, rejecting job")
		if job.Chan != nil {
			close(job.Chan)
		}
	}
}

// GetStats 获取统计信息
func (m *MemoryManager) GetStats() (int, int64, int) {
	m.mu.RLock()
	m.memoryMu.RLock()
	defer m.mu.RUnlock()
	defer m.memoryMu.RUnlock()
	return m.currentJobs, m.currentMemory, cap(m.jobQueue) - len(m.jobQueue)
}