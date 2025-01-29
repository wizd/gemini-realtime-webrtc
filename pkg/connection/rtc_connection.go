package connection

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/realtime-ai/gemini-realtime-webrtc/pkg/elements"
	"github.com/realtime-ai/gemini-realtime-webrtc/pkg/pipeline"
	"google.golang.org/genai"
)

const (
	sampleRate    = 48000
	channels      = 2
	frameSize     = 960      // 20ms @ 48kHz
	opusFrameSize = 960      // 20ms @ 48kHz
	maxDataBytes  = 1000 * 2 // Buffer for Opus encoded data
)

type RTCConnectionWrapper struct {
	id               string
	genaiSession     *genai.Session
	pc               *webrtc.PeerConnection
	dataChannel      *webrtc.DataChannel
	remoteAudioTrack *webrtc.TrackRemote
	localAudioTrack  *webrtc.TrackLocalStaticSample

	webrtcSinkElement       *elements.WebRTCSinkElement
	opusDecodeElement       *elements.OpusDecodeElement
	opusEncodeElement       *elements.OpusEncodeElement
	inAudioResampleElement  *elements.AudioResampleElement
	outAudioResampleElement *elements.AudioResampleElement
	geminiElement           *elements.GeminiElement

	pipeline *pipeline.Pipeline

	cancel context.CancelFunc
	ctx    context.Context // 供整个 PeerConnection 生命周期使用
}

func NewRTCConnectionWrapper(id string, pc *webrtc.PeerConnection) *RTCConnectionWrapper {

	ctx, cancel := context.WithCancel(context.Background())

	return &RTCConnectionWrapper{
		id:          id,
		pc:          pc,
		cancel:      cancel,
		ctx:         ctx,
		dataChannel: nil,
	}
}

func (c *RTCConnectionWrapper) InitAISession(ctx context.Context) error {

	apiKey := os.Getenv("GOOGLE_API_KEY")

	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey, Backend: genai.BackendGoogleAI})
	if err != nil {
		log.Fatal("create client error: ", err)
		return err
	}

	session, err := client.Live.Connect("gemini-2.0-flash-exp", &genai.LiveConnectConfig{
		ResponseModalities: []string{"AUDIO"},
	})
	if err != nil {
		log.Fatal("connect to model error: ", err)
		return err
	}

	c.genaiSession = session

	return nil
}

func (c *RTCConnectionWrapper) Start(ctx context.Context, pc *webrtc.PeerConnection) error {

	c.pc = pc

	pc.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("DataChannel created: %s", d.Label())

		c.dataChannel = d

		go c.readDataChannel(ctx)
	})

	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("OnTrack: %v, codec: %v", track.ID(), track.Codec().MimeType)
		if track.Kind() == webrtc.RTPCodecTypeAudio {
			c.remoteAudioTrack = track
			go c.readRemoteAudio(ctx)
		}
	})

	audioTrack, audioTrackErr := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion")
	if audioTrackErr != nil {
		log.Println("create local audio track error:", audioTrackErr)
		return audioTrackErr
	}
	c.localAudioTrack = audioTrack

	pc.AddTransceiverFromTrack(c.localAudioTrack, webrtc.RTPTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionSendrecv,
	})

	webrtcSinkElement := elements.NewWebRTCSinkElement(100, c.localAudioTrack)
	geminiElement := elements.NewGeminiElement()
	geminiElement.SetSession(c.genaiSession)

	opusDecodeElement := elements.NewOpusDecodeElement(100, 48000, 1)
	inAudioResampleElement := elements.NewAudioResampleElement(48000, 16000, 1, 1)

	elements := []pipeline.Element{
		opusDecodeElement,
		inAudioResampleElement,
		geminiElement,
		webrtcSinkElement,
	}

	pipeline := pipeline.NewPipeline(elements)
	pipeline.Link(opusDecodeElement, inAudioResampleElement)
	pipeline.Link(inAudioResampleElement, geminiElement)
	pipeline.Link(geminiElement, webrtcSinkElement)

	c.webrtcSinkElement = webrtcSinkElement
	c.opusDecodeElement = opusDecodeElement
	c.inAudioResampleElement = inAudioResampleElement
	c.geminiElement = geminiElement

	c.pipeline = pipeline

	return pipeline.Start(ctx)
}

func (c *RTCConnectionWrapper) Stop() error {
	return c.pipeline.Stop()
}

func (c *RTCConnectionWrapper) readRemoteAudio(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		default:
			rtpPacket, _, err := c.remoteAudioTrack.ReadRTP()
			if err != nil {
				log.Println("read RTP error:", err)
				continue
			}

			// 将拿到的 payload 投递给 pipeline 的“输入 element”
			msg := pipeline.PipelineMessage{
				Type: pipeline.MsgTypeAudio,
				AudioData: &pipeline.AudioData{
					Data:       rtpPacket.Payload,
					SampleRate: 48000,
					Channels:   1,
					MediaType:  "audio/x-opus",
					Codec:      "opus",
					Timestamp:  time.Now(),
				},
			}

			c.opusDecodeElement.In() <- msg
		}
	}
}

func (c *RTCConnectionWrapper) readDataChannel(ctx context.Context) {

	defer c.dataChannel.Close()

	c.dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {

		message := msg.Data

		var sendMessage genai.LiveClientMessage
		if err := json.Unmarshal(message, &sendMessage); err != nil {
			log.Println("unmarshal message error ", string(message), err)
			return
		}
		c.genaiSession.Send(&sendMessage)
	})

	<-ctx.Done()
}
