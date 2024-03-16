package tests

import (
	"net"
	"testing"
	"time"

	cast "github.com/jasonkolodziej/go-castv2"
)

func Test_MACResolver(t *testing.T) {
	names, err := net.LookupAddr("192.168.2.152")
	// net.Interface
	net.LookupHost(names[0])
	// net.InterfaceByName()
	ipaddr, err := net.ResolveIPAddr("ip", "192.168.2.152")
	// var dialer = net.DefaultResolver.LookupNetIP()
	if err != nil {
		t.Error(err)
	}
	for _, name := range names {
		t.Log(name)
	}
	t.Log(ipaddr.Network())
}

func Test_FindDevices(t *testing.T) {
	devices := make(chan *cast.Device, 100)
	cast.FindDevices(time.Second*5, devices)
	var i = 0
	for device := range devices {
		// device.PlayMedia("http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4", "video/mp4")
		// time.Sleep(time.Second * 5)
		// device.MediaController.Pause(time.Second * 5)
		// device.QuitApplication(time.Second * 5)
		// status := device.GetStatus(time.Second * 5)
		// t.Log(status)
		_, name := device.Info.AirplayDeviceName()
		t.Logf("UUID: %s", device.Info.Id.String())
		if device.Info.IsGroup() {
			t.Logf("Found Group named ==> %s, with address %v", name, device.Info.IPAddress())
			t.Log(device.Info)
			i++
		}
		if i > 1 {
			t.Errorf("This test expects there to be only 1 group of Devices.. Cleaning up.")
		}
		t.Logf("Device: %s, with address %s, mac: %v", name, device.Info.IPAddress(), device.Info.MAC())
		t.Log(device.Info)
	}
	if i == 0 {
		t.Errorf("This test expects there to be 1 group of Devices..")
	}
	t.Logf("Number of group(s) found: %v", i)
}

func Test_FindSpecific(t *testing.T) {
	ip := net.ParseIP("192.168.2.152")
	var findKitchen = cast.DeviceInfo{
		Fn:        "Kitchen speaker",
		IpAddress: &ip,
	}
	found, err := cast.FindDevice(&findKitchen)

	if err != nil {
		t.Fatal()
	}
	t.Log(found.Info.IpAddress, found.Info.Fn, found.Info.MAC())

	//? Get status
	status := found.GetStatus(time.Second * 5)
	printJson(t, status)

	//? Get media Status
	mStatus := found.GetMediaStatus(time.Second * 5)
	if len(mStatus) == 0 {
		t.Log("Skipping GetMediaStatus")
	} else {
		for _, stat := range mStatus {
			printJson(t, stat)
		}
	}
	//? load and play remote file
	// found.QuitApplication()
	// found.ReceiverController.SetVolume()
	found.PlayMedia(remoteSoundFile, "audio/mp3")
	t.Log("done")
	found.QuitApplication(time.Second * 5)
	// if err != nil {
	// 	t.Error(err)
	// }
	// t.Log(msg)
}
