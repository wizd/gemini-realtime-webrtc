package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type WavStreamWriter struct {
	file          *os.File
	sampleRate    uint32
	numChannels   uint16
	bitsPerSample uint16
	dataBytes     uint32
	closed        bool
}

// NewWavStreamWriter 创建并初始化一个 WAV 文件（写入头部占位）
func NewWavStreamWriter(filename string, sampleRate uint32, numChannels, bitsPerSample uint16) (*WavStreamWriter, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	w := &WavStreamWriter{
		file:          f,
		sampleRate:    sampleRate,
		numChannels:   numChannels,
		bitsPerSample: bitsPerSample,
		dataBytes:     0,
		closed:        false,
	}

	// 写初始头部（用占位值）
	err = w.writeInitialHeader()
	if err != nil {
		f.Close()
		return nil, err
	}

	return w, nil
}

// Write 往 WAV 文件追加写入PCM数据，并**每次**更新头部长度
func (w *WavStreamWriter) Write(pcm []byte) (int, error) {
	if w.closed {
		return 0, fmt.Errorf("WavStreamWriter: 已关闭，不能再写入")
	}
	n, err := w.file.Write(pcm)
	if err != nil {
		return n, err
	}
	w.dataBytes += uint32(n)

	// 关键之处：每次写完，都回头更新头部
	if err := w.updateHeader(); err != nil {
		return n, fmt.Errorf("更新WAV头部失败: %v", err)
	}

	return n, nil
}

// Close 只做关闭动作（此时文件头已经在 Write() 时就不断更新了）
func (w *WavStreamWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	return w.file.Close()
}

// ------------------------ 内部方法 ------------------------

func (w *WavStreamWriter) writeInitialHeader() error {
	// 1. "RIFF"
	if _, err := w.file.Write([]byte("RIFF")); err != nil {
		return err
	}
	// 2. ChunkSize 占位
	if err := binary.Write(w.file, binary.LittleEndian, uint32(0)); err != nil {
		return err
	}
	// 3. "WAVE"
	if _, err := w.file.Write([]byte("WAVE")); err != nil {
		return err
	}
	// 4. "fmt "
	if _, err := w.file.Write([]byte("fmt ")); err != nil {
		return err
	}
	// 5. Subchunk1Size (16)
	if err := binary.Write(w.file, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	// 6. AudioFormat = 1 (PCM)
	if err := binary.Write(w.file, binary.LittleEndian, uint16(1)); err != nil {
		return err
	}
	// 7. NumChannels
	if err := binary.Write(w.file, binary.LittleEndian, w.numChannels); err != nil {
		return err
	}
	// 8. SampleRate
	if err := binary.Write(w.file, binary.LittleEndian, w.sampleRate); err != nil {
		return err
	}
	// 9. ByteRate
	byteRate := w.sampleRate * uint32(w.numChannels) * uint32(w.bitsPerSample/8)
	if err := binary.Write(w.file, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	// 10. BlockAlign
	blockAlign := w.numChannels * w.bitsPerSample / 8
	if err := binary.Write(w.file, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	// 11. BitsPerSample
	if err := binary.Write(w.file, binary.LittleEndian, w.bitsPerSample); err != nil {
		return err
	}
	// 12. "data"
	if _, err := w.file.Write([]byte("data")); err != nil {
		return err
	}
	// 13. Subchunk2Size 占位
	if err := binary.Write(w.file, binary.LittleEndian, uint32(0)); err != nil {
		return err
	}
	return nil
}

// updateHeader 用于在每次写完后回填最新的数据长度
func (w *WavStreamWriter) updateHeader() error {
	chunkSize := 36 + w.dataBytes // 36 = 44 - 8
	// 回到 offset=4 写 chunkSize
	if _, err := w.file.Seek(4, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(w.file, binary.LittleEndian, chunkSize); err != nil {
		return err
	}

	// 回到 offset=40 写 Subchunk2Size
	if _, err := w.file.Seek(40, io.SeekStart); err != nil {
		return err
	}
	if err := binary.Write(w.file, binary.LittleEndian, w.dataBytes); err != nil {
		return err
	}

	// 再次 seek 到文件末尾，方便下一次 Write 在正确位置追加
	if _, err := w.file.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	return nil
}

// Dumper 用于保存音频数据到WAV文件
type Dumper struct {
	sampleRate int // 采样率
	channels   int // 通道数
	writer     *WavStreamWriter
	mu         sync.Mutex
	filename   string
}

// NewDumper 创建新的音频数据保存器
func NewDumper(tag string, sampleRate, channels int) (*Dumper, error) {
	// 生成文件名：tag_timestamp_samplerate_channels.wav
	filename := fmt.Sprintf("tag_%s_audio_%s_%dHz_%dch.wav",
		tag,
		time.Now().Format("20060102_150405"),
		sampleRate,
		channels)

	writer, err := NewWavStreamWriter(filename, uint32(sampleRate), uint16(channels), 16)
	if err != nil {
		return nil, fmt.Errorf("创建WavStreamWriter失败: %w", err)
	}

	return &Dumper{
		sampleRate: sampleRate,
		channels:   channels,
		writer:     writer,
		filename:   filename,
	}, nil
}

// Write 写入音频数据
func (d *Dumper) Write(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.writer == nil {
		return fmt.Errorf("dumper已关闭")
	}

	// 写入WAV文件
	if _, err := d.writer.Write(data); err != nil {
		return fmt.Errorf("写入WAV数据失败: %w", err)
	}

	return nil
}

// Close 关闭文件
func (d *Dumper) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.writer != nil {
		if err := d.writer.Close(); err != nil {
			return fmt.Errorf("关闭文件失败: %w", err)
		}
		d.writer = nil
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
