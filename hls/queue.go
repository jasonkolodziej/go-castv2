package hls

import (
	"context"
	"sync"

	logg "github.com/sirupsen/logrus"
)

// Queue holds name, list of jobs and context with cancel.
//
// [Queue]:https://webdevstation.com/posts/simple-queue-implementation-in-golang/
// [Simple Queue]:https://dev.to/nonsoamadi10/building-a-modern-message-queue-5402
type Queue struct {
	name   string
	jobs   chan Job // * unbuffered channel (with no capacity) to hold Jobs, Sending a value through an unbuffered channel will block until the value is received.
	ctx    context.Context
	cancel context.CancelFunc
}

// Job - holds logic to perform some operations during queue execution.
type Job struct {
	Name   string
	Action func() error // A function that should be executed when the job is running.
}

// NewQueue instantiates new queue.
func NewQueue(name string) *Queue {
	ctx, cancel := context.WithCancel(context.Background())

	return &Queue{
		jobs:   make(chan Job), // * create the unbuffered channel of a Job
		name:   name,
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddJobs adds jobs to the queue and cancels channel.
func (q *Queue) AddJobs(jobs []Job) {
	var wg sync.WaitGroup
	wg.Add(len(jobs))

	for _, job := range jobs {
		// Goroutine which adds job to the queue.
		go func(job Job) {
			q.AddJob(job)
			wg.Done()
		}(job)
	}

	go func() {
		wg.Wait()
		// Cancel queue channel, when all goroutines were done.
		q.cancel()
	}()
}

// AddJob sends job to the channel.
func (q *Queue) AddJob(job Job) {
	q.jobs <- job
	logg.WithField("package", "application").Infof("New job %s added to %s queue", job.Name, q.name)
}

// Run performs job execution.
func (j Job) Run() error {
	logg.WithField("package", "application").Infof("Job running: %s", j.Name)

	err := j.Action()
	if err != nil {
		return err
	}

	return nil
}

// Worker responsible for queue serving.
type Worker struct {
	Queue *Queue
}

// NewWorker initializes a new Worker.
func NewWorker(queue *Queue) *Worker {
	return &Worker{
		Queue: queue,
	}
}

// DoWork processes jobs from the queue (jobs channel).
// we have a for loop, which is listening for the jobs channel,
// and if the channel receives a job, it executes it, by running Job.Run().
// If the context was canceled, it means, that all jobs were executed and we can exit from the loop.
func (w *Worker) DoWork() bool {
	for {
		select {
		// if context was canceled.
		case <-w.Queue.ctx.Done():
			logg.WithField("package", "application").Warnf("Work done in queue %s: %s!", w.Queue.name, w.Queue.ctx.Err())
			return true
		// if job received.
		case job := <-w.Queue.jobs:
			err := job.Run()
			if err != nil {
				logg.WithField("package", "application").Error(err)
				continue
			}
		}
	}
}
