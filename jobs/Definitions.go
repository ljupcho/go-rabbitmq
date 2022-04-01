package jobs

import (
	"fmt"

	"go.uber.org/ratelimit"
)

type JobPayload struct {
	JobName   string
	ID        string
	QueueName string
	Message   []byte
}

type JobInterface interface {
	SetPayload(payload JobPayload)
	GetPayload() JobPayload
	Handle()
}

type JobRateLimitedInterface interface {
	GetRateLimiterName() string
	SetRateLimiter(llimiter *ratelimit.Limiter)
}

func CreateJob(jobType string) JobInterface {
	var job JobInterface
	switch jobType {
	case "CreateProduct":
		job = NewCreateProductJob()
	default:
		fmt.Println("Job was not found.")
	}

	return job
}
