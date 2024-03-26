package virtual

import (
	"bufio"
	"io"
	"os/exec"
	"strings"

	"github.com/jasonkolodziej/go-castv2/sps"
	"github.com/jasonkolodziej/go-castv2/sps/parse"
)

var defaultConfPath = func(zoneName string) string {
	return "/etc/shairport-sync" + zoneName + ".conf"
}

var defaultSystemDServicePath = func(zoneName string) string {
	return "/etc/shairport-sync" + zoneName + ".conf"
}

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
		|||
		| ffmpeg -y -re -fflags nobuffer -f s16le -ac 2 -ar 44100 -i pipe:0 -c:a adts pipe:1
*/
var ffmpegArgs = []string{
	// * arguments
	"-y",
	"-re",
	"-fflags",  // * AVOption flags (default 200)
	"nobuffer", // * reduce the latency introduced by optional buffering
	"-f", "s16le",
	"-ar", "44100",
	"-ac", "2",
	// "-re",         // * encode at 1x playback speed, to not burn the CPU
	"-i", "pipe:", // * input from pipe (stdout->stdin)
	// "-ar", "44100", // * AV sampling rate
	// "-c:a", "flac", // * audio codec
	// "-sample_fmt", "44100", // * sampling rate
	"-ac", "2", // * audio channels, chromecasts don't support more than two audio channels
	// "-f", "mp4", // * fmt force format
	"-bits_per_raw_sample", "8",
	"-movflags", "frag_keyframe+empty_moov", //? https://github.com/fluent-ffmpeg/node-fluent-ffmpeg/issues/967#issuecomment-888843722
	"-f", "adts",
	"pipe:1", // * output to pipe (stdout->) //TODO: this will need to be a file with mov flags
}

func (v *VirtualDevice) Virtualize() error {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	var err error
	// * Start SPS
	// n := strings.ReplaceAll(v.ZoneName(), " ", "") // * default device configuration file
	v.ZoneName()
	n := "/etc/shairport-syncKitchenSpeaker.conf"
	if v.sps != nil { // prevent from being spawned multiple times
		return nil
	}
	v.sps = exec.CommandContext(
		v.ctx,
		"shairport-sync",
		"-c", n,
	)
	v.rawContent, err = v.sps.StdoutPipe() // * assign ffmpeg stdin to sps stdout
	if err != nil {
		z.Err(err).Msg("error: shairport-sync StdoutPipe()")
		v.sps.Cancel()
		return err
	}

	errno, err := v.sps.StderrPipe()
	if err != nil {
		z.Err(err).Msg("error: shairport-sync StderrPipe()")
		v.sps.Cancel()
		return err
	}
	go WriteStdErrnoToLog(errno) // * start collecting the logs of SPS
	//v.content <- out
	return v.sps.Start()
	// err := v.ConnectDeviceToVirtualStream("http://192.168.2.14:3080")
	// return err
}

func (v *VirtualDevice) StartTranscoder() error {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	var err error
	if v.ffmpeg != nil {
		z.Warn().Any("StartTranscoder", "ffmpeg already running").Msg("skipping")
		return nil
	}
	if v.sps == nil {
		z.Err(err).Msg("error: VirtualDevice.sps *exec.Cmd was lost")
		return err
	}
	if v.rawContent == nil {
		z.Err(err).Msg("error: VirtualDevice.rawContent interface was nil")
		return err
	}
	v.ffmpeg = exec.CommandContext(
		v.ctx,
		"ffmpeg",
		ffmpegArgs...,
	)
	v.ffmpeg.Stdin = v.rawContent
	errno, err := v.ffmpeg.StderrPipe()
	if err != nil {
		z.Err(err).Msg("error: ffmpeg StderrPipe()")
		v.ffmpeg.Cancel()
		return err
	}
	go WriteStdErrnoToLog(errno)
	v.content, err = v.ffmpeg.StdoutPipe()
	if err != nil {
		z.Err(err).Msg("error: ffmpeg StdoutPipe()")
		v.ffmpeg.Cancel()
		return err
	}
	// v.content <- out

	return v.ffmpeg.Start()
}

func (v *VirtualDevice) StopTranscoder() error {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	// ? https://stackoverflow.com/questions/69954944/capture-stdout-from-exec-command-line-by-line-and-also-pipe-to-os-stdout
	// rc := <-v.content        // * Get the io.ReadCloser
	// defer v.content.Close()  // * defer closing
	_ = v.ffmpeg.Wait()      // * wait until full disconnect
	return v.ffmpeg.Cancel() // * call cancel then defer
}

// WriteStdErrnoToLog will defer closing errno, io.ReadCloser
func WriteStdErrnoToLog(errno io.ReadCloser) {
	defer errno.Close()
	scanner := bufio.NewScanner(errno)
	for scanner.Scan() {
		z.Warn().Msg(scanner.Text())
	}
}

func (v *VirtualDevice) checkForConfigFile() {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	f, _, err := parse.LoadFile("", "shairport-sync"+v.ZoneName()+".conf")
	if err == nil {
		f.Close() // * file exists
		return
	}
	// * Create a new config
	f, _, err = sps.OpenOriginalConfig()
	if err != nil {
		z.Fatal().AnErr("checkForConfigFile", err).Msg("attempting to create a new config")
	}
	kvTempl := parse.KeyValue{}
	kvTempl.SetDelimiters("=", ";", "/ ")
	sections, err := parse.ParseOpenedFile(f, func() (kvTemplate *parse.KeyValue,
		sectionStartDel string, sectionNameDel string, endSectionDel string) {
		return &kvTempl,
			"{", " =", "};"
	})
	if err != nil {
		z.Fatal().AnErr("checkForConfigFile", err).Msg("parsing error")
	}
	if err = sections.UpdateValueAt("general.output_backend", "stdout"); err != nil {
		z.Fatal().AnErr("checkForConfigFile", err).Msg("editing error")
	}
	if err = sections.UpdateValueAt("general.port", 8009); err != nil {
		z.Fatal().AnErr("checkForConfigFile", err).Msg("editing error")
	}
	_, name := v.Info.AirplayDeviceName()
	if err = sections.UpdateValueAt("general.name", name); err != nil {
		z.Fatal().AnErr("checkForConfigFile", err).Msg("editing error")
	}
	_, id := v.Info.AirplayDeviceId()
	ids := "0x" + strings.ToUpper(strings.ReplaceAll(id.String(), ":", "")) + "L"
	if err = sections.UpdateValueAt("general.airplay_device_id", ids); err != nil {
		z.Fatal().AnErr("checkForConfigFile", err).Msg("editing error")
	}
	// TODO: Lines UDP ports? `udp_port_base` ! states only for airplay 1
	// TODO: Lines for handling session control
	// ! run_this_before_play_begins, run_this_after_play_ends,
	// ! run_this_before_entering_active_state, run_this_after_exiting_active_state
	// TODO: Lines for handling volume up/down
	//	run_this_when_volume_is_set = "/full/path/to/application/and/args";
	//	Run the specified application whenever the volume control is set or changed.
	//		The desired AirPlay volume is appended to the end of the command line –
	// 		leave a space if you want it treated as an extra argument.
	//		AirPlay volume goes from 0.0 to -30.0 and -144.0 means "mute".
}
