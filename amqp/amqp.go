package amqp

import (
	"app/config"
	"app/jobs"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v2/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/ratelimit"
)

var (
	publisher    *amqp.Publisher
	ratelimiters map[string]*ratelimit.Limiter
)

type (
	AmqpInterface interface {
		Run(numOfConnections int)
		Publish(queueName, jobName string, payload []byte)
		PublishWithConnection(queueName, jobName string, payload []byte)
		PublishAll(queueName, jobName string, data [][]byte)
		CreatePublishConnection()
		SetRateLimiters()
		GetRateLimiter(name string) *ratelimit.Limiter
	}

	service struct {
		sync.Mutex
		config *config.RabbitMqConfig
		queues config.Queues
	}
)

func New() AmqpInterface {
	return &service{
		config: config.GetRabbitMqConfig(),
		queues: config.GetQueues(),
	}
}

// Build url to connect to amqp/rabbitmq.
func (svc *service) buildAmqpUrl() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s:%s/%s",
		svc.config.Schema,
		svc.config.Username,
		svc.config.Password,
		svc.config.Host,
		svc.config.Port,
		svc.config.Vhost,
	)
}

func (svc *service) Run(numOfConnections int) {
	for i := 0; i < numOfConnections; i++ {
		svc.runConnection()
	}
	// Leave the workers running to accept jobs.
	select {}
}

// Run all the queues/consumers.
func (svc *service) runConnection() {
	config := amqp.NewDurableQueueConfig(svc.buildAmqpUrl())

	logger := watermill.NewStdLogger(false, false)

	conn, err := amqp.NewConnection(config.Connection, logger)
	if err != nil {
		logger.Error("Failed to build connection for rabbitmq.", err, nil)
		return
	}

	for _, queue := range svc.queues {
		topic := queue.Name
		numOfWorkersPerConsumer := queue.Workers

		// Override some of the defaults.
		config.Exchange.GenerateName = func(topic string) string {
			return topic
		}
		config.Exchange.Type = queue.ExchangeType
		config.QueueBind.GenerateRoutingKey = func(topic string) string {
			return "routing_key_" + topic
		}

		s, err := amqp.NewSubscriberWithConnection(config, logger, conn)
		if err != nil {
			logger.Error("Failed to build consumer for rabbitmq.", err, nil)
			return
		}

		for i := 0; i < numOfWorkersPerConsumer; i++ {
			msgChan, err := s.Subscribe(context.Background(), topic)
			if err != nil {
				logger.Error("Failed to subscribe for consumer for rabbitmq.", err, nil)
				return
			}
			go func() {
				for d := range msgChan {
					jobName := d.Metadata.Get("job")
					queueName := d.Metadata.Get("queue")

					NewPayload := jobs.JobPayload{
						JobName:   jobName,
						ID:        d.UUID,
						QueueName: queueName,
						Message:   d.Payload,
					}

					// Will call factory that will create the right job struct.
					job := jobs.CreateJob(jobName)
					job.SetPayload(NewPayload)
					// Set rate limiter if implementing that interface.
					if m, ok := job.(jobs.JobRateLimitedInterface); ok {
						rateLimiterName := m.GetRateLimiterName()
						ratelimiter := svc.GetRateLimiter(rateLimiterName)
						m.SetRateLimiter(ratelimiter)
					}

					{
						func() {
							defer func() {
								// Catch a panic exception to fail and log the job.
								if err := recover(); err != nil {
									LogException(job, err)
								}
							}()

							job.Handle()
						}()
					}

					d.Ack()
				}
			}()
		}
	}
}

// CreatePublishConnection creates rabbitmq connection and stores it in cache.
func (svc *service) CreatePublishConnection() {
	config := amqp.NewDurableQueueConfig(svc.buildAmqpUrl())
	// We set 4 channels for publishing
	// so they are not closed and waiting for new data.
	config.Publish.ChannelPoolSize = 4
	logger := watermill.NewStdLogger(false, false)

	p, err := amqp.NewPublisher(config, logger)
	if err != nil {
		logger.Error("Failed to create publisher", err, nil)
	}

	// Will be used the same connection for publishing across requests.
	svc.SetPublisher(p)
}

// PublishWithConnection publishes a message to the queue with dedicated connection open
// so there be no need to create new connection on every publish.
// The connection can be cached and reused as it is set on api compilation and
// jobs are run from api as well.
func (svc *service) PublishWithConnection(queueName, jobName string, payload []byte) {
	p := svc.GetPublisher()
	if p == nil || p.Closed() || !p.IsConnected() {
		svc.CreatePublishConnection()
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("job", jobName)
	msg.Metadata.Set("queue", queueName)
	err := p.Publish(queueName, msg)
	if err != nil {
		panic(err)
	}
}

// Publish message on queue with job definitiion.
func (svc *service) Publish(queueName, jobName string, payload []byte) {
	config := amqp.NewDurableQueueConfig(svc.buildAmqpUrl())
	logger := watermill.NewStdLogger(false, false)

	p, err := amqp.NewPublisher(config, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create publisher for queue %v and job %v", queueName, jobName), err, nil)
		return
	}
	// Connection and channel will be closed after publishing.
	defer p.Close()

	msg := message.NewMessage(watermill.NewUUID(), payload)
	// So, the idea would be to use the same queue for different type of jobs.
	msg.Metadata.Set("job", jobName)
	msg.Metadata.Set("queue", queueName)
	err = p.Publish(queueName, msg)
	if err != nil {
		panic(err)
	}
}

// Publish all messages sent as slick of bytes at once to rabbitmq.
func (svc *service) PublishAll(queueName, jobName string, data [][]byte) {
	config := amqp.NewDurableQueueConfig(svc.buildAmqpUrl())
	// Commit all the messages to rabbitmq as a transaction.
	config.Publish.Transactional = true
	logger := watermill.NewStdLogger(false, false)

	p, err := amqp.NewPublisher(config, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create publisher for queue %v and job %v", queueName, jobName), err, nil)
		return
	}
	// Connection and channel will be closed after publishing.
	defer p.Close()

	var msgs []*message.Message
	// Set metadata for each of the jobs to belong to sepecific job.
	for _, payload := range data {
		msg := message.NewMessage(watermill.NewUUID(), payload)
		msg.Metadata.Set("job", jobName)
		msg.Metadata.Set("queue", queueName)
		msgs = append(msgs, msg)
	}

	err = p.Publish(queueName, msgs...)
	if err != nil {
		panic(err)
	}
}

func (svc *service) SetPublisher(p *amqp.Publisher) {
	svc.Lock()
	defer svc.Unlock()

	publisher = p
}

func (svc *service) GetPublisher() *amqp.Publisher {
	svc.Lock()
	defer svc.Unlock()

	return publisher
}

//SetRateLimiters sets different type of rate limiters to be used in jobs.
func (svc *service) SetRateLimiters() {
	svc.Lock()
	defer svc.Unlock()

	// Transform the config rate limiters with actual limiter.
	var NewLimiters = make(map[string]*ratelimit.Limiter)

	limiters := config.GetRateLimiters()
	for name, lim := range limiters {
		value := ratelimit.New(lim.RequestPerSecond)
		NewLimiters[name] = &value
	}

	ratelimiters = NewLimiters
}

// GetRateLimiter gets rate limiter by name.
func (svc *service) GetRateLimiter(name string) *ratelimit.Limiter {
	svc.Lock()
	defer svc.Unlock()

	return ratelimiters[name]
}

// Log failed job to database.
func LogException(job jobs.JobInterface, exc interface{}) {
	// logger.LogError(exc)
	// var failedJob m.FailedJob
	// failedJob.Uuid = job.GetPayload().ID
	// Keep the full job object in order to be reconstructred on retry.
	// failedJob.Payload = ToGOB64(job.GetPayload())
	// Keep the error and full trace.
	// failedJob.Exception = fmt.Sprintf("%v", exc) + "\n" + string(debug.Stack())
	// The queue this job was dispatched on.
	// failedJob.Queue = string(job.GetPayload().QueueName)

	// log to database
	// m.DB.Create(&failedJob)
}

// Binary encoder.
func ToGOB64(m jobs.JobPayload) string {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(m)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

// Binary decoder.
func FromGOB64(str string) jobs.JobPayload {
	m := jobs.JobPayload{}
	by, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		fmt.Println(`failed base64 Decode`, err)
	}
	b := bytes.Buffer{}
	b.Write(by)
	d := gob.NewDecoder(&b)
	err = d.Decode(&m)
	if err != nil {
		fmt.Println(`failed gob Decode`, err)
	}
	return m
}
