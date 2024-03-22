package virtual

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jasonkolodziej/go-castv2"
	"github.com/jasonkolodziej/go-castv2/sps"
	"github.com/rs/zerolog"
)

var z = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

const spss = "shairport-sync"
const txc = "ffmpeg"

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
	"-f", "s16le",
	"-ar", "44100",
	"-ac", "2",
	// "-re",         // * encode at 1x playback speed, to not burn the CPU
	"-i", "pipe:", // * input from pipe (stdout->stdin)
	// "-ar", "44100", // * AV sampling rate
	"-c:a", "flac", // * audio codec
	// "-sample_fmt", "44100", // * sampling rate
	"-ac", "2", // * audio channels, chromecasts don't support more than two audio channels
	// "-f", "mp4", // * fmt force format
	"-bits_per_raw_sample", "8",
	"-f", "flac",
	"-movflags", "frag_keyframe+faststart",
	"-strict", "-experimental",
	"pipe:1", // * output to pipe (stdout->)
}

type ProcBundle interface {
	Output() (output io.ReadCloser, e io.ReadCloser, err error)
	OutputWithArgs(args ...string) (output io.ReadCloser, e io.ReadCloser, err error)
	Chain(config string) (io.ReadCloser, error)
}

type VirtualDevice struct {
	*castv2.Device
	content io.ReadCloser
	sps     *ProcBundle
	txCoder *ProcBundle // * Transcoder FfMPeg
}

func NewVirtualDevice(d *castv2.Device) *VirtualDevice {
	if d != nil {
		return &VirtualDevice{d, nil, nil, nil}
	}
	return nil
}

func (v *VirtualDevice) ZoneName() string {
	_, n := v.Device.Info.AirplayDeviceName()
	return n
}

func (v *VirtualDevice) Content(rc io.ReadCloser) {
	v.content = rc
}

func (v *VirtualDevice) pathString() string {
	return "/devices/" + v.Info.Id.String() + "/stream.flac"
}

func (v *VirtualDevice) ConnectDeviceToVirtualStream(hostAndPort string) error {
	if v == nil { // * basic sanity check
		return fmt.Errorf("device not created")
	}
	v.QuitApplication(time.Second * 5)
	v.PlayMedia(hostAndPort+v.pathString(), "audio/flac", "LIVE")
	// if err != nil {
	// 	return err
	// }
	return nil
}

func (v *VirtualDevice) Virtualize() error {
	// * Start SPS
	// n := strings.ReplaceAll(v.ZoneName(), " ", "") // * default device configuration file
	n := "/etc/shairport-syncKitchenSpeaker.conf"
	out, spsErr, ffmpegErr, cerr := sps.RunPiping(n) // * exec SPS
	if cerr != nil {
		z.Err(cerr).Send()
	}
	// defer ss.Wait()
	defer spsErr.Close()
	go WriteStdErrnoToLog(spsErr) // * start collecting the logs of SPS
	defer ffmpegErr.Close()
	go WriteStdErrnoToLog(ffmpegErr)
	v.content = out
	return nil
	// err := v.ConnectDeviceToVirtualStream("http://192.168.2.14:3080")
	// return err
}

func WriteStdErrnoToLog(errno io.ReadCloser) {
	scanner := bufio.NewScanner(errno)
	for scanner.Scan() {
		z.Warn().Msg(scanner.Text())
	}
}

func (v *VirtualDevice) FiberDeviceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Params("deviceId") != v.Info.Id.String() {
			return c.SendString("Where is john?")
			// => Hello john
		}
		_, name := v.Info.AirplayDeviceName()
		if strings.Contains(c.Path(), "stream.flac") {
			return c.SendString("Hello " + name + ", I am streaming")
		}
		return c.SendString("Hello " + name)

	}
}

func (v *VirtualDevice) FiberDeviceHandlerWithStream() fiber.Handler {
	// defer v.content.Close()
	return func(c *fiber.Ctx) error {
		_, name := v.Info.AirplayDeviceName() // * Get chromecast Id
		if c.Params("deviceId") != v.Info.Id.String() {
			return c.SendString("Where is john?")
			// => Hello john
		} else if !strings.Contains(c.Path(), "stream.flac") { // * does the path not contain `stream.flac`
			return c.SendString("Hello " + name)
		}
		if v.content == nil {
			return c.SendStatus(501)
		}
		// s, err := v.Chain("") //! do someting else
		// if err != nil {
		// 	c.SendStatus(500)
		// }
		// defer s.Close()
		return c.SendStream(v.content)
	}
}

func (v *VirtualDevice) Output() (output io.ReadCloser, e io.ReadCloser, err error) {
	return nil, nil, nil
}

func (v *VirtualDevice) OutputWithArgs(configPath ...string) (output io.ReadCloser, e io.ReadCloser, err error) {
	var confFlag = append([]string{"-c"}, configPath...)
	return sps.SpawnProcessConfig(confFlag...)
}

// func (v *VirtualDevice) Chain(config string) (io.ReadCloser, error) {
// 	encoded, spsErr, txcErr, cErr := sps.RunPiping(config)
// 	if cErr != nil {
// 		return nil, cErr
// 	}
// 	defer txcErr.Close()
// 	defer spsErr.Close()
// 	return encoded, nil
// }

func SpawnCmdWithContext(ctx context.Context, configPath string) io.ReadCloser {
	ctx, cancel := context.WithCancel(context.Background())
	var err error
	sp := exec.CommandContext(ctx, "shairport-sync", "-c", configPath)
	f := exec.CommandContext(
		ctx,
		txc,
		ffmpegArgs...,
	)
	f.Stdin, err = sp.StdoutPipe() // * assign ffmpeg stdin to sps stdout
	if err != nil {
		z.Err(err).Send()
		cancel()
	}
	errno, err := sp.StderrPipe()
	if err != nil {
		z.Err(err).Send()
		cancel()
	}
	go WriteStdErrnoToLog(errno)
	errno, err = f.StderrPipe()
	if err != nil {
		z.Err(err).Send()
		cancel()
	}
	go WriteStdErrnoToLog(errno)
	output, err := f.StdoutPipe()
	if err != nil {
		z.Err(err).Send()
		cancel()
	}
	err = sp.Start()
	if err != nil {
		z.Err(err).Send()
		cancel()
	}
	err = f.Start()
	if err != nil {
		z.Err(err).Send()
		cancel()
	}
	sp.Wait()
	f.Wait()
	ctx.Done()
	return output
}
