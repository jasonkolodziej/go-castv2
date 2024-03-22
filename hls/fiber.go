package hls

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

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

func NewFiberServer(deviceHandlers ...fiber.Handler) *fiber.App {
	// Create Group
	mdev := append([]fiber.Handler{middleware}, deviceHandlers...)
	devices := fib.Group("/devices/:deviceId", mdev...)
	devices.Get("/stream.flac", deviceHandlers...)

	data, _ := json.MarshalIndent(fib.Stack(), "", "  ")
	fmt.Println(string(data))

	return fib
}
