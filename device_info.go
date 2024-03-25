package castv2

import (
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/mdns"
	"github.com/jasonkolodziej/go-castv2/controllers"
	"github.com/jasonkolodziej/go-castv2/primitives"
)

const defaultTimeout = time.Second * 10

type Port = int

const (
	CHROMECAST       Port = 8009
	CHROMECAST_GROUP Port = 32187
)

/** DeviceInfo struct
 * test
 */
type DeviceInfo struct {
	Id        uuid.UUID         `json:"Id"`
	Cd        uuid.UUID         `json:"Cd"`
	hwAddr    *net.HardwareAddr //? MAC Address (used for Airplay 2) "airplay_device_id"
	Md        string            `json:"Md"` //? Device type / Manufacturer
	Fn        string            `json:"Fn"` //? Friendly device name
	other     map[string]string
	port      *Port   //? Port number opened for the chromecast service
	IpAddress *net.IP `json:"IpAddress"`
	// id=UUID cd=UUID rm= ve=05 md=Google Home ic=/setup/icon.png fn=Kitchen speaker ca=199172 st=0 bs=??? nf=1 rs=
	paused bool
}

func Equal[DeviceInfo comparable](x, y DeviceInfo) bool {
	return reflect.DeepEqual(x, y)
}

func (i *DeviceInfo) IsGroup() bool {
	return strings.Contains(i.Md, "Google Cast Group") && (*i.port == CHROMECAST_GROUP)
}

func (i *DeviceInfo) SetPort(p int) {
	i.port = &p
}

func (i *DeviceInfo) Paused() bool {
	return i.paused
}

func (i *DeviceInfo) Port() Port {
	return *i.port
}

func (x *DeviceInfo) Resembles(y DeviceInfo) bool {
	return x.Md == y.Md ||
		(x.Port() == y.Port() || (x.IpAddress) == (y.IpAddress)) ||
		x.hwAddr == y.hwAddr ||
		x.Fn == y.Fn
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

func FromServiceEntryInfo(info []string, svcRecord *mdns.ServiceEntry, mac *net.HardwareAddr) *DeviceInfo {
	var d DeviceInfo
	d.hwAddr = mac
	d.other = make(map[string]string)
	if svcRecord != nil {
		d.port = &svcRecord.Port
		d.IpAddress = &svcRecord.Addr
	}
	if info != nil {
		for _, item := range info {
			kv := strings.Split(item, "=")
			d.other[kv[0]] = kv[1]
		}
		d.Id = uuid.MustParse(d.other["id"])
		d.Cd = uuid.MustParse(d.other["cd"])
		d.Md = d.other["md"]
		d.Fn = d.other["fn"]
	}

	// mac, err := net.ParseMAC(d.other["bs"])
	return &d
}

func NewDeviceFromDeviceInfo(info *DeviceInfo) (Device, error) {
	var device Device

	client, err := primitives.NewClient(*info.IpAddress, info.Port())
	if err != nil {
		return device, err
	}
	device.client = client
	device.Info = info

	// entries := make(chan *mdns.ServiceEntry, 5)
	// go lookupChromecastMDNSEntries(entries, time.Second*5)

	// for e := range entries {

	// }

	device.heartbeatController = controllers.NewHeartbeatController(client, defaultChromecastSenderID, defaultChromecastReceiverID)
	device.heartbeatController.Start()

	device.connectionController = controllers.NewConnectionController(client, defaultChromecastSenderID, defaultChromecastReceiverID)
	device.connectionController.Connect()

	device.ReceiverController = controllers.NewReceiverController(client, defaultChromecastSenderID, defaultChromecastReceiverID)

	device.MediaController = controllers.NewMediaController(client, defaultChromecastSenderID, device.ReceiverController)

	device.YoutubeController = controllers.NewYoutubeController(client, defaultChromecastSenderID, device.ReceiverController)
	return device, nil
}
