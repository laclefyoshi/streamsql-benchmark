package main

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"

	"flag"
	"net/http"
	"io/ioutil"
	"log"
	"strconv"
	"time"
)

func main() {
	url := flag.String("url", "https://api.bitcoinaverage.com/ticker/global/all", "JSON URL")
	topic := flag.String("topic", "test-bitcoin", "Topic")
	wait := flag.Int("wait", 30, "seconds")
    // another example: https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson, test-geo, 3600
	flag.Parse()

	s := session.Must(session.NewSession())
	// env: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION
	client := kinesis.New(s)
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
			message <- strings.Replace(string(body[:]), "\n", " ", -1)
			time.Sleep(time.Duration(*wait) * time.Second)
		}
	}()

	for {
		m, ok := <- message
		if !ok {
			log.Fatalln("Channel error")
		}
		key := strconv.Itoa(time.Now().Second())
		params := &kinesis.PutRecordInput{
			StreamName: topic,
			Data: []byte(m),
			PartitionKey: &key,
		}
		resp, err := client.PutRecord(params)
		if err != nil {
			log.Fatalln("Failed to send message to Kinesis:", err)
		} else {
			log.Printf("Succeed to send message to Kinesis: %v\n", resp)
		}
	}
}

