package main

import (
	"github.com/joho/godotenv"
	"github.com/realtime-ai/gemini-realtime-webrtc/pkg/server"
)

func main() {
	godotenv.Load()
	// if err := gateway.StartServer(":8080"); err != nil {
	// 	log.Fatal(err)
	// }

	server.StartServer(":8080")
}
