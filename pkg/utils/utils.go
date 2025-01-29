package utils

// []int16 -> []byte (小端)
func Int16SliceToByteSlice(samples []int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, v := range samples {
		// 小端序：低位字节在前，高位字节在后
		out[2*i] = byte(v)        // 低位
		out[2*i+1] = byte(v >> 8) // 高位
	}
	return out
}

// []byte -> []int16 (小端)
func ByteSliceToInt16Slice(data []byte) []int16 {

	if len(data)%2 != 0 {
		panic("audio data length must be multiple of 2")
	}
	sampleCount := len(data) / 2
	samples := make([]int16, sampleCount)

	for i := 0; i < sampleCount; i++ {
		// 小端序：低位在前，高位在后
		low := data[2*i]
		high := data[2*i+1]
		// 合并成 int16
		samples[i] = int16(low) | int16(high)<<8
	}

	return samples
}
