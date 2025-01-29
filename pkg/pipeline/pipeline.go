package pipeline

import (
	"context"
	"sync"
	"time"
)

// https://chatgpt.com/c/678d0634-058c-8002-909d-d298453449e9

// type Pipeline struct {
// 	elements []Element
// }

type AudioData struct {
	Data       []byte
	SampleRate int
	Channels   int
	MediaType  string // "audio/x-raw", "audio/x-opus", etc.
	Codec      string
	Timestamp  time.Time
}

type VideoData struct {
	Data           []byte
	Width          int
	Height         int
	MediaType      string
	Format         string
	FramerateNum   int
	FramerateDenom int
	Codec          string
	Timestamp      time.Time
}

type PipelineMessageType int

const (
	MsgTypeAudio PipelineMessageType = iota
	MsgTypeVideo
	MsgTypeText
)

type PipelineMessage struct {
	Type PipelineMessageType

	// SessionID 会话 ID
	SessionID string
	// Timestamp 时间戳
	Timestamp time.Time

	// AudioData 音频数据块
	AudioData *AudioData

	// Metadata 元数据
	Metadata interface{}
}

type Pipeline struct {
	mu       sync.Mutex
	elements []Element
}

func NewPipeline(elements []Element) *Pipeline {
	return &Pipeline{
		elements: elements,
	}
}

func (p *Pipeline) Link(a, b Element) {
	// a.Out() -> b.In()
	go func() {
		for msg := range a.Out() {
			b.In() <- msg
		}
		close(b.In())
	}()
}

func (p *Pipeline) Start(ctx context.Context) error {
	for _, e := range p.elements {
		if err := e.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pipeline) Stop() error {
	// 倒序停止更稳妥，也可以正序
	for i := len(p.elements) - 1; i >= 0; i-- {
		if err := p.elements[i].Stop(); err != nil {
			return err
		}
	}
	return nil
}
