package audio

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Dumper 用于保存音频数据到文件
type Dumper struct {
	sampleRate int // 采样率
	channels   int // 通道数
	file       *os.File
	mu         sync.Mutex
	filename   string
}

// NewDumper 创建新的音频数据保存器
func NewDumper(tag string, sampleRate, channels int) (*Dumper, error) {
	// 生成文件名：timestamp_samplerate_channels.pcm
	filename := fmt.Sprintf("tag_%s_audio_%s_%dHz_%dch.pcm",
		tag,
		time.Now().Format("20060102_150405"),
		sampleRate,
		channels)

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}

	return &Dumper{
		sampleRate: sampleRate,
		channels:   channels,
		file:       file,
		filename:   filename,
	}, nil
}

// Write 写入音频数据
func (d *Dumper) Write(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.file == nil {
		return fmt.Errorf("dumper已关闭")
	}

	_, err := d.file.Write(data)
	if err != nil {
		return fmt.Errorf("写入数据失败: %w", err)
	}

	d.file.Sync()

	return nil
}

// Close 关闭文件
func (d *Dumper) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.file != nil {
		err := d.file.Close()
		d.file = nil
		if err != nil {
			return fmt.Errorf("关闭文件失败: %w", err)
		}
	}
	return nil
}

// GetFilename 获取当前录制文件的名称
func (d *Dumper) GetFilename() string {
	return d.filename
}

// GetSampleRate 获取采样率
func (d *Dumper) GetSampleRate() int {
	return d.sampleRate
}

// GetChannels 获取通道数
func (d *Dumper) GetChannels() int {
	return d.channels
}
