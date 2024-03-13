/*
	Package scanner utlizes pcap to detect the MAC address of remote devices on a local network

See "[Issue with libpcap]".

[Issue with libpcap]:(https://github.com/google/gopacket/issues/280#issuecomment-410145559)

! requires `apt-get install ffmpeg` linux TODO: add golang build constraints
*/
package hls

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/jasonkolodziej/go-castv2/controllers/media"
	logg "github.com/sirupsen/logrus"

	"github.com/pkg/errors"
)

var httpServer *http.ServeMux = nil
var serverDebug bool = false

type LocalMediaDataBuilder struct {
	media.MediaDataBuilder
	transcode bool
}

type LocalMediaData struct {
	media.StandardMediaMetadata
}

func (builder *LocalMediaDataBuilder) SetCustomData(custom map[string]interface{}) {
	builder.MediaDataBuilder.SetCustomData(custom)
}

func log(message string, args ...interface{}) {
	if serverDebug {
		logg.WithField("package", "application").Infof(message, args...)
	}
}

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

func startStreamingServer(serverPort int) error {
	if httpServer != nil {
		return nil
	}
	log("trying to find available port to start streaming server on")

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(int(serverPort)))
	if err != nil {
		return errors.Wrap(err, "unable to bind to local tcp address")
	}

	serverPort = listener.Addr().(*net.TCPAddr).Port
	log("found available port :%d", serverPort)

	httpServer = http.NewServeMux()

	httpServer.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check to see if we have a 'filename' and if it is one of the ones that have
		// already been validated and is useable.
		filename := r.URL.Query().Get("media_file")
		canServe := false
		// for _, fn := range mediaFilenames {
		// 	if fn == filename {
		// 		canServe = true
		// 	}
		// }

		// playedItems[filename] = PlayedItem{ContentID: filename, Started: time.Now().Unix()}
		// writePlayedItems()

		// Check to see if this is a live streaming video and we need to use an
		// infinite range request / response. This comes from media that is either
		// live or currently being transcoded to a different media format.
		liveStreaming := false
		if ls := r.URL.Query().Get("live_streaming"); ls == "true" {
			liveStreaming = true
		}

		log("canServe=%t, liveStreaming=%t, filename=%s", canServe, liveStreaming, filename)
		if canServe {
			if !liveStreaming {
				http.ServeFile(w, r, filename)
			} else {
				serveLiveStreaming(w, r, filename)
			}
		} else {
			http.Error(w, "Invalid file", 400)
		}
		log("method=%s, headers=%v, reponse_headers=%v", r.Method, r.Header, w.Header())
		// pi := playedItems[filename]

		// TODO(vishen): make this a pointer?
		// pi.Finished = time.Now().Unix()
		// playedItems[filename] = pi
		// writePlayedItems()
	})

	go func() {
		log("media server listening on %d", serverPort)
		if err := http.Serve(listener, httpServer); err != nil && err != http.ErrServerClosed {
			logg.WithField("package", "application").WithError(err).Fatal("error serving HTTP")
		}
	}()

	return nil
}

func serveLiveStreaming(w http.ResponseWriter, r *http.Request, filename string) {
	cmd := exec.Command(
		"ffmpeg",
		"-re", // encode at 1x playback speed, to not burn the CPU
		"-i", filename,
		"-vcodec", "h264",
		"-acodec", "aac",
		"-ac", "2", // chromecasts don't support more than two audio channels
		"-f", "mp4",
		"-movflags", "frag_keyframe+faststart",
		"-strict", "-experimental",
		"pipe:1",
	)

	cmd.Stdout = w
	if serverDebug {
		cmd.Stderr = os.Stderr
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Transfer-Encoding", "chunked")

	if err := cmd.Run(); err != nil {
		logg.WithField("package", "application").WithFields(logg.Fields{
			"filename": filename,
		}).WithError(err).Error("error transcoding")
	}
}
