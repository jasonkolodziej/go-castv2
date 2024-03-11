package tests

import (
	"testing"
	"time"

	cast "github.com/jasonkolodziej/go-castv2"
)

// func kitchenSpeaker() cast.DeviceInfo {
// 	ip, _, _ := net.ParseCIDR("192.168.2.152")
// 	mac, _ := net.ParseMAC("FA8FCA8766F6")
// "f4:f5:d8:be:cd:ec"
// 	return cast.DeviceInfo{IpAddress: &ip, Bs: &mac}
// }

// func Test_MACResolver(t *testing.T) {
// 	names, err := net.LookupAddr("192.168.2.152")
// 	net.LookupHost(names[0])
// 	// net.InterfaceByName()
// 	ipaddr, err := net.ResolveIPAddr("ip", "192.168.2.152")
// 	// var dialer = net.DefaultResolver.LookupNetIP()
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	for _, name := range names {
// 		t.Log(name)
// 	}
// 	t.Log(ipaddr.Network())
// }

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
		if device.Info.IsGroup() {
			t.Logf("Found Group named ==> %s, with address %v", name, device.Info.IPAddress())
			t.Log(device.Info)
			i++
		}
		if i > 1 {
			t.Errorf("This test expects there to be only 1 group of Devices.. Cleaning up.")
		}
		t.Logf("Device: %s, with address %s, mac: %s", name, device.Info.IPAddress(), device.Info.MAC.String())
		t.Log(device.Info)
	}
	if i == 0 {
		t.Errorf("This test expects there to be 1 group of Devices..")
	}
	t.Logf("Number of group(s) found: %v", i)
}
