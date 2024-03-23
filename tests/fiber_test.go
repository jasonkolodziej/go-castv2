package tests

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
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

func Test_PlayWav(t *testing.T) {
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
	localIp := net.ParseIP("192.168.2.14:5123")
	K := virtual.NewVirtualDevice(&kitchen, context.Background())
	K.VirtualHostAddr(&net.IPAddr{IP: localIp, Zone: ""}, "", "")
	// K.QuitApplication(time.Second * 20)
	K.PlayMedia("https://www2.cs.uic.edu/~i101/SoundFiles/PinkPanther30.wav", "audio/wav", "LIVE")
	t.Log("done")

}
func TestMain(t *testing.T) {
	// * Fiber router setup
	// if err = K.SpawnCmdWithContext(context.Background(), ""); err != nil {
	// 	t.Fatal(err)
	// }

	var fib = fiber.New(fiber.Config{
		Prefork:       true,
		CaseSensitive: true,
		StrictRouting: true,
		ServerHeader:  "Fiber",
		AppName:       "Test App v1.0.1",
		GETOnly:       true,
		Views:         html.New("./templates", ".tpl"),
	})

	// var middleware = func(c *fiber.Ctx) error {
	// 	// Set a custom header on all responses:
	// 	c.Set("Access-Control-Allow-Origin", "*")
	// 	c.Set("Transfer-Encoding", "chunked")
	// 	c.Set("X-Custom-Header", "Hello, World")
	// 	// Go to next middleware:
	// 	return c.Next()
	// }
	// fib.Get("/", func(c *fiber.Ctx) error {
	// 	return c.Render("index", nil)
	// })
	// Define a route for streaming video

	// func(ctx *fiber.Ctx) error {
	// 	pwd, _ := os.Getwd()
	// 	z.Debug().Msgf("StreamAudio: %s", pwd)
	// 	filePath := pwd + "/data/flac_test1.flac"
	// 	// file := "video.mp4"
	// 	return ctx.SendFile(filePath, true)
	// }
	fib.Get("/stream", streamVideo)
	// deviceHandlers := []fiber.Handler{K.StreamAudio}

	// mdev := append([]fiber.Handler{middleware}, deviceHandlers...)
	// device := fib.Group("/devices/:deviceId", mdev...)
	// device.Get("/disconnect", K.DisconnectDeviceHandler())
	// device.Get("/connect", K.ConnectDeviceHandler())
	// device.Get("/stream", K.StreamAudio)
	// virtual.PrintStack(fib)
	// err = K.Virtualize()
	// if err != nil {

	// 	t.Fatal(err)
	// }
	fib.Listen(":5123")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// ! Working
	// ip := net.ParseIP("192.168.2.152")
	// mac, err := net.ParseMAC("f4:f5:d8:be:cd:ec")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// var findKitchen = cast.FromServiceEntryInfo(nil, nil, &mac)
	// findKitchen.Fn = "Kitchen speaker"
	// findKitchen.Id = uuid.MustParse("a548ff5a-d1fa-c194-1101-acb5a1204788")
	// findKitchen.IpAddress = &ip
	// findKitchen.SetPort(cast.CHROMECAST)
	// kitchen, err := cast.NewDeviceFromDeviceInfo(findKitchen)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// localIp := net.ParseIP("192.168.2.14:5123")
	// K := virtual.NewVirtualDevice(&kitchen, context.Background())
	// K.VirtualHostAddr(&net.IPAddr{IP: localIp, Zone: ""}, "", "")
	// K.PlayMedia("http://192.168.2.14:5123/stream", "audio/flac", "BUFFERED")
	// t.Log("done")

	// t.Cleanup(K.Cancel)
}

func streamVideo(ctx *fiber.Ctx) error {
	pwd, _ := os.Getwd()
	z.Debug().Msgf("StreamAudio: %s", pwd)
	filePath := pwd + "/data/flac_test1.flac"

	// Open the video file
	file, err := os.Open(filePath)
	if err != nil {
		z.Err(err).Msg("Error opening audio file:")
		return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}
	defer file.Close()
	// open the pipe
	// reader, writer := io.Pipe()
	// if err != nil {
	// 	z.Err(err).Msg("Error opening audio file:")
	// 	return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	// }
	// defer reader.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		z.Err(err).Msg("Error getting file information:")
		return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
	}

	// * get the file mime informations
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	z.Debug().Msgf("MIMEType: %s", mimeType)
	// ? Does the requestor have these

	secFetchDest := ctx.Get("Sec-Fetch-Dest") // ? Sec-Fetch-Dest: video || 1 - document
	secFetchMode := ctx.Get("Sec-Fetch-Mode") // ? Sec-Fetch-Mode: no-cors || 1 - navigate
	secFetchSite := ctx.Get("Sec-Fetch-Site") // ? Sec-Fetch-Site: same-origin || 1 - none
	z.Debug().Msgf("Request: Sec-Fetch-Dest: %s, Sec-Fetch-Mode:%s, Sec-Fetch-Site:%s", secFetchDest, secFetchMode, secFetchSite)
	// * get file size
	fileSize := fileInfo.Size()
	// * Get the header from the request
	rangeHeader := ctx.Get(fiber.HeaderRange)   // * Get Range:
	keepAlive := ctx.Get(fiber.HeaderKeepAlive) // * Get Keep-Alive:
	ctx.Set(fiber.HeaderAcceptRanges, "bytes")  // * Set Accept-Ranges:
	ctx.Set(fiber.HeaderContentType, mimeType)  // * Set Content-Type:
	// ctx.Set(fiber.HeaderAccessControlAllowOrigin, "*")     // * "Access-Control-Allow-Origin"
	// ctx.Set(fiber.HeaderTransferEncoding, "chunked")       // * Set Transfer-Encoding:
	ctx.Set(fiber.HeaderConnection, fiber.HeaderKeepAlive) // * Set Connection: Keep-Alive
	// * Add or adjust Keep-Alive: timeout=X, max=Xx
	var timeout int64 = 5
	var max int64 = 100
	if keepAlive != "" {
		z.Debug().Any("requestKeepAlive", keepAlive).Msgf("Request: contains a Keep-Alive header")
		vals := strings.Split(keepAlive, ",")
		if len(vals) != 2 {
			z.Error().Msgf("Invalid Keep-Alive Header: %s", keepAlive)
			return ctx.Status(http.StatusInternalServerError).SendString("Internal Server Error")
		}
		timeout, err = strconv.ParseInt(strings.TrimPrefix(vals[0], "timeout="), 10, 64)
		if err != nil {
			z.Err(err).Msgf("Parser Error Keep-Alive Header: %s", vals[0])
			return ctx.Status(http.StatusInternalServerError).SendString("Internal Server Error")
		}
		max, err = strconv.ParseInt(strings.TrimPrefix(vals[1], "max="), 10, 64)
		if err != nil {
			z.Err(err).Msgf("Parser Error Keep-Alive Header: %s", vals[1])
			return ctx.Status(http.StatusInternalServerError).SendString("Internal Server Error")
		}
	} else {
		z.Debug().Msgf("Request: DOES NOT have Keep-Alive header, Setting DEFAULTS")
		// * handle default Keep-Alive
		keepAlive = fmt.Sprintf("timeout=%d, max=%d", timeout, max)
		ctx.Set(fiber.HeaderKeepAlive, keepAlive)
	}
	// * Handle Range Header in Request
	if rangeHeader != "" {
		z.Debug().Any("requestRangeHeader", rangeHeader).Msgf("Request: DOES have Range header, handling...")
		var start, end int64

		ranges := strings.Split(rangeHeader, "=")
		if len(ranges) != 2 {
			z.Err(err).Msg("Invalid Range Header:")
			return ctx.Status(http.StatusInternalServerError).SendString("Internal Server Error")
		}

		byteRange := ranges[1]
		byteRanges := strings.Split(byteRange, "-")

		// * get the start range
		start, err := strconv.ParseInt(byteRanges[0], 10, 64)
		if err != nil {
			z.Err(err).Msg("Error parsing start byte position:")
			return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		// * Calculate the end range
		if len(byteRanges) > 1 && byteRanges[1] != "" {
			end, err = strconv.ParseInt(byteRanges[1], 10, 64)
			if err != nil {
				z.Err(err).Msg("Error parsing end byte position:")
				return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
			}
		} else {
			end = fileSize - 1
		}

		// Setting required response headers
		if end != 1 {
			max -= 1
			keepAlive = fmt.Sprintf("timeout=%d, max=%d", timeout, max)
			ctx.Set(fiber.HeaderKeepAlive, keepAlive) // * Set Keep-Alive header
		}
		ctx.Set(fiber.HeaderContentLength, strconv.FormatInt(end-start+1, 10)) // * Set the Content-Length header for the range being served
		// ctx.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/*", start, end)) // * Set the Content-Range header
		ctx.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size())) // * Set the Content-Range header
		ctx.Status(fiber.StatusPartialContent)                                                        // * Set the status code to 206 (Partial Content)
		// Seek to the start position
		_, seekErr := file.Seek(start, io.SeekStart)
		if seekErr != nil {
			z.Err(seekErr).Msg("Error seeking to start position:")
			return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}

		// Copy the specified range of bytes to the response
		_, copyErr := io.CopyN(ctx.Response().BodyWriter(), file, end-start+1)
		if copyErr != nil {
			z.Err(copyErr).Msg("Error copying bytes to response:")
			return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}
		z.Debug().Msgf("Header: Content-Length:=%v, Content-Range: %s, Seeking file to position: start: %d, copying range to response: %d",
			strconv.FormatInt(end-start+1, 10),
			fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size()),
			start,
			end-start+1)
	} else {
		z.Debug().Msg("Request DOES NOT contain Range Header, sending whole file")
		// If no Range header is present, serve the entire video
		ctx.Set("Content-Length", strconv.FormatInt(fileSize, 10)) // * Set the Content-Length header for the range being served
		_, copyErr := io.Copy(ctx.Response().BodyWriter(), file)
		if copyErr != nil {
			z.Err(copyErr).Msg("Error copying entire file to response:")
			return ctx.Status(fiber.StatusInternalServerError).SendString("Internal Server Error")
		}
		return ctx.SendStatus(fiber.StatusOK)
	}

	return nil

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
