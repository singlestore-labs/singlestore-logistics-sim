package main

import (
	"fmt"
	"time"
)

func main() {
	// TODO: find a kafka producer
	// https://github.com/ORBAT/krater/ looks promising

	// use github.com/hamba/avro
	// Encoder to setup avro writing for each of the topics

	// use github.com/satori/go.uuid

	for {
		fmt.Println("hello world")
		time.Sleep(time.Second)
	}
}
