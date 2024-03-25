package virtual

import (
	"fmt"
	"os"
	"os/exec"
)

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


-f segment -segment_time 10 -segment_list outputlist.m3u8 -segment_format mpegts output%03d.ts

*/
// -c:a libmp3lame - audio codec
// -b:a 128k bitrate
