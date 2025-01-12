package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/hraban/opus"
	"github.com/pion/webrtc/v4"
	"google.golang.org/genai"
)

const (
	sampleRate    = 48000
	channels      = 1
	frameSize     = 960
	opusFrameSize = 960  // 20ms @ 48kHz
	maxDataBytes  = 1000 // 足够大的缓冲区用于 Opus 编码数据
)

// WebRTCServer manages WebRTC connections
type WebRTCServer struct {
	sync.RWMutex
	peers map[string]*PeerConnection
}

// 创建一个PeerConnection 封装
type PeerConnection struct {
	id             string
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
	metadata       map[string]interface{}
	remoteAudio    *webrtc.TrackRemote
	localAudio     webrtc.TrackLocal
	// 初始化一个genai client
	genaiSession *genai.Session
}

// NewWebRTCServer creates a new WebRTC server instance
func NewWebRTCServer() *WebRTCServer {
	return &WebRTCServer{
		peers: make(map[string]*PeerConnection),
	}
}

// HandleNegotiate handles the WebRTC negotiation endpoint
func (s *WebRTCServer) HandleNegotiate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	offer := webrtc.SessionDescription{}
	if err := json.Unmarshal(body, &offer); err != nil {
		http.Error(w, "Failed to parse offer", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Create WebRTC configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		http.Error(w, "Failed to create peer connection", http.StatusInternalServerError)
		return
	}

	peer := &PeerConnection{
		id:             uuid.New().String(),
		peerConnection: peerConnection,
		dataChannel:    nil,
		metadata:       make(map[string]interface{}),
	}

	// Set up data channel handler
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {

		log.Printf("data channel: %+v", d)
		peer.dataChannel = d
		go s.HandleDataChannel(ctx, peer)
	})

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Received track: %+v", track)
		peer.remoteAudio = track
		go s.HandleRemoteAudio(ctx, peer)
	})

	audioTrack, audioTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	if audioTrackErr != nil {
		panic(audioTrackErr)
	}
	peer.localAudio = audioTrack

	peerConnection.AddTransceiverFromTrack(peer.localAudio, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionSendrecv,
	})

	go s.HandleLocalAudio(ctx, peer)

	// Set the remote SessionDescription
	if err = peerConnection.SetRemoteDescription(offer); err != nil {
		http.Error(w, "Failed to set remote description", http.StatusInternalServerError)
		return
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		http.Error(w, "Failed to create answer", http.StatusInternalServerError)
		return
	}

	// Set local SessionDescription
	if err = peerConnection.SetLocalDescription(answer); err != nil {
		http.Error(w, "Failed to set local description", http.StatusInternalServerError)
		return
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{Backend: genai.BackendGoogleAI})
	if err != nil {
		log.Fatal("create client error: ", err)
		http.Error(w, "Failed to create client", http.StatusInternalServerError)
		return
	}

	session, err := client.Live.Connect("gemini-2.0-flash-exp", &genai.LiveConnectConfig{})
	if err != nil {
		log.Fatal("connect to model error: ", err)
		http.Error(w, "Failed to connect to model", http.StatusInternalServerError)
		return
	}

	peer.genaiSession = session
	// Wait for ICE gathering to complete
	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	<-gatherComplete

	s.Lock()
	s.peers[peer.id] = peer
	s.Unlock()

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Received track: %+v", track)

		if track.Kind() == webrtc.RTPCodecTypeAudio {
			log.Printf("Received audio track: %+v", track)

		}
	})

	// Marshal and send the answer
	w.Header().Set("Content-Type", "application/sdp")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(peerConnection.LocalDescription())
}

func (s *WebRTCServer) HandleSession(ctx context.Context, peer *PeerConnection) {

	for {
		message, err := peer.genaiSession.Receive()
		if err != nil {
			log.Fatal("receive model response error: ", err)
		}

		log.Printf("Received message: %+v", message)

		// todo 解析audio

		messageBytes, err := json.Marshal(message)
		if err != nil {
			log.Fatal("marhal model response error: ", message, err)
		}
		err = peer.dataChannel.Send(messageBytes)
		if err != nil {
			log.Println("write message error: ", err)
			break
		}
	}

}

func (s *WebRTCServer) HandleRemoteAudio(ctx context.Context, peer *PeerConnection) {

	decoder, err := opus.NewDecoder(48000, 1)
	if err != nil {
		log.Printf("创建 Opus 解码器失败: %v\n", err)
		return
	}

	// todo need add jitter buffer

	buffer := make([]int16, frameSize)
	pcm := make([]int16, frameSize)      // 解码后的 PCM 数据
	samples := make([]byte, frameSize*2) // 用于转换的临时缓冲区

	for {
		rtp, _, err := peer.remoteAudio.ReadRTP()
		if err != nil {
			log.Printf("read rtp error: %v\n", err)
			break
		}

		n, err := decoder.Decode(rtp.Payload, pcm)
		if err != nil {
			log.Printf("解码音频失败: %v\n", err)
			continue
		}

		copy(buffer, pcm[:n])

		// 将 int16 数据转换为字节序列
		for i := 0; i < n; i++ {
			samples[i*2] = byte(buffer[i])
			samples[i*2+1] = byte(buffer[i] >> 8)
		}

		// todo resample
	}

}

func (s *WebRTCServer) HandleLocalAudio(ctx context.Context, peer *PeerConnection) {

	// 发送音频
}

// function createContent(msg) {
// 	data = { 'clientContent': { 'turnComplete': true, 'turns': [{ 'parts': [{ 'text': msg }] }] } };
// 	return JSON.stringify(data);
// }

// function createImageContent(msg) {
// 	data = { 'realtimeInput': { 'mediaChunks': [{ 'data': msg, 'mimeType': 'image/jpeg' }] } };
// 	return JSON.stringify(data);
// }

// function createAudioContent(msg) {
// 	data = { 'realtimeInput': { 'mediaChunks': [{ 'data': msg, 'mimeType': 'audio/pcm' }] } };
// 	return JSON.stringify(data);
// }

func (s *WebRTCServer) HandleDataChannel(ctx context.Context, peer *PeerConnection) {

	defer peer.dataChannel.Close()

	peer.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("Received message: %s", string(msg.Data))

		message := msg.Data

		var sendMessage genai.LiveClientMessage
		if err := json.Unmarshal(message, &sendMessage); err != nil {
			log.Println("unmarshal message error ", string(message), err)
			return
		}

		log.Printf("send message to model: %+v", sendMessage)
		peer.genaiSession.Send(&sendMessage)
	})

	<-ctx.Done()

}

// StartServer starts the WebRTC server on the specified port
func StartServer(addr string) error {
	server := NewWebRTCServer()
	http.HandleFunc("/session", server.HandleNegotiate)
	log.Printf("WebRTC server starting on %s", addr)
	return http.ListenAndServe(addr, nil)
}
