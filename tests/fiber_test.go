package tests

import (
	"context"
	"io"
	"net"
	"os"
	"os/exec"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	cast "github.com/jasonkolodziej/go-castv2"
	"github.com/jasonkolodziej/go-castv2/virtual"
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

var fib = fiber.New(fiber.Config{
	Prefork:       true,
	CaseSensitive: true,
	StrictRouting: true,
	ServerHeader:  "Fiber",
	AppName:       "Test App v1.0.1",
	GETOnly:       true,
})

var middleware = func(c *fiber.Ctx) error {
	// Set a custom header on all responses:
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Custom-Header", "Hello, World")
	// Go to next middleware:
	return c.Next()
}

func TestMain(t *testing.T) {
	// * Fiber router setup

	// * Construct Temp Device
	ip := net.ParseIP("192.168.2.152")
	mac, err := net.ParseMAC("f4:f5:d8:be:cd:ec")
	if err != nil {
		t.Fatal(err)
	}
	var findKitchen = cast.FromServiceEntryInfo(nil, nil, &mac)
	findKitchen.Fn = "Kitchen speaker"
	findKitchen.Id = uuid.MustParse("a548ff5a-d1fa-c194-1101-acb5a1204788")
	findKitchen.IpAddress = &ip
	findKitchen.SetPort(cast.CHROMECAST)
	kitchen, err := cast.NewDeviceFromDeviceInfo(findKitchen)
	if err != nil {
		t.Fatal(err)
	}
	K := virtual.NewVirtualDevice(&kitchen)
	// if err = K.SpawnCmdWithContext(context.Background(), ""); err != nil {
	// 	t.Fatal(err)
	// }
	deviceHandlers := []fiber.Handler{K.FiberDeviceHandlerWithStream()}

	mdev := append([]fiber.Handler{middleware}, deviceHandlers...)
	device := fib.Group("/devices/:deviceId", mdev...)
	device.Get("/disconnect", K.DisconnectDeviceHandler())
	device.Get("/connect", K.ConnectDeviceHandler())
	device.Get("/stream.flac", deviceHandlers...)
	virtual.PrintStack(fib)

	err = fib.Listen(":5123")
	if err != nil {
		t.Fatal(err)
	}

}

func SpawnCmdWithContext(ctx context.Context, configPath string) io.ReadCloser {
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
		sp.Cancel()
	}
	errno, err := sp.StderrPipe()
	if err != nil {
		z.Err(err).Send()
		sp.Cancel()
	}
	go virtual.WriteStdErrnoToLog(errno)
	errno, err = f.StderrPipe()
	if err != nil {
		z.Err(err).Send()
		f.Cancel()
	}
	go virtual.WriteStdErrnoToLog(errno)
	output, err := f.StdoutPipe()
	if err != nil {
		z.Err(err).Send()
		f.Cancel()
	}
	err = sp.Start()
	if err != nil {
		z.Err(err).Send()
		sp.Cancel()
	}
	err = f.Start()
	if err != nil {
		z.Err(err).Send()
		f.Cancel()
	}
	sp.Wait()
	f.Wait()
	ctx.Done()
	return output
}
