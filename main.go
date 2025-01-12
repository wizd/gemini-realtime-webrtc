package main

import (
	"log"

	"github.com/realtime-ai/gemini-live-webrt/pkg/gateway"
)

func main() {
	if err := gateway.StartServer(":8080"); err != nil {
		log.Fatal(err)
	}
}
