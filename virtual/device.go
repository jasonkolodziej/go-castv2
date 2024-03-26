package virtual

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/jasonkolodziej/go-castv2"
	"github.com/jasonkolodziej/go-castv2/sps"
	"github.com/rs/zerolog"
)

var z = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

// shairport-sync -c /etc/shairport-syncKitchenSpeaker.conf -o stdout | ffmpeg -f s16le -ar 44100 -ac 2 -i pipe: -ac 2 -bits_per_raw_sample 8 -c:a pcm_s32le -y flac_test1.wav

type VirtualDevice struct {
	*castv2.Device
	content        io.ReadCloser
	rawContent     io.ReadCloser
	ctx            context.Context
	Cancel         context.CancelFunc
	virtualhostAdr net.Addr
	sps, ffmpeg    *exec.Cmd
	connectionPool *ConnectionPool
	contentType    *string
	// mu             sync.Mutex
}

// * curl -s -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:5123/devices/<deviceId>/connect

func NewVirtualDevice(d *castv2.Device, ctx context.Context) *VirtualDevice {
	var v *VirtualDevice
	var contentType = "audio/aac"
	if d == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.TODO()
	}
	return &VirtualDevice{
		Device:     d,
		content:    nil,
		rawContent: nil,
		ctx:        ctx,
		Cancel: func() {
			v.teardown()
		},
		virtualhostAdr: nil,
		sps:            nil,
		ffmpeg:         nil,
		connectionPool: NewConnectionPool(),
		contentType:    &contentType,
		// mu:             sync.Mutex{},
	}
	// Check for sps device conf
	// v.checkForConfigFile()
}

func (v *VirtualDevice) teardown() error {
	// defer close(v.content)
	defer v.content.Close()
	defer v.ffmpeg.Cancel()
	defer v.sps.Cancel()
	_ = v.sps.Wait()
	_ = v.ffmpeg.Wait()
	<-v.ctx.Done()
	return v.ctx.Err()
}

func (v *VirtualDevice) ZoneName() string {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	_, n := v.Device.Info.AirplayDeviceName()
	return n
}

func (v *VirtualDevice) StartStream() {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	GetStreamFromReader(v.connectionPool, v.content)
}

func (v *VirtualDevice) StartAndGoWait() {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	if err := v.Virtualize(); err != nil {
		z.Error().AnErr("StartAndGoWait", err).Msg("trying to Virtualize")
		return
	}
	// if err := v.StartTranscoder(); err != nil {
	// 	z.Error().AnErr("StartAndGoWait", err).Msg("trying to StartTranscoder")
	// 	return
	// }
	// go v.sps.Wait()
	// go v.ffmpeg.Wait()
}

func (v *VirtualDevice) VirtualHostAddr(netAddr net.Addr, hostname, port string) {
	if netAddr != nil {
		v.virtualhostAdr = netAddr
	} else {
		// v.virtualhostAdr;
	}
}

// Content populates VirtualDevice.content channel with a non-nil io.ReaderCloser coming from rc, a recieve-only channel
func (v *VirtualDevice) Content(rcvRc <-chan io.ReadCloser) {
	for rc := range rcvRc {
		if rc == nil {
			return
		}
		// v.content <- rc
	}
}

func (v *VirtualDevice) ConnectDeviceToVirtualStream() error {
	// defer v.mu.Unlock()
	// v.mu.Lock()
	if v == nil || v.virtualhostAdr == nil { // * basic sanity check
		return fmt.Errorf("device not created")
	}
	v.QuitApplication(time.Second * 5)
	v.PlayMedia("http://"+v.virtualhostAdr.String()+v.pathString(), "audio/aac", "")
	return nil
}

func (v *VirtualDevice) Output() (output io.ReadCloser, e io.ReadCloser, err error) {
	return nil, nil, nil
}

func (v *VirtualDevice) OutputWithArgs(configPath ...string) (output io.ReadCloser, e io.ReadCloser, err error) {
	var confFlag = append([]string{"-c"}, configPath...)
	return sps.SpawnProcessConfig(confFlag...)
}
