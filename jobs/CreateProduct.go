package jobs

import (
	"fmt"

	"go.uber.org/ratelimit"
)

type CreateProduct struct {
	Name        string
	payload     JobPayload
	ratelimiter *ratelimit.Limiter
}

func NewCreateProductJob() *CreateProduct {
	return &CreateProduct{
		Name: "create-product",
	}
}

func (job *CreateProduct) SetPayload(payload JobPayload) {
	job.payload = payload
}

func (job *CreateProduct) GetPayload() JobPayload {
	return job.payload
}

func (job *CreateProduct) GetRateLimiterName() string {
	return "products"
}

func (job *CreateProduct) SetRateLimiter(limiter *ratelimit.Limiter) {
	job.ratelimiter = limiter
}

func (job *CreateProduct) Handle() {
	id := string(job.payload.Message)

	// Rate limit the request to external api.
	// Set this before each http call.
	// Queues are FIFO in rabbitmq so this will be shortly delayed only.
	// Rate limiting should effect all queues, consumers and connections.

	// (*job.ratelimiter).Take()

	fmt.Println(fmt.Sprintf("Received a message in create product job %v", id))
}
