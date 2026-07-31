package generation

import (
	"os"
	"sync"
)

// safe shared set for staged file paths
// used by worker goroutines without racing on a plain map

type SafeStringSet struct {
	mu     sync.RWMutex
	values map[string]*os.File
}

func NewSafeStringSet() *SafeStringSet {
	return &SafeStringSet{values: make(map[string]*os.File)}
}

func (s *SafeStringSet) Add(value string, file *os.File) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[value] = file
}

func (s *SafeStringSet) Contains(value string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.values[value]
	return ok
}

// tile plotting job
type PlotJob struct {
	tile        Tile
	zoom        int
	p1          Point
	p2          Point
	routes      []Route
	stagedFiles *SafeStringSet
}

// manages a pool of goroutines
type WorkerPool struct {
	jobQueue   chan PlotJob
	numWorkers int
	wg         sync.WaitGroup
}

// creates a new worker pool with given number of workers
// buffersize is the max amount of jobs
func NewWorkerPool(numWorkers int, bufferSize int) *WorkerPool {
	return &WorkerPool{
		jobQueue:   make(chan PlotJob, bufferSize),
		numWorkers: numWorkers,
	}
}

// initializes the workers -> listening for jobs
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// function of each worker -> pulls jobs from job queue while there are jobs in there
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for job := range wp.jobQueue {
		// actual plotting
		plotRoutes(job.routes, job.p1, job.p2, job.tile, job.zoom, job.stagedFiles)
	}
}

// adds job to the queue
func (wp *WorkerPool) Submit(job PlotJob) {
	wp.jobQueue <- job
}

// waits for jobs to complete than stops workers
func (wp *WorkerPool) Close() {
	close(wp.jobQueue)
	wp.wg.Wait()
}
