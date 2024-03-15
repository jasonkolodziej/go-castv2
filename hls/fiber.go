package hls

import "github.com/gofiber/fiber/v2"

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

func NewFiberServer() {
	// Create Group
	devices := fib.Group("/devices/*", middleware)
	device := devices.Get("/:deviceId?", func(c *fiber.Ctx) error {
		if c.Params("name") != "" {
			return c.SendString("Hello " + c.Params("name"))
		}
		return c.SendString("Where is john?")
	})
	device.Get("/stream.flac", func(c *fiber.Ctx) error {
		return nil
	})
}
