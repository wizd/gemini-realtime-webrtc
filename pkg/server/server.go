package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/realtime-ai/gemini-realtime-webrtc/pkg/connection"
)

type WebRTCServer struct {
	sync.RWMutex
	peers map[string]*connection.RTCConnectionWrapper
}

func NewWebRTCServer() *WebRTCServer {
	return &WebRTCServer{
		peers: make(map[string]*connection.RTCConnectionWrapper),
	}
}

// HandleNegotiate 处理 /session 路由
func (s *WebRTCServer) HandleNegotiate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理 OPTIONS 请求
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	var offer webrtc.SessionDescription
	if err := json.Unmarshal(body, &offer); err != nil {
		http.Error(w, "Failed to parse offer", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// 创建 PeerConnection
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		http.Error(w, "Failed to create peer connection", http.StatusInternalServerError)
		return
	}

	peerID := uuid.New().String()
	wrapper := connection.NewRTCConnectionWrapper(peerID, pc)

	// 将 wrapper 加入 server 管理
	s.Lock()
	s.peers[peerID] = wrapper
	s.Unlock()

	// 在此处启动或初始化 AI Session
	if err := wrapper.InitAISession(ctx); err != nil {
		log.Println("Failed to init AI session:", err)
	}

	err = wrapper.Start(ctx, pc)
	if err != nil {
		log.Println("Failed to start wrapper:", err)
	}

	// 将远端 Offer 设置为本地 PeerConnection 的 RemoteDescription
	if err := pc.SetRemoteDescription(offer); err != nil {
		http.Error(w, "Failed to set remote description", http.StatusInternalServerError)
		return
	}

	// 创建应答
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		http.Error(w, "Failed to create answer", http.StatusInternalServerError)
		return
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		http.Error(w, "Failed to set local description", http.StatusInternalServerError)
		return
	}

	// 等待 ICE gathering 完成
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	<-gatherComplete

	w.Header().Set("Content-Type", "application/sdp")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pc.LocalDescription())
}

// StartServer 启动 WebRTC 服务器
func StartServer(addr string) error {
	server := NewWebRTCServer()
	http.HandleFunc("/session", server.HandleNegotiate)

	log.Printf("WebRTC server starting on %s", addr)
	return http.ListenAndServe(addr, nil)
}
