package virtual

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

func (v *VirtualDevice) pathString() string {
	return "/devices/" + v.Info.Id.String() + "/stream"
}

func PrintStack(app *fiber.App) {
	data, _ := json.MarshalIndent(app.Stack(), "", "  ")
	fmt.Println(string(data))
	// z.Info().Msg(string(data))
}

func (v *VirtualDevice) ConnectDeviceHandler() fiber.Handler {
	z.Debug().Msg("ConnectDeviceHandler")
	return func(c *fiber.Ctx) error {
		z.Debug().Msg("ConnectDeviceHandler")
		if c.Params("deviceId") == v.Info.Id.String() &&
			strings.Contains(c.Path(), "connect") {
			// TODO: ffmpeg and Chromecast application
			err := v.StartTranscoder()
			if err != nil {
				z.Err(err).Msg("error: StartTranscoder()")
				//data, _ := json.MarshalIndent(err, "", "  ")
				return c.SendStatus(500)
			}
			v.ConnectDeviceToVirtualStream()
			return c.SendString("connecting")
		}
		return c.Next()
	}
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
	z.Debug().Msg("DisonnectDeviceHandler")
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

func (v *VirtualDevice) HandleStream() fiber.Handler {
	go GetStreamFromReader(v.connectionPool, v.content)
	z.Info().Msg("virtual.HandleStream has started")
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
