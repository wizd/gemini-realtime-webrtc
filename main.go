package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/realtime-ai/gemini-realtime-webrtc/pkg/server"
)

// StartServer 启动 WebRTC 服务器
func StartServer(addr string) error {
	rtcServer := server.NewWebRTCServer(9000)
	rtcServer.Start()

	http.HandleFunc("/session", rtcServer.HandleNegotiate)

	log.Printf("WebRTC server starting on %s", addr)
	return http.ListenAndServe(addr, nil)
}

func main() {
	godotenv.Load()
	// if err := gateway.StartServer(":8080"); err != nil {
	// 	log.Fatal(err)
	// }

	StartServer(":8280")
}
