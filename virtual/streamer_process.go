package virtual

import (
	"fmt"
	"os"
	"os/exec"
)

/*
* ffmpegArgs https://ffmpeg.org/ffmpeg-protocols.html#toc-pipe
? (e.g. 0 for stdin, 1 for stdout, 2 for stderr).
  - $ ffmpeg -formats | grep PCM
  - DE alaw            PCM A-law
  - DE f32be           PCM 32-bit floating-point big-endian
  - DE f32le           PCM 32-bit floating-point little-endian
  - DE f64be           PCM 64-bit floating-point big-endian
  - DE f64le           PCM 64-bit floating-point little-endian
  - DE mulaw           PCM mu-law
  - DE s16be           PCM signed 16-bit big-endian
  - DE s16le           PCM signed 16-bit little-endian
  - DE s24be           PCM signed 24-bit big-endian
  - DE s24le           PCM signed 24-bit little-endian
  - DE s32be           PCM signed 32-bit big-endian
  - DE s32le           PCM signed 32-bit little-endian
  - DE s8              PCM signed 8-bit
  - DE u16be           PCM unsigned 16-bit big-endian
  - DE u16le           PCM unsigned 16-bit little-endian
  - DE u24be           PCM unsigned 24-bit big-endian
  - DE u24le           PCM unsigned 24-bit little-endian
  - DE u32be           PCM unsigned 32-bit big-endian
  - DE u32le           PCM unsigned 32-bit little-endian
  - DE u8              PCM unsigned 8-bit

Example:

	shairport-sync -c /etc/shairport-syncKitchenSpeaker.conf -o stdout \
		| ffmpeg -f s16le -ar 44100 -ac 2 -i pipe: -ac 2 -bits_per_raw_sample 8 -c:a flac -y flac_test1.flac
*/
var ffmpegArgs = []string{
	// * arguments
	"-f", "s16le", // * format as RAW signed 16le
	"-ar", "44100", // * rate set audio sampling rate (in Hz)
	"-ac", "2", // * channels set number of audio channels
	"-re",         // * encode at 1x playback speed, to not burn the CPU
	"-i", "pipe:", // * input from pipe (stdout->stdin)
	// "-ar", "44100", // * AV sampling rate
	"-c:a", "flac", // * audio codec
	// "-sample_fmt", "44100", // * sampling rate
	"-ac", "2", // * audio channels, chromecasts don't support more than two audio channels
	// "-f", "mp4", // * fmt force format
	"-bits_per_raw_sample", "8",
	"-f", "flac",
	// "-movflags", "frag_keyframe+faststart",
	// "-strict", // * how strictly to follow the standards (from INT_MIN to INT_MAX) (default 0)
	// "experimental", // *  allow non-standardized experimental things
	"pipe:1", // * output to pipe (stdout->)
}

// IProcess is an interface around the FFMPEG process
type IProcess interface {
	Spawn(path, URI string) *exec.Cmd
}

// ProcessLoggingOpts describes options for process logging
type ProcessLoggingOpts struct {
	Enabled    bool   // Option to set logging for transcoding processes
	Directory  string // Directory for the logs
	MaxSize    int    // Maximum size of kept logging files in megabytes
	MaxBackups int    // Maximum number of old log files to retain
	MaxAge     int    // Maximum number of days to retain an old log file.
	Compress   bool   // Indicates if the log rotation should compress the log files
}

// Process is the main type for creating new processes
type Process struct {
	keepFiles bool
	audio     bool
	codec     string
}

// Type check
var _ IProcess = (*Process)(nil)

// NewProcess creates a new process able to spawn transcoding FFMPEG processes
func NewProcess(
	keepFiles bool,
	audio bool,
	codec string,
) *Process {
	return &Process{keepFiles, audio, codec}
}

// getHLSFlags are for getting the flags based on the config context
func (p Process) getHLSFlags() string {
	if p.keepFiles {
		return "append_list"
	}
	return "delete_segments+append_list"
}

// Spawn creates a new FFMPEG cmd
func (p Process) Spawn(path, URI string) *exec.Cmd {
	os.MkdirAll(path, os.ModePerm)
	processCommands := []string{
		"-y",
		"-fflags",         // * AVOption flags (default 200)
		"nobuffer",        // * reduce the latency introduced by optional buffering
		"-rtsp_transport", // * set RTSP transport protocols (default 0)
		"tcp",
		"-i",
		URI,
		"-f",    // * format
		"lavfi", //*  Libavfilter input virtual device.
		//* This input device reads data from the open output pads of a libavfilter filtergraph.
		"-i",
		"anullsrc=channel_layout=stereo:sample_rate=44100", // * anullsrc
		"-vsync", // *video sync method
		"0",
		"-copyts", // * copy timestamps
		"-vcodec", // * force video codec (‘copy’ to copy stream)
		p.codec,
		"-movflags",                // * MOV muxer flags (default 0)
		"frag_keyframe+empty_moov", // * Fragment at video keyframes & Make the initial moov atom empty (not supported by QuickTime)
	}
	if !p.audio {
		processCommands = append(processCommands, "-an")
	}
	processCommands = append(processCommands,
		"-hls_flags",
		p.getHLSFlags(),
		"-f",
		"hls",
		"-segment_list_flags", // * set flags affecting segment list generation (default 1)
		"live",                // * enable live-friendly list generation (useful for HLS)
		"-hls_time",           // * set segment length in seconds (from 0 to FLT_MAX) (default 2)
		"1",
		"-hls_list_size", // * set maximum number of playlist entries (from 0 to INT_MAX) (default 5)
		"3",
		"-hls_segment_filename",
		fmt.Sprintf("%s/%%d.ts", path),
		fmt.Sprintf("%s/index.m3u8", path),
	)
	cmd := exec.Command("ffmpeg", processCommands...)
	return cmd
}

/*

shairport-sync -c /etc/shairport-syncKitchenSpeaker.conf | ffmpeg -y -re -fflags nobuffer -f s16le -ac 2 -ar 44100 -i pipe:0 -c:a aac output.aac

-f segment -segment_time 10 -segment_list outputlist.m3u8 -segment_format mpegts output%03d.ts

*/
// -c:a libmp3lame - audio codec
// -b:a 128k bitrate
