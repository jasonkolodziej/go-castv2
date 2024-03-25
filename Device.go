package castv2

import (
	"os"
	"reflect"

	// "text/scanner"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/jasonkolodziej/go-castv2/configs"
	"github.com/jasonkolodziej/go-castv2/controllers"
	"github.com/jasonkolodziej/go-castv2/controllers/media"
	"github.com/jasonkolodziej/go-castv2/controllers/receiver"
	"github.com/jasonkolodziej/go-castv2/primitives"
	"github.com/jasonkolodziej/go-castv2/scanner"
	"github.com/rs/zerolog"
)

var z = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

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
	currentStatus        *media.MediaStatus
}

func (x *Device) Equal(y Device) bool {
	return reflect.DeepEqual(*x.Info, y.Info)
}

func (x *Device) Resembles(y DeviceInfo) bool {
	return x.Info.Resembles(y)
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
	device.GetMediaStatus(defaultTimeout)
	return device, nil
}

// Play just plays.
func (device *Device) Pause() {
	device.MediaController.Pause(defaultTimeout)
	// device.Info.paused = true
}

func (device *Device) Play() {
	device.MediaController.Play(defaultTimeout)
	// device.Info.paused = false
}

// PlayMedia plays a video via the media controller.
func (device *Device) PlayMedia(URL, MIMEType, MediaStreamType string) {
	appID := configs.MediaReceiverAppID
	response, err := device.ReceiverController.LaunchApplication(&appID, defaultTimeout, false)
	if err != nil {
		z.Err(err).Msg("Device.PlayMedia:LaunchApplication")
	}
	z.Debug().Any("recieverController.LaunchApplication", response).Msg("Device.PlayMedia:")
	m, err := device.MediaController.Load(URL, MIMEType, MediaStreamType, defaultTimeout)
	if err != nil {
		z.Err(err).Msg("Device.PlayMedia:")
	}
	z.Debug().Any("castMessage", m).Msg("Device.PlayMedia:")

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
	device.currentStatus = response[0]
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

func (device *Device) GetVolume(timeout time.Duration) *receiver.Volume {
	if device.currentStatus != nil {
		return (*receiver.Volume)(device.currentStatus.Volume)
	} else {
		device.GetMediaStatus(time.Second * 5)
	}
	if device.currentStatus != nil {
		return (*receiver.Volume)(device.currentStatus.Volume)
	}
	response, err := device.ReceiverController.GetVolume(time.Second * 5)
	if err != nil {
		return nil
	}
	return response
}

func (device *Device) SetVolume(level float64, muted bool, timeout time.Duration) {
	_, err := device.ReceiverController.SetVolume(
		&receiver.Volume{Level: &level, Muted: &muted},
		time.Second*5)
	if err != nil {
		z.Warn().AnErr("SetVolume", err).Msg("Device")
		return
	}
	device.currentStatus.Volume = &media.Volume{Level: &level, Muted: &muted}
}
