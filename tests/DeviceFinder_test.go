package tests

import (
	"testing"
	"time"

	cast "github.com/jasonkolodziej/go-castv2"
)

func Test_FindDevices(t *testing.T) {
	devices := make(chan *cast.Device, 100)
	cast.FindDevices(time.Second*5, devices)
	for device := range devices {
		// device.PlayMedia("http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4", "video/mp4")
		// time.Sleep(time.Second * 5)
		// device.MediaController.Pause(time.Second * 5)
		// device.QuitApplication(time.Second * 5)
		status := device.GetStatus(time.Second * 5)
		t.Log(status)
		t.Log(device.Info)
	}
}
