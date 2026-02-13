package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// PerformanceOptimizer handles performance optimization for large projects
type PerformanceOptimizer struct {
	config        *OptimizationConfig
	cache         *CacheManager
	concurrency   *ConcurrencyManager
	memoryMonitor *MemoryMonitor
	metrics       *PerformanceMetrics
}

// OptimizationConfig holds performance optimization configuration
type OptimizationConfig struct {
	MaxConcurrentSessions int
	CacheEnabled          bool
	CacheSizeMB           int
	MemoryLimitMB         int
	GCThreshold           float64
	BatchProcessing       bool
	BatchSize             int
	SessionTimeout        time.Duration // Add dedicated timeout config
	RequestTimeout        time.Duration // Add request-level timeout
}

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	TotalOperations    int64
	SuccessfulOps      int64
	FailedOps          int64
	AvgResponseTime    time.Duration
	CacheHitRate       float64
	MemoryUsageMB      int64
	GCCycles           int64
	ConcurrentSessions int64
	LastError          string
	LastUpdated        time.Time
}

// CacheManager handles caching for performance optimization
type CacheManager struct {
	entries     map[string]*CacheEntry
	maxSize     int
	currentSize int
	mutex       sync.RWMutex
	hitCount    int64
	missCount   int64
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key         string
	Value       interface{}
	Created     time.Time
	LastAccess  time.Time
	AccessCount int64
}

// ConcurrencyManager manages concurrent operations
type ConcurrencyManager struct {
	semaphore     chan struct{}
	maxWorkers    int
	activeWorkers int64
	mutex         sync.RWMutex
}

// MemoryMonitor tracks memory usage and triggers GC when needed
type MemoryMonitor struct {
	limitMB     int64
	threshold   float64
	lastGC      time.Time
	gcTriggered int64
	callback    func()
}

// NewPerformanceOptimizer creates a new performance optimizer with better defaults
func NewPerformanceOptimizer(config *OptimizationConfig) *PerformanceOptimizer {
	// Set reasonable defaults if not configured
	if config.SessionTimeout == 0 {
		config.SessionTimeout = 30 * time.Minute
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 5 * time.Minute
	}

	return &PerformanceOptimizer{
		config:        config,
		cache:         NewCacheManager(config.CacheSizeMB),
		concurrency:   NewConcurrencyManager(config.MaxConcurrentSessions),
		memoryMonitor: NewMemoryMonitor(config.MemoryLimitMB, config.GCThreshold),
		metrics:       &PerformanceMetrics{},
	}
}

// NewCacheManager creates a new cache manager
func NewCacheManager(maxSizeMB int) *CacheManager {
	return &CacheManager{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSizeMB * 1024 * 1024, // Convert MB to bytes
	}
}

// NewConcurrencyManager creates a new concurrency manager
func NewConcurrencyManager(maxWorkers int) *ConcurrencyManager {
	return &ConcurrencyManager{
		semaphore:  make(chan struct{}, maxWorkers),
		maxWorkers: maxWorkers,
	}
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(limitMB int, threshold float64) *MemoryMonitor {
	return &MemoryMonitor{
		limitMB:   int64(limitMB),
		threshold: threshold,
		lastGC:    time.Now(),
	}
}

// OptimizeContext optimizes a context with proper timeouts
func (po *PerformanceOptimizer) OptimizeContext(ctx context.Context) context.Context {
	// Use request timeout for individual operations
	newCtx, _ := context.WithTimeout(ctx, po.config.RequestTimeout)
	return newCtx
}

// WithCaching executes a function with caching
func (po *PerformanceOptimizer) WithCaching(key string, fn func() (interface{}, error)) (interface{}, error) {
	if !po.config.CacheEnabled {
		return fn()
	}

	// Try to get from cache
	if value, found := po.cache.Get(key); found {
		po.metrics.CacheHitRate = float64(po.cache.hitCount) / float64(po.cache.hitCount+po.cache.missCount)
		return value, nil
	}

	// Execute function and cache result
	value, err := fn()
	if err != nil {
		return nil, err
	}

	po.cache.Set(key, value)
	po.metrics.CacheHitRate = float64(po.cache.hitCount) / float64(po.cache.hitCount+po.cache.missCount)
	return value, nil
}

// WithConcurrency executes a function with concurrency control
func (po *PerformanceOptimizer) WithConcurrency(fn func() error) error {
	startTime := time.Now()

	// Acquire semaphore
	po.concurrency.Acquire()
	defer po.concurrency.Release()

	// Update active sessions count
	po.metrics.ConcurrentSessions = po.concurrency.GetActiveCount()

	// Execute function
	err := fn()

	// Update metrics
	po.metrics.TotalOperations++
	if err != nil {
		po.metrics.FailedOps++
		po.metrics.LastError = err.Error()
	} else {
		po.metrics.SuccessfulOps++
	}

	po.metrics.AvgResponseTime = (po.metrics.AvgResponseTime*time.Duration(po.metrics.TotalOperations-1) +
		time.Since(startTime)) / time.Duration(po.metrics.TotalOperations)
	po.metrics.LastUpdated = time.Now()

	// Monitor memory usage
	po.memoryMonitor.CheckAndGC(func() {
		po.metrics.GCCycles++
		runtime.GC()
	})

	return err
}

// GetMetrics returns current performance metrics
func (po *PerformanceOptimizer) GetMetrics() *PerformanceMetrics {
	po.metrics.MemoryUsageMB = getCurrentMemoryUsage()
	return po.metrics
}

// Cache operations
func (cm *CacheManager) Get(key string) (interface{}, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	entry, exists := cm.entries[key]
	if !exists {
		cm.missCount++
		return nil, false
	}

	entry.LastAccess = time.Now()
	entry.AccessCount++
	cm.hitCount++

	return entry.Value, true
}

func (cm *CacheManager) Set(key string, value interface{}) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Remove oldest entries if cache is full
	for cm.currentSize >= cm.maxSize && len(cm.entries) > 0 {
		cm.removeOldest()
	}

	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		Created:     time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 1,
	}

	cm.entries[key] = entry
	cm.currentSize += cm.getSize(value)
}

func (cm *CacheManager) removeOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range cm.entries {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	if oldestKey != "" {
		entry := cm.entries[oldestKey]
		cm.currentSize -= cm.getSize(entry.Value)
		delete(cm.entries, oldestKey)
	}
}

func (cm *CacheManager) getSize(value interface{}) int {
	// Simplified size calculation
	// In practice, you'd want more accurate size measurement
	return 1024 // Assume 1KB per entry for simplicity
}

// Concurrency operations
func (cm *ConcurrencyManager) Acquire() {
	cm.semaphore <- struct{}{}
	cm.mutex.Lock()
	cm.activeWorkers++
	cm.mutex.Unlock()
}

func (cm *ConcurrencyManager) Release() {
	<-cm.semaphore
	cm.mutex.Lock()
	cm.activeWorkers--
	cm.mutex.Unlock()
}

func (cm *ConcurrencyManager) GetActiveCount() int64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.activeWorkers
}

// Memory monitoring operations
func (mm *MemoryMonitor) CheckAndGC(callback func()) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memoryMB := int64(m.Alloc) / 1024 / 1024

	if memoryMB > mm.limitMB {
		usageRatio := float64(memoryMB) / float64(mm.limitMB)
		if usageRatio > mm.threshold {
			callback()
			mm.lastGC = time.Now()
			mm.gcTriggered++
		}
	}
}

// Helper function to get current memory usage
func getCurrentMemoryUsage() int64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int64(m.Alloc) / 1024 / 1024
}

// BatchProcessor handles batch processing for large datasets
type BatchProcessor struct {
	batchSize int
	workers   int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize, workers int) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
		workers:   workers,
	}
}

// ProcessBatch processes items in batches with concurrency
func (bp *BatchProcessor) ProcessBatch(items []interface{}, processor func(interface{}) error) error {
	if len(items) == 0 {
		return nil
	}

	// Split into batches
	batches := bp.splitIntoBatches(items)

	// Process batches concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(batches))

	for _, batch := range batches {
		wg.Add(1)
		go func(batch []interface{}) {
			defer wg.Done()
			for _, item := range batch {
				if err := processor(item); err != nil {
					errChan <- err
					return
				}
			}
		}(batch)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (bp *BatchProcessor) splitIntoBatches(items []interface{}) [][]interface{} {
	var batches [][]interface{}

	for i := 0; i < len(items); i += bp.batchSize {
		end := i + bp.batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}

	return batches
}

// GeneratePerformanceReport creates a performance optimization report
func (po *PerformanceOptimizer) GeneratePerformanceReport() string {
	metrics := po.GetMetrics()

	report := fmt.Sprintf(`
# Performance Optimization Report

Generated: %s

## Resource Usage
- Memory Usage: %d MB
- Concurrent Sessions: %d
- GC Cycles: %d

## Operation Metrics
- Total Operations: %d
- Successful Operations: %d
- Failed Operations: %d
- Success Rate: %.2f%%
- Average Response Time: %v

## Cache Performance
- Cache Hit Rate: %.2f%%
- Active Cache Entries: %d

## Last Error
%s
`,
		time.Now().Format("2006-01-02 15:04:05"),
		metrics.MemoryUsageMB,
		metrics.ConcurrentSessions,
		metrics.GCCycles,
		metrics.TotalOperations,
		metrics.SuccessfulOps,
		metrics.FailedOps,
		float64(metrics.SuccessfulOps)/float64(metrics.TotalOperations)*100,
		metrics.AvgResponseTime,
		metrics.CacheHitRate*100,
		len(po.cache.entries),
		metrics.LastError)

	return report
}
