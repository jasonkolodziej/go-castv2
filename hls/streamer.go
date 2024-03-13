package hls

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

func createHLS(inputFile string, outputDir string, segmentDuration int) error {
	// Create the output directory if it does not exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Create the HLS playlist and segment the video using ffmpeg
	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", inputFile,
		"-profile:v", "baseline", // baseline profile is compatible with most devices
		"-level", "3.0",
		"-start_number", "0", // start numbering segments from 0
		"-hls_time", strconv.Itoa(segmentDuration), // duration of each segment in seconds
		"-hls_list_size", "0", // keep all segments in the playlist
		"-f", "hls",
		fmt.Sprintf("%s/playlist.m3u8", outputDir),
	)

	output, err := ffmpegCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create HLS: %v\nOutput: %s", err, string(output))
	}

	return nil
}
