package main

import (
	"github.com/Shopify/sarama"

	"flag"
	"net/http"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

func main() {
	brokers := flag.String("brokers", "127.0.0.1:9092", "Comma separted brokers list")
	url := flag.String("url", "https://api.bitcoinaverage.com/ticker/global/all", "JSON URL")
	topic := flag.String("topic", "test-bitcoin", "Topic")
	wait := flag.Int("wait", 30, "seconds")
	// another example: https://api.github.com/events, test-github, 1
	flag.Parse()

	brokerList := strings.Split(*brokers, ",")
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}
	defer producer.Close()
	message := make(chan string)

	go func() {
		client := new(http.Client)
		for {
			req, _ := http.NewRequest("GET", *url, nil)
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalln("Failed to fetch data: ", err)
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln("Failed to get response body: ", err)
			}
			message <- string(body[:])
			time.Sleep(time.Duration(*wait) * time.Second)
		}
	}()

	for {
		m, ok := <- message
		if !ok {
			log.Fatalln("Channel error")
		}
		partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
			Topic: *topic,
			Value: sarama.StringEncoder(m),
		})
		if err != nil {
			log.Fatalln("Failed to send message to Kafka:", err)
		} else {
			log.Printf("Succeed to send message to Kafka: %s/%d/%d\n",
				*topic, partition, offset)
		}
	}
}

