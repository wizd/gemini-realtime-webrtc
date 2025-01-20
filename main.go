package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/realtime-ai/gemini-realtime-webrtc/pkg/gateway"
)

func main() {
	godotenv.Load()
	if err := gateway.StartServer(":8280"); err != nil {
		log.Fatal(err)
	}
}
