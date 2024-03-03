package castv2

import (
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/mdns"
	"github.com/jasonkolodziej/go-castv2/configs"
	"github.com/jasonkolodziej/go-castv2/controllers"
	"github.com/jasonkolodziej/go-castv2/controllers/media"
	"github.com/jasonkolodziej/go-castv2/controllers/receiver"
	"github.com/jasonkolodziej/go-castv2/primitives"
)

const defaultTimeout = time.Second * 10

// Device Object to run basic chromecast commands
type Device struct {
	client               *primitives.Client
	heartbeatController  *controllers.HeartbeatController
	connectionController *controllers.ConnectionController
	ReceiverController   *controllers.ReceiverController
	MediaController      *controllers.MediaController
	YoutubeController    *controllers.YoutubeController
	svcRecord            *mdns.ServiceEntry
	Info                 *DeviceInfo
}

type DeviceInfo struct {
	Id    uuid.UUID
	Cd    uuid.UUID
	Md    string
	Fn    string
	other map[string]string
	// id=UUID cd=UUID rm= ve=05 md=Google Home ic=/setup/icon.png fn=Kitchen speaker ca=199172 st=0 bs=??? nf=1 rs=
}

func (i *DeviceInfo) IsGroup() bool {
	return strings.Contains(i.Md, "Google Cast Group")
}

func (i *DeviceInfo) IsTv() (bool, string) {
	return strings.Contains(strings.ToLower(i.Fn), "tv"), i.Md
}

func FromServiceEntryInfo(info []string) *DeviceInfo {
	var d DeviceInfo
	d.other = make(map[string]string)
	for _, item := range info {
		kv := strings.Split(item, "=")
		d.other[kv[0]] = kv[1]
	}
	d.Id = uuid.MustParse(d.other["id"])
	d.Cd = uuid.MustParse(d.other["cd"])
	d.Md = d.other["md"]
	d.Fn = d.other["fn"]
	return &d
}

// NewDevice is constructor for Device struct
func NewDevice(host net.IP, port int, record *mdns.ServiceEntry) (Device, error) {
	var device Device

	client, err := primitives.NewClient(host, port)
	if err != nil {
		return device, err
	}
	device.client = client
	device.svcRecord = record
	device.Info = FromServiceEntryInfo(record.InfoFields)

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
