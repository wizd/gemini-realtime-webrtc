package pipeline

import "context"

type Element interface {
	In() chan<- PipelineMessage
	Out() <-chan PipelineMessage
	Start(ctx context.Context) error
	Stop() error
}

type BaseElement struct {
	InChan  chan PipelineMessage
	OutChan chan PipelineMessage
}

func NewBaseElement(bufferSize int) *BaseElement {
	return &BaseElement{
		InChan:  make(chan PipelineMessage, bufferSize),
		OutChan: make(chan PipelineMessage, bufferSize),
	}
}

func (b *BaseElement) In() chan<- PipelineMessage {
	return b.InChan
}

func (b *BaseElement) Out() <-chan PipelineMessage {
	return b.OutChan
}

func (b *BaseElement) Start(ctx context.Context) error {
	return nil // 具体逻辑由子结构实现
}

func (b *BaseElement) Stop() error {
	return nil
}
