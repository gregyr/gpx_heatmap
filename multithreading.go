package main

import "sync"

// PlotJob represents a single tile plotting task to be processed by a worker
type PlotJob struct {
	tile   Tile
	zoom   int
	p1     Point
	p2     Point
	routes []Route
}

// WorkerPool manages a pool of goroutines that process plotting jobs
type WorkerPool struct {
	jobQueue   chan PlotJob
	numWorkers int
	wg         sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with given number of workers
// numWorkers: how many concurrent goroutines to spawn
// bufferSize: size of the job queue channel (higher = more jobs queued before blocking)
func NewWorkerPool(numWorkers int, bufferSize int) *WorkerPool {
	return &WorkerPool{
		jobQueue:   make(chan PlotJob, bufferSize),
		numWorkers: numWorkers,
	}
}

// Start initializes the worker goroutines and starts listening for jobs
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// worker is the function each goroutine runs - pulls jobs from queue and processes them
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for job := range wp.jobQueue {
		// Process the job - this replaces the current plotRoutes call
		plotRoutes(job.routes, job.p1, job.p2, job.tile, job.zoom)
	}
}

// Submit adds a new plotting job to the queue (blocks if queue is full)
func (wp *WorkerPool) Submit(job PlotJob) {
	wp.jobQueue <- job
}

// Close waits for all jobs to complete and shuts down workers
func (wp *WorkerPool) Close() {
	close(wp.jobQueue)
	wp.wg.Wait()
}
