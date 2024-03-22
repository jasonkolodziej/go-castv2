package castv2

import (
	"log"
	"strings"
	"time"

	"github.com/jasonkolodziej/go-castv2/scanner"

	"github.com/google/gopacket/routing"
	"github.com/hashicorp/mdns"
)

// Hard defined device buffer size for
// Chromecasts are chatty so we wouldn't need to worry too much about lots of devices in one network. It's not really feasible.
const deviceBufferSearchSize = 100

// FindDevice finds a specific Device based on DeviceInfo given criteria
func FindDevice(find *DeviceInfo) (*Device, error) {
	// var err error
	devices := make(chan *Device, 100)
	FindDevices(time.Second*5, devices)
	for device := range devices {
		if device.Resembles(*find) {
			return device, nil
		}
	}
	return nil, nil
}

// FindDevices searches the LAN for chromecast devices via mDNS and sends them to a channel.
func FindDevices(timeout time.Duration, devices chan<- *Device) { // * recieve only channel

	// Make a channel for results and start listening
	entries := make(chan *mdns.ServiceEntry, deviceBufferSearchSize)

	go lookupChromecastMDNSEntries(entries, timeout)
	go createDeviceObjects(entries, devices)
}

func appendDeviceInfo(devices <-chan *Device, skipScan bool) {
	// defer close(devices)
	router, err := routing.New()
	if err != nil {
		log.Fatal("routing error:", err)
	}
	for device := range devices {
		if device.Info == nil || device.Info.IPAddress() == "" || skipScan {
			return
		}
		s, err := scanner.NewScanner(*device.Info.IpAddress, router) //* Create a scanner for the device using mdns.ServiceEntry
		if err != nil {
			log.Fatal("scanner error:", err)
		}
		mac, err := (*s).GetHwAddr(scanner.DefaultHwAddrParam)
		device.Info.hwAddr = &mac
		s.Close()
	}
}

// createDeviceObjects populates devices, a send-only channel, with Device, only when an entry from
// entries, a recieve-only channel, is properly populated
func createDeviceObjects(entries <-chan *mdns.ServiceEntry, devices chan<- *Device) {
	defer close(devices)
	// Create a new router to use
	router, err := routing.New()
	if err != nil {
		log.Fatal("routing error:", err)
	}
	for entry := range entries {
		if !strings.Contains(entry.Name, chromecastServiceName) {
			return
		}
		//* Create a scanner for the device using mdns.ServiceEntry

		scanner, err := scanner.NewScanner(entry.AddrV4, router)
		if err != nil {
			log.Fatal("scanner error:", err)
		}
		device, err := NewDevice(entry, scanner)
		if err != nil {
			return
		}
		devices <- &device
	}
}

// lookupChromecastMDNSEntries returns nil after querying and populating entries, a send-only channel, for the time.Duration, timeout.
func lookupChromecastMDNSEntries(entries chan<- *mdns.ServiceEntry, timeout time.Duration) {
	defer close(entries)
	mdns.Query(&mdns.QueryParam{
		DisableIPv6: true,
		Service:     chromecastServiceName,
		Timeout:     timeout,
		Entries:     entries,
	})
}
