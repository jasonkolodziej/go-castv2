package virtual

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jasonkolodziej/go-castv2"
	"github.com/jasonkolodziej/go-castv2/sps"
	"github.com/reugn/go-streams/extension"
	"github.com/reugn/go-streams/flow"
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
	content *io.ReadCloser
	sps     *ProcBundle
	txCoder *ProcBundle // * Transcoder FfMPeg
}

func NewVirtualDevice(d *castv2.Device) *VirtualDevice {
	if d != nil {
		return &VirtualDevice{d, nil, nil, nil}
	}
	return nil
}

func (v *VirtualDevice) pathString() string {
	return "/devices/" + v.Info.Id.String() + "/stream.flac"
}

func (v *VirtualDevice) connectDeviceToVirtualStream(urlPath string) error {
	if v == nil { // * basic sanity check
		return fmt.Errorf("Device not created")
	}
	v.QuitApplication(time.Second * 5)
	v.PlayMedia(urlPath, "audio/flac", "LIVE")
	// if err != nil {
	// 	return err
	// }
	return nil
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
		return c.SendStream(*v.content)
	}
}

func (v *VirtualDevice) Output() (output io.ReadCloser, e io.ReadCloser, err error) {
	return nil, nil, nil
}

func (v *VirtualDevice) OutputWithArgs(configPath ...string) (output io.ReadCloser, e io.ReadCloser, err error) {
	var confFlag = append([]string{"-c"}, configPath...)
	return sps.SpawnProcessConfig(confFlag...)
}

func (v *VirtualDevice) Chain(config string) (io.ReadCloser, error) {
	encoded, spsErr, txcErr, cErr := sps.RunPiping(config)
	if cErr != nil {
		return nil, cErr
	}
	defer txcErr.Close()
	defer spsErr.Close()
	return encoded, nil
}

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

func (v *VirtualDevice) Streams(config string) (encoded io.ReadCloser, spsErr io.ReadCloser, txcErr io.ReadCloser, cErr error) {
	out, spsErr, cErr := sps.SpawnProcessWConfig(config)
	if cErr != nil {
		z.Err(cErr).Msg("shairport-sync Wait():")
		return nil, nil, nil, cErr
	}
	peeker := bufio.NewReader(out)
	peeked, err := peeker.Peek(1)
	if len(peeked) == 1 && err != nil { // * Check to see if there is audio

	}
	dsrc, _ := NewDataSource(out, nil)
	f := flow.NewFilter[io.ReadCloser](
		func(b io.ReadCloser) bool {
			peeker := bufio.NewReader(out)
			peeked, err := peeker.Peek(1)
			if len(peeked) == 1 && err != nil { // * Check to see if there is audio
				return true
			}
			return false
		}, 1)
	// pt := flow.NewPassThrough()
	good := dsrc.Via(f)
	o := make(<-chan io.ReadCloser) // * Receive only channel
	// defer close(o)
	good.In() <- o
	encoded, txcErr, cErr = sps.SpawnFfMpegWith(o)
	if cErr != nil {
		z.Err(cErr).Msg("FFMpeg Wait():")
		return nil, nil, nil, cErr
	}
	return encoded, spsErr, txcErr, nil

}
