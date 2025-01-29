package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/realtime-ai/gemini-realtime-webrtc/pkg/connection"
)

type WebRTCServer struct {
	sync.RWMutex
	peers      map[string]*connection.RTCConnectionWrapper
	rtcUDPPort int
	api        *webrtc.API
}

func NewWebRTCServer(rtcUDPPort int) *WebRTCServer {

	return &WebRTCServer{
		rtcUDPPort: rtcUDPPort,
		peers:      make(map[string]*connection.RTCConnectionWrapper),
	}
}

func (s *WebRTCServer) Start() error {

	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetLite(true)

	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeTCP4,
	})

	udpListener, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: s.rtcUDPPort,
	})

	if err != nil {
		fmt.Printf("监听 UDP 端口失败: %v\n", err)
		return err
	}

	udpMux := webrtc.NewICEUDPMux(nil, udpListener)
	settingEngine.SetICEUDPMux(udpMux)

	api := webrtc.NewAPI(webrtc.WithSettingEngine(settingEngine))

	s.api = api

	return nil

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
	pc, err := s.api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{},
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
		fmt.Printf("Failed to set remote description: %v\n", err)
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
