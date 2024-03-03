package main

import (
	"time"

	"github.com/jasonkolodziej/go-castv2"
)

// A simple example, showing how to create a device and use it.
func main() {
	deviceCh := make(chan *castv2.Device, 100)
	castv2.FindDevices(time.Second*30, deviceCh)
	for device := range deviceCh {
		device.PlayMedia(
			"http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/images/BigBuckBunny.jpg",
			"image/jpeg")
	}

}
