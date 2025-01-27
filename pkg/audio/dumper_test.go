package audio

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDumper(t *testing.T) {
	// 创建 Dumper
	dumper, err := NewDumper("test", 48000, 2)
	require.NoError(t, err)
	defer dumper.Close()
	defer os.Remove(dumper.GetFilename()) // 清理测试文件

	t.Run("Check initial state", func(t *testing.T) {
		assert.Equal(t, 48000, dumper.GetSampleRate())
		assert.Equal(t, 2, dumper.GetChannels())
		assert.Contains(t, dumper.GetFilename(), "48000Hz_2ch.wav")
		assert.Contains(t, dumper.GetFilename(), time.Now().Format("20060102"))
	})

	t.Run("Write data", func(t *testing.T) {
		testData := []byte{1, 2, 3, 4}
		err := dumper.Write(testData)
		assert.NoError(t, err)

		// 验证文件存在
		_, err = os.Stat(dumper.GetFilename())
		assert.NoError(t, err)
	})

	t.Run("Write multiple times", func(t *testing.T) {
		testData1 := []byte{5, 6, 7, 8}
		testData2 := []byte{9, 10, 11, 12}

		err := dumper.Write(testData1)
		assert.NoError(t, err)
		err = dumper.Write(testData2)
		assert.NoError(t, err)

		// 验证文件存在
		_, err = os.Stat(dumper.GetFilename())
		assert.NoError(t, err)
	})

	t.Run("Close and write", func(t *testing.T) {
		err := dumper.Close()
		assert.NoError(t, err)

		// 尝试写入已关闭的dumper
		err = dumper.Write([]byte{1, 2, 3, 4})
		assert.Error(t, err)
	})
}

func TestDumperWithSineWave(t *testing.T) {
	// 创建一个1秒钟的1kHz正弦波
	sampleRate := 48000
	duration := 1       // 1秒
	frequency := 1000.0 // 1kHz
	numSamples := sampleRate * duration

	// 生成正弦波数据
	samples := make([]byte, numSamples*2) // 16-bit samples = 2 bytes per sample
	for i := 0; i < numSamples; i++ {
		// 生成-32768到32767范围的正弦波
		value := int16(32767.0 * math.Sin(2.0*math.Pi*frequency*float64(i)/float64(sampleRate)))
		// 转换为字节
		samples[i*2] = byte(value & 0xFF)
		samples[i*2+1] = byte((value >> 8) & 0xFF)
	}

	// 创建dumper并写入数据
	dumper, err := NewDumper("sine", sampleRate, 1)
	require.NoError(t, err)
	defer os.Remove(dumper.GetFilename())

	// 分多次写入数据以模拟实时输入
	chunkSize := 960 // 20ms at 48kHz
	for i := 0; i < len(samples); i += chunkSize {
		end := i + chunkSize
		if end > len(samples) {
			end = len(samples)
		}
		err = dumper.Write(samples[i:end])
		assert.NoError(t, err)
	}

	// 关闭dumper
	err = dumper.Close()
	assert.NoError(t, err)

	// 验证文件存在且大小合理
	info, err := os.Stat(dumper.GetFilename())
	assert.NoError(t, err)
	// WAV文件大小应该是音频数据大小 + WAV头大小
	expectedMinSize := numSamples*2 + 44 // 44 bytes is standard WAV header size
	assert.Equal(t, info.Size(), int64(expectedMinSize))
}
