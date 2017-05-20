package main

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync/atomic"
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
	port = 4242
)

func main() {
	http.API(name, buildDate, gitCommit, port, routes)
}

func routes(r *gin.Engine) {
	r.GET("/stream", stream)
}

func stream(c *gin.Context) {
	brokers := c.Query("b")
	topic := c.Query("t")
	user := c.Query("u")
	password := c.Query("p")

	if brokers == "" || topic == "" || user == "" || password == "" {
		fmt.Println("Invalid params")
		c.JSON(400, gin.H{"error": "Missing parameter"})
		return
	}

	// Init config
	clusterConfig := cluster.NewConfig()
	clusterConfig.ClientID = user
	clusterConfig.Net.TLS.Enable = true
	clusterConfig.Net.SASL.Enable = true
	clusterConfig.Net.SASL.User = user
	clusterConfig.Net.SASL.Password = password

	clusterConfig.Consumer.Return.Errors = true
	clusterConfig.Group.Return.Notifications = true
	clusterConfig.Version = sarama.V0_10_1_0
	clusterConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	consumerGID := user + "." + name + "." + fmt.Sprintf("%d-%d", time.Now().Unix(), rand.Intn(10000))

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
			logrus.WithError(err).Error("Fail to consume")
		}
	}()

	// Consumer notifications
	go func() {
		for note := range consumer.Notifications() {
			logrus.WithField("note", note).Print("Rebalanced consumer")
		}
	}()

	var msgs uint64

	go func() {
		for range consumer.Messages() {
			atomic.AddUint64(&msgs, 1)
		}
	}()

	sseChan := make(chan uint64)

	go func() {
		tick := time.NewTicker(time.Millisecond * 50)
		for range tick.C {
			sseChan <- msgs
		}
	}()

	c.Stream(func(w io.Writer) bool {
		c.SSEvent("message", <-sseChan)
		return true
	})
}
