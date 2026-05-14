package main

import "sync"

// tile plotting job
type PlotJob struct {
	tile   Tile
	zoom   int
	p1     Point
	p2     Point
	routes []Route
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
		plotRoutes(job.routes, job.p1, job.p2, job.tile, job.zoom)
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
