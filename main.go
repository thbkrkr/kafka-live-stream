package main

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Sirupsen/logrus"
	"github.com/bsm/sarama-cluster"
	"github.com/gin-gonic/gin"
	"github.com/thbkrkr/go-utilz/http"
)

var (
	buildDate = "dev"
	gitCommit = "dev"

	name = "kafka-live-stream"
)

func main() {
	http.API(name, buildDate, gitCommit, routes)
}

func routes(r *gin.Engine) {
	r.GET("/stream", stream)
}

func stream(c *gin.Context) {
	brokers := c.Query("b")
	key := c.Query("k")
	topic := c.Query("t")

	if brokers == "" || key == "" || topic == "" {
		fmt.Println("Invalid params")
		c.JSON(400, gin.H{"error": "Missing parameter"})
		return
	}

	consumerGID := name + fmt.Sprintf("%d-%d", time.Now().Unix(), rand.Intn(10000))

	// Init config
	clusterConfig := cluster.NewConfig()
	clusterConfig.ClientID = key
	clusterConfig.Consumer.Return.Errors = true
	clusterConfig.Group.Return.Notifications = true
	clusterConfig.Version = sarama.V0_9_0_1
	clusterConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	// Init consumer
	consumer, err := cluster.NewConsumer(strings.Split(brokers, ","), consumerGID, []string{topic}, clusterConfig)
	if err != nil {
		fmt.Println("Fail to create consumer ", err.Error())
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	defer func() {
		if err := consumer.Close(); err != nil {
		}
	}()

	// Consumer errors
	go func() {
		for err := range consumer.Errors() {
			logrus.Printf("Error: %s\n", err.Error())
		}
	}()

	// Consumer notifications
	go func() {
		for note := range consumer.Notifications() {
			logrus.WithField("note", note).Print("Rebalanced consumer")
		}
	}()

	c.Stream(func(w io.Writer) bool {
		c.SSEvent("message", <-consumer.Messages())
		return true
	})
}
