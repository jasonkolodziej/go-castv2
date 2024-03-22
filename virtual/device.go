package virtual

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	"-i", "pipe:0", // * input from pipe (stdout->stdin)
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

// shairport-sync -c /etc/shairport-syncKitchenSpeaker.conf -o stdout | ffmpeg -f s16le -ar 44100 -ac 2 -i pipe: -ac 2 -bits_per_raw_sample 8 -c:a pcm_s32le -y flac_test1.wav
type ProcBundle interface {
	Output() (output io.ReadCloser, e io.ReadCloser, err error)
	OutputWithArgs(args ...string) (output io.ReadCloser, e io.ReadCloser, err error)
	Chain(config string) (io.ReadCloser, error)
}

type VirtualDevice struct {
	*castv2.Device
	content        io.ReadCloser
	rawContent     io.ReadCloser
	ctx            context.Context
	Cancel         context.CancelFunc
	virtualhostAdr net.Addr
	sps, ffmpeg    *exec.Cmd
}

// * curl -s -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:5123/devices/<deviceId>/connect

func NewVirtualDevice(d *castv2.Device, ctx context.Context) *VirtualDevice {
	var v *VirtualDevice
	if d != nil {
		v = &VirtualDevice{d,
			nil, nil,
			ctx, func() { v.teardown() }, nil, nil, nil}
		// v.content <- nil
		// v.rawContent <- nil
	}
	return v
}

func (v *VirtualDevice) teardown() error {
	// defer close(v.content)
	defer v.content.Close()
	defer v.ffmpeg.Cancel()
	defer v.sps.Cancel()
	_ = v.sps.Wait()
	_ = v.ffmpeg.Wait()
	<-v.ctx.Done()
	return v.ctx.Err()
}

func (v *VirtualDevice) ZoneName() string {
	_, n := v.Device.Info.AirplayDeviceName()
	return n
}

func (v *VirtualDevice) VirtualHostAddr(netAddr net.Addr, hostname, port string) {
	if netAddr != nil {
		v.virtualhostAdr = netAddr
	} else {
		// v.virtualhostAdr;
	}
}

// Content populates VirtualDevice.content channel with a non-nil io.ReaderCloser coming from rc, a recieve-only channel
func (v *VirtualDevice) Content(rcvRc <-chan io.ReadCloser) {
	for rc := range rcvRc {
		if rc == nil {
			return
		}
		// v.content <- rc
	}
}

func (v *VirtualDevice) pathString() string {
	return "/devices/" + v.Info.Id.String() + "/stream.flac"
}

func (v *VirtualDevice) ConnectDeviceToVirtualStream() error {
	if v == nil || v.virtualhostAdr == nil { // * basic sanity check
		return fmt.Errorf("device not created")
	}
	v.QuitApplication(time.Second * 5)
	v.PlayMedia("http://"+v.virtualhostAdr.String()+v.pathString(), "audio/flac", "NONE")
	return nil
}

func (v *VirtualDevice) Virtualize() error {
	var err error
	// * Start SPS
	// n := strings.ReplaceAll(v.ZoneName(), " ", "") // * default device configuration file
	n := "/etc/shairport-syncKitchenSpeaker.conf"
	v.sps = exec.CommandContext(v.ctx, "shairport-sync", "-c", n)
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
	var err error
	if v.sps == nil {
		z.Err(err).Msg("error: VirtualDevice.sps *exec.Cmd was lost")
		return err
	}
	if v.rawContent == nil {
		z.Err(err).Msg("error: VirtualDevice.rawContent interface was nil")
		return err
	}
	ffmpeg := exec.CommandContext(
		v.ctx,
		txc,
		ffmpegArgs...,
	)
	ffmpeg.Stdin = v.rawContent
	errno, err := ffmpeg.StderrPipe()
	if err != nil {
		z.Err(err).Msg("error: ffmpeg StderrPipe()")
		ffmpeg.Cancel()
		return err
	}
	go WriteStdErrnoToLog(errno)
	output, err := ffmpeg.StdoutPipe()
	if err != nil {
		z.Err(err).Msg("error: ffmpeg StdoutPipe()")
		ffmpeg.Cancel()
		return err
	}
	v.content = output
	return ffmpeg.Start()
}

func (v *VirtualDevice) StopTranscoder() error {
	// ? https://stackoverflow.com/questions/69954944/capture-stdout-from-exec-command-line-by-line-and-also-pipe-to-os-stdout
	// rc := <-v.content        // * Get the io.ReadCloser
	defer v.content.Close()  // * defer closing
	_ = v.ffmpeg.Wait()      // * wait until full disconnect
	return v.ffmpeg.Cancel() // * call cancel then defer
}

// func (v *VirtualDevice) Virtualize() error {
// 	// * Start SPS
// 	// n := strings.ReplaceAll(v.ZoneName(), " ", "") // * default device configuration file
// 	n := "/etc/shairport-syncKitchenSpeaker.conf"
// 	out, spsErr, ffmpegErr, cerr := sps.RunPiping(n) // * exec SPS
// 	if cerr != nil {
// 		z.Err(cerr).Send()
// 	}
// 	// defer ss.Wait()
// 	defer spsErr.Close()
// 	go WriteStdErrnoToLog(spsErr) // * start collecting the logs of SPS
// 	defer ffmpegErr.Close()
// 	go WriteStdErrnoToLog(ffmpegErr)
// 	v.content <- out
// 	return nil
// 	// err := v.ConnectDeviceToVirtualStream("http://192.168.2.14:3080")
// 	// return err
// }

// WriteStdErrnoToLog will defer closing errno, io.ReadCloser
func WriteStdErrnoToLog(errno io.ReadCloser) {
	defer errno.Close()
	scanner := bufio.NewScanner(errno)
	for scanner.Scan() {
		z.Warn().Msg(scanner.Text())
	}
}

func (v *VirtualDevice) ConnectDeviceHandler() fiber.Handler {
	z.Debug().Msg("ConnectDeviceHandler")
	return func(c *fiber.Ctx) error {
		z.Debug().Msg("ConnectDeviceHandler")
		if c.Params("deviceId") != v.Info.Id.String() {
			return c.SendStatus(500)
		}
		if strings.Contains(c.Path(), "connect") {
			// TODO: ffmpeg and Chromecast application
			// err := v.StartTranscoder()
			// if err != nil {
			// 	z.Err(err).Msg("error: StartTranscoder()")
			// 	//data, _ := json.MarshalIndent(err, "", "  ")
			// 	c.JSON(err)
			// 	return c.SendStatus(500)
			// }
			v.ConnectDeviceToVirtualStream()
			return c.SendString("connecting")
		}
		return c.Next()
	}
}

func (v *VirtualDevice) DisconnectDeviceHandler() fiber.Handler {
	z.Debug().Msg("DisonnectDeviceHandler")
	return func(c *fiber.Ctx) error {
		if c.Params("deviceId") != v.Info.Id.String() {
			return c.SendString("Where is john?")
			// => Hello john
		}
		_, name := v.Info.AirplayDeviceName()
		if strings.Contains(c.Path(), "disconnect") {
			z.Info().Msg("disconnect invoked: calling QuitApplication")
			v.QuitApplication(time.Second * 5)
			z.Info().Msg("disconnect invoked: calling ffmpeg pipe to be killed")
			err := v.StopTranscoder()
			if err != nil {
				z.Err(err).Msg("disconnect invoked: StopTranscoder()")
				return c.SendStatus(500)
			}
			// TODO: Replace with OK
			return c.SendString(fmt.Sprintf("Device: %s, Path: %s, Hostname: %s", name, c.Path(), c.Hostname()))
		}
		return c.Next()

	}
}

func (v *VirtualDevice) SpawnCmdWithContext(ctx context.Context, configPath string) error {
	var err error
	if configPath == "" {
		configPath = "/etc/shairport-syncKitchenSpeaker.conf"
	}
	sp := exec.CommandContext(ctx, "shairport-sync", "-c", configPath)
	spOut, err := sp.StdoutPipe() // * assign ffmpeg stdin to sps stdout
	if err != nil {
		z.Err(err).Msg("error: shairport-sync StdoutPipe()")
		sp.Cancel()
		return err
	}
	errno, err := sp.StderrPipe()
	if err != nil {
		z.Err(err).Msg("error: shairport-sync StderrPipe()")
		sp.Cancel()
		return err
	}
	out := make(chan io.ReadCloser)
	go MonitorOutput(spOut, out)
	go WriteStdErrnoToLog(errno)
	f := exec.CommandContext(
		ctx,
		txc,
		ffmpegArgs...,
	)
	errno, err = f.StderrPipe()
	if err != nil {
		z.Err(err).Msg("error: ffmpeg StderrPipe()")
		sp.Cancel()
		f.Cancel()
		return err
	}
	go WriteStdErrnoToLog(errno)
	output, err := f.StdoutPipe()
	if err != nil {
		z.Err(err).Msg("error: ffmpeg StdoutPipe()")
		sp.Cancel()
		f.Cancel()
		return err
	}
	// ch := make(chan io.ReadCloser) // * bi-directional channel
	// ch <- output                   // * send output through the bi-directional channel
	// v.Content(ch)                  // * pass ch as a recieve-only channel
	v.content = output
	err = sp.Start()
	if err != nil {
		z.Err(err).Msg("error: ffmpeg Start()")
		sp.Cancel()
		f.Cancel()
		return err
	}
	err = f.Start()
	if err != nil {
		z.Err(err).Msg("error: ffmpeg Start()")
		sp.Cancel()
		f.Cancel()
		return err
	}
	sp.Wait()
	f.Wait()
	<-ctx.Done()
	return nil
}

// MonitorOutput sends an io.ReadCloser through out, send-only channel
func MonitorOutput(content io.ReadCloser, out chan<- io.ReadCloser) {
	defer close(out)
	scanner := bufio.NewScanner(content)
	scanner.Split(bufio.ScanBytes)
	// io.TeeReader(content, bufio.NewWriter(nil))
	for scanner.Scan() {
		out <- content
	}
	out <- nil
}

func MonitorInput(in <-chan io.ReadCloser, stdIn io.Reader) {
	for ch := range in {
		if ch == nil {
			return
		}
		stdIn = ch
	}
}

// func Stream(ctx context.Context, out chan<- Value) error {
//     for {
//         v, err := DoSomething(ctx)
//         if err != nil {
//             return err
//         }
//         select {
//         case <-ctx.Done():
//             return ctx.Err()
//         case out <- v:
//         }
//     }
// }

func (v *VirtualDevice) FiberDeviceHandlerWithStream() fiber.Handler {
	// defer v.content.Close()
	return func(c *fiber.Ctx) error {
		z.Debug().Msg("FiberDeviceHandlerWithStream")
		z.Debug().Msg(c.Path())
		//_, name := v.Info.AirplayDeviceName() // * Get chromecast Id
		if c.Params("deviceId") != v.Info.Id.String() {
			return c.SendString("Where is jason?" + " " + c.Params("deviceId"))
			// => Hello john
		} else if !strings.Contains(c.Path(), "stream.flac") { // * does the path not contain `stream.flac`
			return c.Next()
		}
		content := v.content
		if content == nil {
			z.Error().Msg("content nil")
			return c.SendStatus(501)
		}
		c.Set("Transfer-Encoding", "chunked")
		c.Context().SetContentType("audio/mpeg; codecs=\"flac\"")
		// s, err := v.Chain("") //! do someting else
		// if err != nil {
		// 	c.SendStatus(500)
		// }
		// defer s.Close()
		return c.SendStream(content)
	}
}

func (v *VirtualDevice) StreamVideo(ctx *fiber.Ctx) error {
	z.Debug().Msg("StreamAudio2")
	if ctx.Params("deviceId") != v.Info.Id.String() {
		return ctx.SendStatus(504)
		// => Hello john
	} else if !strings.Contains(ctx.Path(), "stream") { // * does the path not contain `stream.flac`
		return ctx.Next()
	}
	pwd, _ := os.Getwd()
	z.Debug().Msgf("StreamAudio2: %s", pwd)
	filePath := pwd + "/data/flac_test1.wav"
	// file := "video.mp4"
	return ctx.SendFile(filePath, true)
}

func (v *VirtualDevice) StreamAudio(ctx *fiber.Ctx) error {
	z.Debug().Msg("StreamAudio")
	if ctx.Params("deviceId") != v.Info.Id.String() {
		return ctx.SendStatus(504)
		// => Hello john
	} else if !strings.Contains(ctx.Path(), "stream") { // * does the path not contain `stream.flac`
		return ctx.Next()
	}
	pwd, _ := os.Getwd()
	z.Debug().Msgf("StreamAudio: %s", pwd)
	filePath := pwd + "/data/flac_test1.wav"

	// Open the video file
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening audio file:", err)
		return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}
	defer file.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("Error getting file information:", err)
		return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	// get the file mime informations
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	z.Debug().Msgf("MIMEType: %s", mimeType)

	// get file size
	fileSize := fileInfo.Size()

	// Get the range header from the request
	rangeHeader := ctx.GetRespHeader("range") // * ctx.GetReqHeaders()["range"]
	if rangeHeader != "" {
		var start, end int64

		ranges := strings.Split(rangeHeader, "=")
		if len(ranges) != 2 {
			log.Println("Invalid Range Header:", err)
			return ctx.Status(http.StatusInternalServerError).SendString("Internal Server Error")
		}

		byteRange := ranges[1]
		byteRanges := strings.Split(byteRange, "-")

		// get the start range
		start, err := strconv.ParseInt(byteRanges[0], 10, 64)
		if err != nil {
			log.Println("Error parsing start byte position:", err)
			return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		// Calculate the end range
		if len(byteRanges) > 1 && byteRanges[1] != "" {
			end, err = strconv.ParseInt(byteRanges[1], 10, 64)
			if err != nil {
				log.Println("Error parsing end byte position:", err)
				return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
			}
		} else {
			end = fileSize - 1
		}

		// Setting required response headers
		ctx.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size())) // Set the Content-Range header
		ctx.Set(fiber.HeaderContentLength, strconv.FormatInt(end-start+1, 10))                        // Set the Content-Length header for the range being served
		ctx.Set(fiber.HeaderContentType, mimeType)                                                    // Set the Content-Type
		ctx.Set(fiber.HeaderAcceptRanges, "bytes")                                                    // Set Accept-Ranges
		ctx.Status(fiber.StatusPartialContent)                                                        // Set the status code to 206 (Partial Content)

		// Seek to the start position
		_, seekErr := file.Seek(start, io.SeekStart)
		if seekErr != nil {
			log.Println("Error seeking to start position:", seekErr)
			return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		// Copy the specified range of bytes to the response
		_, copyErr := io.CopyN(ctx.Response().BodyWriter(), file, end-start+1)
		if copyErr != nil {
			log.Println("Error copying bytes to response:", copyErr)
			return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

	} else {
		// If no Range header is present, serve the entire video
		ctx.Set("Content-Length", strconv.FormatInt(fileSize, 10))
		_, copyErr := io.Copy(ctx.Response().BodyWriter(), file)
		if copyErr != nil {
			log.Println("Error copying entire file to response:", copyErr)
			return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}
	}

	return nil

}

func (v *VirtualDevice) Output() (output io.ReadCloser, e io.ReadCloser, err error) {
	return nil, nil, nil
}

func (v *VirtualDevice) OutputWithArgs(configPath ...string) (output io.ReadCloser, e io.ReadCloser, err error) {
	var confFlag = append([]string{"-c"}, configPath...)
	return sps.SpawnProcessConfig(confFlag...)
}

func PrintStack(app *fiber.App) {
	data, _ := json.MarshalIndent(app.Stack(), "", "  ")
	fmt.Println(string(data))
	// z.Info().Msg(string(data))
}
