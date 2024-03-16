package virtual

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jasonkolodziej/go-castv2"
	"github.com/jasonkolodziej/go-castv2/sps"
)

type ProcBundle interface {
	Output() (output io.ReadCloser, e io.ReadCloser, err error)
	OutputWithArgs(args ...string) (output io.ReadCloser, e io.ReadCloser, err error)
	Chain(config string) (io.ReadCloser, error)
}

type VirtualDevice struct {
	*castv2.Device
	sps     *ProcBundle
	txCoder *ProcBundle // * Transcoder FfMPeg
}

func NewVirtualDevice(d *castv2.Device) *VirtualDevice {
	if d != nil {
		return &VirtualDevice{d, nil, nil}
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

func (v *VirtualDevice) FiberDeviceHandlerWithStream(reader *io.Reader) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, name := v.Info.AirplayDeviceName()
		if c.Params("deviceId") != v.Info.Id.String() {
			return c.SendString("Where is john?")
			// => Hello john
		} else if !strings.Contains(c.Path(), "stream.flac") {
			return c.SendString("Hello " + name)
		}
		s, err := v.Chain("")
		if err != nil {
			c.SendStatus(500)
		}
		defer s.Close()
		return c.SendStream(s)
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
