package hls

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"

	logg "github.com/sirupsen/logrus"

	aud "github.com/go-audio/audio"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

// ref: https://stackoverflow.com/questions/33089523/how-to-mark-golang-struct-as-implementing-interface
var _ io.ReadWriteCloser = FLACStream{}       // Verify that T implements I.
var _ io.ReadWriteCloser = (*FLACStream)(nil) // Verify that *T implements I.

type PCMBuffer = aud.PCMBuffer

// defaultStreamInfo [ref]:https://github.com/ains/aircast/blob/236f5e860e4e962c880096faad59a275ffae678e/src/aircast.py#L22
var defaultStreamInfo = &meta.StreamInfo{
	SampleRate:    44100,
	NChannels:     2,
	BitsPerSample: 16,
	// ! compression_level should result to 8
}

var Broadcaster = flac.New //* used to pass stdOutPipe or stdOut from exec.Cmd
var fakeWriter = bytes.NewBuffer([]byte{})

var BroadcasterEncoder = flac.NewEncoder // * example: flac.NewEncoder(fakeWriter, defaultStreamInfo, nil)

// var defaultStreamInfo =

type FLACStream struct {
	io.ReadWriteCloser
	packets chan *flac.Stream
	config  *meta.StreamInfo
}

// * Note: Please modify config after successful creation
func NewFLACStream(stream io.ReadWriteCloser, config *meta.StreamInfo) *FLACStream {
	w := FLACStream{stream, make(chan *flac.Stream), defaultStreamInfo} // * wrapper with Default config
	if config != nil {
		w = FLACStream{stream, make(chan *flac.Stream), config} // * wrapper
	}
	w.readPackets()
	return &w
}

func (f *FLACStream) readPackets() {
	// var l uint32 // * length
	go func() {
		for {
			s, err := Broadcaster(f)
			if err != nil {
				logg.Errorf("Failed to read packet length: %s", err)
				return
			}
			if reflect.DeepEqual(f.config, s.Info) { // * check to see if the set config made it
				_, err := s.Next() // * returns the next flac.Frame with Stream.Header ONLY
				if err != nil {
					logg.Errorf("Failed to read packet: %s", err)
					return
				}
				f.packets <- &*s // TODO: see if this fails
				// hb, err := s.ParseNext() // * returns the next flac.Frame with Stream.Header and Stream.Blocks
			}

		}
	}()
}

func (f *FLACStream) read() *flac.Stream {
	return <-f.packets
}

func (f *FLACStream) write(data *[]byte) (int, error) {
	e, err := BroadcasterEncoder(f, f.config) // * encoder or error
	if err != nil {
		logg.Errorf("Failed to write packet length %d. error:%s", len(*data), err)
		return 0, err
	}
	fr, err := e.Next()
	if err != nil {
		logg.Errorf("Failed to invoke next encoder frame length %d. error:%s", len(*data), err)
		return 0, err
	}
	err = e.WriteFrame(fr) // ? write to the header frame?
	if err != nil {
		logg.Errorf("Failed to write frame length %v. error:%s", fr, err)
		return 0, err
	}
	return int(fr.Num), nil
	// f.Write()
}

// packetStream is a wrapper for a socket connection for easier uses.
type packetStream struct {
	stream  io.ReadWriteCloser
	packets chan *[]byte
}

// newPacketStream is the constructor.
func newPacketStream(stream io.ReadWriteCloser) *packetStream {
	wrapper := packetStream{stream, make(chan *[]byte)}
	wrapper.readPackets()

	return &wrapper
}

// Continually processes events from the stream.
func (w *packetStream) readPackets() {
	var length uint32

	go func() {
		for {

			err := binary.Read(w.stream, binary.BigEndian, &length)
			if err != nil {
				logg.Errorf("Failed to read packet length: %s", err)
				return
			}

			if length > 0 {
				packet := make([]byte, length)

				i, err := w.stream.Read(packet)
				if err != nil {
					logg.Errorf("Failed to read packet: %s", err)
					return
				}

				if i != int(length) {
					logg.Errorf("Invalid packet size. Wanted: %d Read: %d", length, i)
					return
				}
				w.packets <- &packet
			}

		}
	}()
}

func (w *packetStream) read() *[]byte {
	return <-w.packets
}

// Sends events to the stream to be read.
func (w *packetStream) write(data *[]byte) (int, error) {

	err := binary.Write(w.stream, binary.BigEndian, uint32(len(*data)))

	if err != nil {
		logg.Errorf("Failed to write packet length %d. error:%s", len(*data), err)
		return 0, err
	}

	return w.stream.Write(*data)
}
