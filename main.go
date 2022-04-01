package main

import (
	"net/http"

	"app/amqp"

	"github.com/gin-gonic/gin"
)

var router *gin.Engine

func main() {
	router = gin.Default()

	router.GET("/api", func(c *gin.Context) {
		// This is a lot a faster with existing connectinon and poll of 4 channels already setup on creation of the api.
		// For dedicated publish you can create a new connection and use `PublishAll` method with all
		// the messages to be published with the pool of channels.
		amqp.New().PublishWithConnection("products", "CreateProduct", []byte("Hello, world!"))

		c.JSON(http.StatusOK, gin.H{
			"data": "CreateProduct job run.",
		})
	})

	// Create a connection to rabbitmq to publishing with poll of channels.
	amqp.New().CreatePublishConnection()

	// Start serving the application
	router.Run()

}
