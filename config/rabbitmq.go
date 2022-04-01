package config

type (
	QueueConfig struct {
		Name         string
		ExchangeType string
		Workers      int
	}

	Queues []QueueConfig

	RabbitMqConfig struct {
		Schema         string
		Username       string
		Password       string
		Host           string
		Port           string
		Vhost          string
		ConnectionName string
	}

	RateLimiter struct {
		Description      string
		RequestPerSecond int
	}
)

// GetQueues gets queue configuration.
func GetQueues() Queues {
	return []QueueConfig{
		{
			Name:         "registrations",
			ExchangeType: "direct",
			Workers:      2,
		},
		{
			Name:         "products",
			ExchangeType: "direct",
			Workers:      4,
		},
		{
			Name:         "orders",
			ExchangeType: "direct",
			Workers:      4,
		},
		// Add new queue here and restart consumers.
	}
}

// GetRabbitMqConfig sets credentials to rabbitmq server.
func GetRabbitMqConfig() *RabbitMqConfig {
	config := &RabbitMqConfig{
		Schema:         "amqp",
		Username:       "guest",
		Password:       "guest",
		Host:           "127.0.0.1",
		Port:           "5672",
		Vhost:          "demo",
		ConnectionName: "demo",
	}

	return config
}

// GetRateLimiters defines rate limiters with different rates.
func GetRateLimiters() map[string]RateLimiter {
	limiters := map[string]RateLimiter{
		"products": {
			Description:      "rate limiting products",
			RequestPerSecond: 4,
		},
		"products_restrictive": {
			Description:      "rate limiting products more restrictively",
			RequestPerSecond: 2,
		},
		// Add new rate limiters that you'd define and use in jobs.
	}

	return limiters
}
