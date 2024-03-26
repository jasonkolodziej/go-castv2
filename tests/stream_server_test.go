package tests

import (
	"bufio"
	"context"
	"net"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	cast "github.com/jasonkolodziej/go-castv2"
	"github.com/jasonkolodziej/go-castv2/virtual"
)

var fib = fiber.New(fiber.Config{
	Prefork:       true,
	CaseSensitive: true,
	StrictRouting: true,
	ServerHeader:  "Fiber",
	AppName:       "Test App v1.0.1",
	// GETOnly:       true,
})

func Test_SendStream(t *testing.T) {
	ctn, _ := loadTestFile(t, "output.aac", false)
	defer ctn.Close()
	connPool := virtual.NewConnectionPool()

	go virtual.GetStreamFromReader(connPool, ctn)
	fib.Get("/", func(c *fiber.Ctx) error {
		// z.Info().Any("CtxId", c.Context().ID()).Send()
		// z.Info().Any("headers", c.Context().Request.String()).Send()
		c.Context().SetContentType("audio/aac")
		c.Set(fiber.HeaderConnection, fiber.HeaderKeepAlive)
		var connection = connPool.HasConnectionWithId(c.Context().RemoteIP())
		// connection, ok := c.Context().Value("connection").(*virtual.Connection)
		if connection == nil {
			// z.Warn().Msg("Assembling a new connection!")
			connection = virtual.NewConnectionWithId(c.Context().RemoteIP())
			connPool.AddConnection(connection)
		} else {
			z.Warn().Msg("Found a existing connection")
		}
		z.Info().Msgf("%s has connected to the audio stream\n", c.Context().RemoteIP().String())
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			for {
				buf := <-connection.BufferCh()
				if _, err := w.Write(buf); err != nil {
					connPool.DeleteConnection(connection)
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
	})

	fib.Listen(":8080")
}

func Test_VirtualDeviceHandlers(t *testing.T) {
	// * Fiber router setup
	devices := fib.Group("/devices")

	// ! Working
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

	err = K.Virtualize()
	if err != nil {
		t.Fatal(err)
	}
	devices.Get("/:deviceId", K.Handlers()...)
	K.VirtualHostAddr(&net.IPAddr{IP: localIp, Zone: ""}, "", "")
	K.PlayMedia("http://192.168.2.14:5123/stream", "audio/flac", "BUFFERED")
	z.Info().Msg("startingserver has started")
	err = fib.Listen(":5123")
	if err != nil {
		t.Fatal(err)
	}
	// t.Log("done")

	// t.Cleanup(K.Cancel)
}
