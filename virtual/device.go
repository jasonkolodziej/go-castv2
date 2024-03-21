package virtual

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jasonkolodziej/go-castv2"
	"github.com/jasonkolodziej/go-castv2/sps"
	"github.com/reugn/go-streams/extension"
	"github.com/rs/zerolog"
)

var z = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

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
	out, errno, ss, cerr := sps.SpawnProcessWConfig(n) // * exec SPS
	if cerr != nil {
		z.Err(cerr).Send()
	}
	defer ss.Wait()
	defer errno.Close()
	go WriteStdErrnoToLog(errno) // * start collecting the logs of SPS
	// defer out.Close()
	var cErrno io.ReadCloser
	var ffmpeg *exec.Cmd
	v.content, cErrno, ffmpeg, cerr = sps.SpawnFfMpeg(out)
	if cerr != nil {
		z.Err(cerr).Msg("FFMpeg:")
	}
	defer ffmpeg.Wait()
	defer cErrno.Close()
	go WriteStdErrnoToLog(cErrno)
	return nil
	// err := v.ConnectDeviceToVirtualStream("http://192.168.2.14:3080")
	// return err
}

func WriteStdErrnoToLog(errno io.ReadCloser) {
	scanner := bufio.NewScanner(errno)
	for scanner.Scan() {
		z.Info().Msg(scanner.Text())
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

func NewDataSource(r io.ReadCloser, w io.WriteCloser) (readerSource, writerSource *extension.ChanSource) {
	var nc, nnc chan any = nil, nil
	if r != nil {
		nc = make(chan any)
		nc <- r
	}
	if w != nil {
		nnc = make(chan any)
		nnc <- w
	}
	return extension.NewChanSource(nc), extension.NewChanSource(nc)
}

func NewDataSink(r io.ReadCloser, w io.WriteCloser) (readerSink, writerSink *extension.ChanSink) {
	var nc, nnc chan any = nil, nil
	if r != nil {
		nc = make(chan any)
		nc <- r
	}
	if w != nil {
		nnc = make(chan any)
		nnc <- w
	}
	return extension.NewChanSink(nc), extension.NewChanSink(nc)
}

func PerformWhenContent(maybe io.ReadCloser, f func(io.ReadCloser, ...string) (io.ReadCloser, io.ReadCloser, error)) (io.ReadCloser, io.ReadCloser, error) {
	defer maybe.Close()
	if sps.PipePeeker(maybe) {
		z.Info().Stack().Msg("Content detected")
		return f(maybe)
	}
	return nil, nil, nil
}
