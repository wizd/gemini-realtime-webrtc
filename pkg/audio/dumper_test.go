package audio

import (
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
		assert.Contains(t, dumper.GetFilename(), "48000Hz_2ch.pcm")
		assert.Contains(t, dumper.GetFilename(), time.Now().Format("20060102"))
	})

	t.Run("Write data", func(t *testing.T) {
		testData := []byte{1, 2, 3, 4}
		err := dumper.Write(testData)
		assert.NoError(t, err)

		// 验证文件大小
		info, err := os.Stat(dumper.GetFilename())
		assert.NoError(t, err)
		assert.Equal(t, int64(len(testData)), info.Size())
	})

	t.Run("Write multiple times", func(t *testing.T) {
		testData1 := []byte{5, 6, 7, 8}
		testData2 := []byte{9, 10, 11, 12}

		err := dumper.Write(testData1)
		assert.NoError(t, err)
		err = dumper.Write(testData2)
		assert.NoError(t, err)

		// 验证文件大小
		info, err := os.Stat(dumper.GetFilename())
		assert.NoError(t, err)
		assert.Equal(t, int64(12), info.Size()) // 4 + 4 + 4 bytes
	})

	t.Run("Close and write", func(t *testing.T) {
		err := dumper.Close()
		assert.NoError(t, err)

		// 尝试写入已关闭的dumper
		err = dumper.Write([]byte{1, 2, 3, 4})
		assert.Error(t, err)
	})
}

func TestDumperInvalidPath(t *testing.T) {
	// 尝试在不存在的目录创建dumper
	_, err := NewDumper("test", -1, -1)
	assert.NoError(t, err) // 应该能创建文件，因为是在当前目录

	// 清理测试文件
	files, err := os.ReadDir(".")
	require.NoError(t, err)
	for _, file := range files {
		if len(file.Name()) > 4 && file.Name()[len(file.Name())-4:] == ".pcm" {
			os.Remove(file.Name())
		}
	}
}
