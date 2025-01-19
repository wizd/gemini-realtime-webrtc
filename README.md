# Gemini Realtime WebRTC


Gemini Realtime API with WebRTC， Like OpenAI Realtime API with WebRTC.



## Features

- Real-time voice communication with Gemini AI
- High-quality audio processing:
  - 48kHz sample rate support
  - Opus codec for efficient audio compression
  - Automatic audio resampling
  - Smart audio buffering with 100ms accumulation
- WebRTC-based communication:
  - Low-latency audio streaming
  - Reliable data channel for text messages
- Debug capabilities:
  - Configurable audio dumping for all streams
  - Detailed logging
  - PCM file format support

## Prerequisites

- Go 1.21 or higher
- FFmpeg libraries (for audio processing)
- Opus codec library
- Google API Key for Gemini AI

## Installation

1. Install system dependencies:

```bash
# For Debian/Ubuntu
apt-get install pkg-config libopus-dev libavcodec-dev libavformat-dev libavutil-dev libswresample-dev

# For macOS
brew install opus ffmpeg
```

2. Clone the repository:

```bash
git clone https://github.com/realtime-ai/gemini-realtime-webrtc.git
cd gemini-realtime-webrtc
```

3. Install Go dependencies:

```bash
go mod download
```

## Configuration

1. Set up environment variables:

```bash
# Required
export GOOGLE_API_KEY=your_api_key_here

# Optional (for audio debugging)
export DUMP_SESSION_AUDIO=true  # Dump AI response audio
export DUMP_REMOTE_AUDIO=true   # Dump user input audio
export DUMP_LOCAL_AUDIO=true    # Dump playback audio
```

## Running the Application

1. Start the server:

```bash
go run main.go
```

2. Open the web client:
   - Navigate to `tests/gemini_realtime_webrtc.html` in your browser
   - Click "Connect" to establish WebRTC connection
   - Allow microphone access when prompted

## Architecture

- `pkg/gateway`: WebRTC server and connection management
- `pkg/audio`: Audio processing utilities
  - Resampling between different sample rates
  - Audio buffering with smart accumulation
  - PCM/WAV file handling
- `pkg/utils`: Common utilities and helper functions

## Development

### Building from source:

```bash
go build -o server
```

### Running tests:

```bash
go test ./...
```


## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

