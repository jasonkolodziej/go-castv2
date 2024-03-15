package castv2

import (
	"fmt"
	"net"
	"reflect"
	"strings"

	// "text/scanner"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/mdns"
	"github.com/jasonkolodziej/go-castv2/configs"
	"github.com/jasonkolodziej/go-castv2/controllers"
	"github.com/jasonkolodziej/go-castv2/controllers/media"
	"github.com/jasonkolodziej/go-castv2/controllers/receiver"
	"github.com/jasonkolodziej/go-castv2/primitives"
	"github.com/jasonkolodziej/go-castv2/scanner"
)

const defaultTimeout = time.Second * 10

type Port = int

const (
	CHROMECAST       Port = 8009
	CHROMECAST_GROUP Port = 32187
)

// Device Object to run basic chromecast commands
type Device struct {
	client               *primitives.Client
	heartbeatController  *controllers.HeartbeatController
	connectionController *controllers.ConnectionController
	ReceiverController   *controllers.ReceiverController
	MediaController      *controllers.MediaController
	YoutubeController    *controllers.YoutubeController
	svcRecord            *mdns.ServiceEntry //? svcRecord is a pointer to the mDNS Service Entry of the Chromecast Device
	Info                 *DeviceInfo        //? Info extracts information from svcRecord to be Used in DeviceInfo struct
}

/** DeviceInfo struct
 * test
 */
type DeviceInfo struct {
	Id        uuid.UUID
	Cd        uuid.UUID
	hwAddr    *net.HardwareAddr //? MAC Address (used for Airplay 2) "airplay_device_id"
	Md        string            //? Device type / Manufacturer
	Fn        string            //? Friendly device name
	other     map[string]string
	port      *Port //? Port number opened for the chromecast service
	IpAddress *net.IP
	// id=UUID cd=UUID rm= ve=05 md=Google Home ic=/setup/icon.png fn=Kitchen speaker ca=199172 st=0 bs=??? nf=1 rs=
}

func (x *Device) Equal(y Device) bool {
	return reflect.DeepEqual(*x.Info, y.Info)
}

func (x *Device) Resembles(y DeviceInfo) bool {
	return x.Info.Md == y.Md ||
		(x.Info.port == y.port || (x.Info.IpAddress) == (y.IpAddress)) ||
		x.Info.hwAddr == y.hwAddr ||
		x.Info.Fn == y.Fn
}

func Equal[DeviceInfo comparable](x, y DeviceInfo) bool {
	return reflect.DeepEqual(x, y)
}

func (i *DeviceInfo) IsGroup() bool {
	return strings.Contains(i.Md, "Google Cast Group") && (*i.port == CHROMECAST_GROUP)
}

func (i *DeviceInfo) IsTv() (bool, string) {
	return strings.Contains(strings.ToLower(i.Fn), "tv"), i.Md
}

func (i *DeviceInfo) AirplayDeviceId() (key string, hwid net.HardwareAddr) {
	return "airplay_device_id", i.MAC()
}

func (i *DeviceInfo) MAC() net.HardwareAddr {
	return *i.hwAddr
}

func (i *DeviceInfo) AirplayDeviceName() (key string, val string) {
	return "name", i.Fn
}

func (i *DeviceInfo) IPAddress() string {
	return i.IpAddress.String()
}

func (d *Device) FiberDeviceHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Params("deviceId") != d.Info.Id.String() {
			_, name := d.Info.AirplayDeviceName()
			return c.SendString("Hello " + name)
			// => Hello john
		}
		return c.SendString("Where is john?")
	}
}

/*
discoverLocalInterfaces

	disovers interfaces used by the device executing this function
*/
func discoverLocalInterfaces() []net.Interface {
	var ret []net.Interface
	netFaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, face := range netFaces {
		addrs, err := face.Addrs()
		if err != nil {
			panic(err)
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
				fmt.Println(ipNet.IP)
				ret = append(ret, face)
			}
		}
	}
	return ret
}

//    0.000032000 "shairport.c:2401" daemon status is 0.
// 0.000040538 "shairport.c:1567" PID file: "/var/run/shairport-sync/shairport-sync.pid".
// 0.000088770 "shairport.c:2402" daemon pid file path is "/var/run/shairport-sync/shairport-sync.pid".

//          0.000070769 "rtsp.c:374" Creating metadata queue "multicast".
// 0.007861923 "mdns_avahi.c:220" avahi: service '9483C43DCE3B@GL-MT3000' group is not yet committed.
// 0.004553154 "mdns_avahi.c:277" avahi: avahi_entry_group_commit 0
// 0.000261769 "mdns_avahi.c:477" avahi_dacp_monitor_start Avahi DACP monitor successfully started
// 0.000255539 "mdns_avahi.c:224" avahi: service '9483C43DCE3B@GL-MT3000' group is registering.
// 0.883230385 "mdns_avahi.c:191" avahi: service '9483C43DCE3B@GL-MT3000' successfully added.
//          0.000028000 "audio_alsa.c:2039" keep_dac_busy is now "no"
// 0.000063615 "shairport.c:2409" run_this_before_play_begins action is "(null)".
// 0.000079616 "shairport.c:2410" run_this_after_play_ends action is "(null)".
// 0.000027538 "shairport.c:2411" wait-cmd status is 0.
// 0.000056385 "shairport.c:2412" run_this_before_play_begins may return output is 0.
// 0.000026846 "shairport.c:2413" run_this_if_an_unfixable_error_is_detected action is "(null)".
// 0.000025846 "shairport.c:2415" run_this_before_entering_active_state action is  "(null)".
// 0.000026000 "shairport.c:2417" run_this_after_exiting_active_state action is  "(null)".
// 0.000025385 "shairport.c:2419" active_state_timeout is  10.000000 seconds.

func FromServiceEntryInfo(info []string, svcRecord *mdns.ServiceEntry, mac *net.HardwareAddr) *DeviceInfo {
	var d DeviceInfo
	d.port = &svcRecord.Port
	d.IpAddress = &svcRecord.Addr
	d.hwAddr = mac
	d.other = make(map[string]string)
	for _, item := range info {
		kv := strings.Split(item, "=")
		d.other[kv[0]] = kv[1]
	}
	d.Id = uuid.MustParse(d.other["id"])
	d.Cd = uuid.MustParse(d.other["cd"])
	d.Md = d.other["md"]
	d.Fn = d.other["fn"]

	// mac, err := net.ParseMAC(d.other["bs"])
	return &d
}

// NewDevice is constructor for Device struct
// host net.IP, port int,
func NewDevice(record *mdns.ServiceEntry, s *scanner.Scanner) (Device, error) {
	var device Device

	client, err := primitives.NewClient(record.Addr, record.Port)
	if err != nil {
		return device, err
	}
	device.client = client
	device.svcRecord = record
	mac, err := (*s).GetHwAddr(scanner.DefaultHwAddrParam)
	if err != nil {
		return device, err
		// continue
	}
	s.Close()
	device.Info = FromServiceEntryInfo(record.InfoFields, record, &mac)

	device.heartbeatController = controllers.NewHeartbeatController(client, defaultChromecastSenderID, defaultChromecastReceiverID)
	device.heartbeatController.Start()

	device.connectionController = controllers.NewConnectionController(client, defaultChromecastSenderID, defaultChromecastReceiverID)
	device.connectionController.Connect()

	device.ReceiverController = controllers.NewReceiverController(client, defaultChromecastSenderID, defaultChromecastReceiverID)

	device.MediaController = controllers.NewMediaController(client, defaultChromecastSenderID, device.ReceiverController)

	device.YoutubeController = controllers.NewYoutubeController(client, defaultChromecastSenderID, device.ReceiverController)
	return device, nil
}

// Play just plays.
func (device *Device) Play() {
	device.MediaController.Play(defaultTimeout)
}

// PlayMedia plays a video via the media controller.
func (device *Device) PlayMedia(URL string, MIMEType string) {
	appID := configs.MediaReceiverAppID
	device.ReceiverController.LaunchApplication(&appID, defaultTimeout, false)
	device.MediaController.Load(URL, MIMEType, defaultTimeout)
}

// QuitApplication that is currently running on the device
func (device *Device) QuitApplication(timeout time.Duration) {
	status, err := device.ReceiverController.GetStatus(timeout)
	if err != nil {
		return
	}
	for _, appSessions := range status.Applications {
		session := appSessions.SessionID
		device.ReceiverController.StopApplication(session, timeout)
	}
}

// PlayYoutubeVideo launches the youtube app and tries to play the video based on its id.
func (device *Device) PlayYoutubeVideo(videoID string) {
	appID := configs.YoutubeAppID
	device.ReceiverController.LaunchApplication(&appID, defaultTimeout, false)
	device.YoutubeController.PlayVideo(videoID, "")
}

// GetMediaStatus of current media controller
func (device *Device) GetMediaStatus(timeout time.Duration) []*media.MediaStatus {
	response, err := device.MediaController.GetStatus(time.Second * 5)
	if err != nil {
		emptyStatus := make([]*media.MediaStatus, 0)
		return emptyStatus
	}
	return response
}

// GetStatus of the device.
func (device *Device) GetStatus(timeout time.Duration) *receiver.ReceiverStatus {
	response, err := device.ReceiverController.GetStatus(time.Second * 5)
	if err != nil {
		return nil
	}
	return response
}
