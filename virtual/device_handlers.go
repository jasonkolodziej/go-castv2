package virtual

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func (v *VirtualDevice) pathString() string {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	return "/devices/" + v.Info.Id.String() + "/stream"
}

func PrintStack(app *fiber.App) {
	data, _ := json.MarshalIndent(app.Stack(), "", "  ")
	fmt.Println(string(data))
	// z.Info().Msg(string(data))
}
func (v *VirtualDevice) Handlers() []fiber.Handler {
	return []fiber.Handler{
		v.DefaultHandler(),
		v.ConnectDeviceHandler(),
		v.DisconnectDeviceHandler(),
		v.HandleStream(),
		v.PauseDeviceHandler(),
		v.VolumeHandler(),
	}
}

func (v *VirtualDevice) Router(api fiber.Router) {
	r := api.Get("/:deviceId", v.DefaultHandler())
	r.Get("/connect", v.ConnectDeviceHandler())
	r.Get("/disconnect", v.DisconnectDeviceHandler())
	r.Get("/stream", v.HandleStream())
	r.Get("/pause", v.PauseDeviceHandler())
	r.Post("/volume", v.VolumeHandler())
}

func (v *VirtualDevice) DefaultHandler() fiber.Handler {
	if !fiber.IsChild() {
		z.Debug().Msg("DefaultHandler: parent")
	} else {
		z.Debug().Msgf("DefaultHandler: child pid: %d", os.Getpid())
	}
	return func(c *fiber.Ctx) error {
		if c.Params("deviceId") != v.Info.Id.String() {
			return c.Next()
		}
		switch c.Params("*") {
		case "", "/":
			z.Debug().Msg("case empty")
			return c.SendString("hello " + v.Info.Fn)
		default:
			return c.Next()
		}
	}
}

func (v *VirtualDevice) ConnectDeviceHandler() fiber.Handler {
	z.Debug().Msg("ConnectDeviceHandler")
	return func(c *fiber.Ctx) error {
		if c.Params("deviceId") == v.Info.Id.String() &&
			strings.Contains(c.Path(), "connect") {
			z.Info().Msgf("%s has called connect", c.Context().RemoteIP())
			// TODO: ffmpeg and Chromecast application
			if err := v.StartTranscoder(); err != nil {
				z.Error().AnErr("ConnectDeviceHandler", err).Msg("StartTranscoder")
			}
			if v.content == nil {
				z.Debug().AnErr("ConnectDeviceHandler", fmt.Errorf("content deemed of nil Type")).Msg("error: StartTranscoder()")
				//data, _ := json.MarshalIndent(err, "", "  ")
				return c.SendStatus(500)
			}
			if v.connectionPool.Empty() { // * if this is the first time /connect was called
				v.openFileAndStream()
			}
			// v.ConnectDeviceToVirtualStream() // * Inform the google chromecast to play
			return c.SendString("connecting... /stream should be avail.")
		}
		return c.Next()
	}
}

func (v *VirtualDevice) openFileAndStream() {
	var err error
	v.content, err = os.Open(v.fileName)
	if err != nil {
		z.Err(err).Msg("opening temp file")
	}
	// defer v.content.Close()
	z.Debug().Msgf("opened file: %s", v.fileName)
	go GetStreamFromReader(v.connectionPool, v.content)
}

func (v *VirtualDevice) DisconnectDeviceHandler() fiber.Handler {
	z.Debug().Msg("DisconnectDeviceHandler")
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

func (v *VirtualDevice) PauseDeviceHandler() fiber.Handler {
	z.Debug().Msg("PauseDeviceHandler")
	return func(c *fiber.Ctx) error {
		if c.Params("deviceId") == v.Info.Id.String() &&
			strings.Contains(c.Path(), "pause") {
			z.Info().Msg("pause invoked: calling Pause")
			v.Pause()
			return c.SendStatus(200)
		}
		return c.Next()

	}
}

func (v *VirtualDevice) VolumeHandler() fiber.Handler {
	z.Debug().Msg("VolumeHandler")
	return func(c *fiber.Ctx) error {
		if c.Params("deviceId") == v.Info.Id.String() &&
			strings.Contains(c.Path(), "volume") {
			// TODO: Handle changing volume of device
			switch c.Method() {
			case fiber.MethodGet:
				return c.JSON(v.GetVolume(time.Second * 5))
			case fiber.MethodPost:
			case fiber.MethodPatch:
			case fiber.MethodPut:
				z.Info().Any("providedBody", string(c.Request().Body()))
				return c.SendStatus(200)
			}
			return c.SendStatus(200)
		}
		return c.Next()
	}
}

func (v *VirtualDevice) HandleStream() fiber.Handler {
	z.Info().Msg("HandleStream")
	//// go GetStreamFromReader(v.connectionPool, v.content)
	return func(ctx *fiber.Ctx) error {
		if ctx.Params("deviceId") != v.Info.Id.String() && // * does the path not contain v.Info.Id
			!strings.Contains(ctx.Path(), "stream") { // * does the path not contain `stream`
			return ctx.Next()
		}
		ctx.Response().Header.Add(fiber.HeaderContentType, *v.contentType)
		ctx.Response().Header.Add(fiber.HeaderConnection, fiber.HeaderKeepAlive)
		var connection = v.connectionPool.HasConnectionWithId(ctx.Context().RemoteIP())
		if connection == nil {
			z.Warn().Msg("Assembling a new connection!")
			// connection = NewConnection()
			connection = NewConnectionWithId(ctx.Context().RemoteIP())
			v.connectionPool.AddConnection(connection)
		} else {
			z.Warn().Msg("Found a existing connection")
		}
		z.Info().Msgf("%s has connected to the audio stream\n", ctx.Context().RemoteIP())
		ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			for {
				buf := <-connection.BufferCh()
				if _, err := w.Write(buf); err != nil {
					v.connectionPool.DeleteConnection(connection)
					z.Info().Err(err).Msgf("connection to the audio stream has been closed\n")
					return
				}
				if err := w.Flush(); err != nil {
					z.Warn().Err(err).Msg("calling writer.Flush")
				}
				connection.ClearBuffer()
			}
		})
		return nil
	}
}
