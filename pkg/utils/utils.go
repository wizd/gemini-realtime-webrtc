package utils

import (
	"bytes"
	"encoding/binary"
)

// []int16 -> []byte (小端)
func Int16SliceToByteSlice(samples []int16) []byte {
	buf := new(bytes.Buffer)
	// 将 samples 按照 LittleEndian（小端）写入
	// 注意：需要保证写入类型与切片实际类型相符
	_ = binary.Write(buf, binary.LittleEndian, samples)
	return buf.Bytes()
}

// []byte -> []int16 (小端)
func ByteSliceToInt16Slice(data []byte) []int16 {
	buf := bytes.NewReader(data)
	// 由于不知道具体元素数量，可先计算可能的数量，然后 make 适当长度的切片
	count := len(data) / 2
	out := make([]int16, count)
	// 从 buf 中按 LittleEndian 读取到 out 切片中
	_ = binary.Read(buf, binary.LittleEndian, out)
	return out
}
