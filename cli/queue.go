package main

import (
	"app/amqp"
	"os"
)

func main() {
	// definitions of queue cli commands:
	// go run cli/queue.go start
	// go run cli/queue.go retry {JobId}

	args := os.Args

	switch args[1] {
	case "start":
		start()
	case "retry":
		var jobId string
		jobId = args[2]
		retryJob(jobId)
	}
}

// Start the queue workers.
func start() {
	// Define number of connection that will be used for consumers.
	// We will define 1 per process and start multipe processes with supervisor,
	// so we would have more processes for fallover.
	// You set also set multiple rabbitmq connection for one single process as well.
	numOfConnections := 1

	srv := amqp.New()
	srv.SetRateLimiters()
	srv.Run(numOfConnections)
}

// Retry a job that has failed.
func retryJob(jobId string) {
	// find failed job by jobId
	// job := m.FailedJobById(jobId)
	// // re publish the job back to the queue
	// payload := amqp.FromGOB64(job.Payload)
	// amqp.New().Publish(job.Queue, payload.JobName, payload.Message)
	// // delete the existing one in failed_jobs table
	// m.DeleteFailedJob(jobId)
}
