package hls

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	logg "github.com/sirupsen/logrus"

	"github.com/go-audio/audio"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// ref: https://stackoverflow.com/questions/33089523/how-to-mark-golang-struct-as-implementing-interface
// var _ io.ReadWriteCloser = FLACStream{}       // Verify that T implements I.
// var _ io.ReadWriteCloser = (*FLACStream)(nil) // Verify that *T implements I.

type PCMBuffer = audio.PCMBuffer

// defaultStreamInfo [ref]:https://github.com/ains/aircast/blob/236f5e860e4e962c880096faad59a275ffae678e/src/aircast.py#L22
var DefaultStreamInfo = &meta.StreamInfo{
	SampleRate:    44100,
	NChannels:     2,
	BitsPerSample: 16,
	// ! compression_level should result to 8
}

// var rw = bufio.NewReadWriter(dec, )
var eInfo = &meta.StreamInfo{ // * Start the initialization of the Encoder
	// 	SampleRate:    44100,
	// 	NChannels:     2,
	// 	BitsPerSample: 16, // dec.NumChannels
	// 	// ! compression_level should result to 8
	// Minimum block size (in samples) used in the stream; between 16 and
	// 65535 samples.
	BlockSizeMin: 16, // adjusted by encoder.
	// Maximum block size (in samples) used in the stream; between 16 and
	// 65535 samples.
	BlockSizeMax: 65535, // adjusted by encoder.
	// Minimum frame size in bytes; a 0 value implies unknown.
	//FrameSizeMin // set by encoder.
	// Maximum frame size in bytes; a 0 value implies unknown.
	//FrameSizeMax // set by encoder.
	// Sample rate in Hz; between 1 and 655350 Hz.
	SampleRate: 44100,
	// Number of channels; between 1 and 8 channels.
	NChannels: 2,
	// Sample size in bits-per-sample; between 4 and 32 bits.
	BitsPerSample: 16,
	// Total number of inter-channel samples in the stream. One second of
	// 44.1 KHz audio will have 44100 samples regardless of the number of
	// channels. A 0 value implies unknown.
	//NSamples // set by encoder.
	// MD5 checksum of the unencoded audio data.
	//MD5sum // set by encoder.
}

var Broadcaster = flac.New //* used to pass stdOutPipe or stdOut from exec.Cmd
var fakeWriter = bytes.NewBuffer([]byte{})

var z = zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

var BroadcasterEncoder = flac.NewEncoder // * example: flac.NewEncoder(fakeWriter, defaultStreamInfo, nil)

// var RawAudio = new(chunk.Reader) //* used to pass stdOutPipe or stdOut from exec.Cmd

// packetStream is a wrapper for a socket connection for easier uses.
type PacketStream struct {
	stream  io.ReadWriteCloser
	packets chan *[]byte
}

// AudioBuffer returns an audio.IntBuffer with nChannels, sampleRate, and bps from a Decoder or
// if streamer is not nil, associated values will be used instead
func AudioBuffer(streamer *meta.StreamInfo, nChannels, sampleRate, bps int) *audio.IntBuffer {
	if streamer != nil {
		nChannels = int(streamer.NChannels)
		sampleRate = int(streamer.SampleRate)
		bps = int(streamer.BitsPerSample)
	}
	const nsamplesPerChannel = 16 // * Number of samples per channel and block
	nsamplesPerBlock := nChannels * nsamplesPerChannel
	return &audio.IntBuffer{ // * Initialize an audio.Buffer of type audio.IntBuffer
		Format: &audio.Format{
			NumChannels: nChannels,
			SampleRate:  sampleRate,
		},
		Data:           make([]int, nsamplesPerBlock),
		SourceBitDepth: bps,
	}
}

func CalculateSubFrames(streamer *meta.StreamInfo, nChannels int) []*frame.Subframe {
	const nsamplesPerChannel = 16 // * Number of samples per channel and block
	if streamer != nil {
		nChannels = int(streamer.NChannels)
	}
	subframes := make([]*frame.Subframe, nChannels) // * Calculate the subframes for the given number of channels
	for i := range subframes {
		subframe := &frame.Subframe{
			// SubHeader: frame.SubHeader{
			// 	Pred:   frame.PredVerbatim, // * Specifies the prediction method used to encode the audio sample of the subframe.
			// 	Order:  0,                  // * Prediction order used by fixed and FIR linear prediction decoding.
			// 	Wasted: 0,                  //* Wasted bits-per-sample.
			// },
			Samples: make([]int32, nsamplesPerChannel),
		}
		subframes[i] = subframe
	} // * End of initializing the SubFrame buffer
	return subframes
}

func UpdateSamplesField(subframes *[]*frame.Subframe, bufferData *[]int, n int, nChannel int) {
	for _, subframe := range *subframes {
		subHdr := frame.SubHeader{
			Pred:   frame.PredVerbatim, // * Specifies the prediction method used to encode the audio sample of the subframe.
			Order:  0,                  // * Prediction order used by fixed and FIR linear prediction decoding.
			Wasted: 0,                  //* Wasted bits-per-sample.
		}
		subframe.SubHeader = subHdr
		subframe.NSamples = n / nChannel
		subframe.Samples = subframe.Samples[:subframe.NSamples]
	}
	for i, sample := range *bufferData {
		subframe := (*subframes)[i%nChannel]
		subframe.Samples[i/nChannel] = int32(sample) // ! This line panics at frameNum == 82687
	}
	for _, subframe := range *subframes { //*  Check if the subframe may be encoded as constant; when all samples are the same
		sample := subframe.Samples[0]
		constant := true
		for _, s := range subframe.Samples[1:] {
			if sample != s {
				constant = false
			}
		}
		if constant {
			// t.Log("subframe was encoded with a constant method")
			subframe.SubHeader.Pred = frame.PredConstant
		}
	}
}

func NewFrame(head *frame.Header, frames []*frame.Subframe) *frame.Frame {
	return &frame.Frame{Header: *head, Subframes: frames}
}

func NewFrameHeaderBasedOnNBytes(streamer *meta.StreamInfo, nBytesRead, dBlockSize, sampleRate, nChannel, bps int) *frame.Header {
	var nBlockSize int = dBlockSize
	if streamer != nil {
		dBlockSize = int(streamer.BlockSizeMin)
		sampleRate = int(streamer.SampleRate)
		nChannel = int(streamer.NChannels)
		bps = int(streamer.BitsPerSample)
	}
	if nBytesRead >= 0 { // * nBytesRead needs to be used instead of default
		nBlockSize = nBytesRead
	}
	ch, _ := getChannels(nChannel)
	return &frame.Header{
		// Specifies if the block size is fixed or variable.
		HasFixedBlockSize: false,
		// Block size in inter-channel samples, i.e. the number of audio samples
		// in each subframe.
		BlockSize: uint16(nBlockSize),
		// Sample rate in Hz; a 0 value implies unknown, get sample rate from
		// StreamInfo.
		SampleRate: uint32(sampleRate),
		// Specifies the number of channels (subframes) that exist in the frame,
		// their order and possible inter-channel decorrelation.
		Channels: ch,
		// Sample size in bits-per-sample; a 0 value implies unknown, get sample
		// size from StreamInfo.
		BitsPerSample: uint8(bps),
		// Specifies the frame number if the block size is fixed, and the first
		// sample number in the frame otherwise. When using fixed block size, the
		// first sample number in the frame can be derived by multiplying the
		// frame number with the block size (in samples).
		//Num // set by encoder.
	}
}

// getChannels returns the channels assignment matching the given number of
// channels.
func getChannels(nchannels int) (frame.Channels, error) {
	switch nchannels {
	case 1:
		// 1 channel: mono.
		return frame.ChannelsMono, nil
	case 2:
		// 2 channels: left, right.
		return frame.ChannelsLR, nil
		//return frame.ChannelsLeftSide, nil  // 2 channels: left, side; using inter-channel decorrelation.
		//return frame.ChannelsSideRight, nil // 2 channels: side, right; using inter-channel decorrelation.
		//return frame.ChannelsMidSide, nil   // 2 channels: mid, side; using inter-channel decorrelation.
	case 3:
		// 3 channels: left, right, center.
		return frame.ChannelsLRC, nil
	case 4:
		// 4 channels: left, right, left surround, right surround.
		return frame.ChannelsLRLsRs, nil
	case 5:
		// 5 channels: left, right, center, left surround, right surround.
		return frame.ChannelsLRCLsRs, nil
	case 6:
		// 6 channels: left, right, center, LFE, left surround, right surround.
		return frame.ChannelsLRCLfeLsRs, nil
	case 7:
		// 7 channels: left, right, center, LFE, center surround, side left, side right.
		return frame.ChannelsLRCLfeCsSlSr, nil
	case 8:
		// 8 channels: left, right, center, LFE, left surround, right surround, side left, side right.
		return frame.ChannelsLRCLfeLsRsSlSr, nil
	default:
		return 0, errors.Errorf("support for %d number of channels not yet implemented", nchannels)
	}
}

// newPacketStream is the constructor.
func NewPacketStream(stream io.ReadWriteCloser) *PacketStream {
	wrapper := PacketStream{stream, make(chan *[]byte)}
	wrapper.readPackets()
	return &wrapper
}

// Continually processes events from the stream.
func (w *PacketStream) readPackets() {
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

func (w *PacketStream) read() *[]byte {
	return <-w.packets
}

// func (w *PacketStream) Read([]byte) (int, error) {
// 	return <-w.packets
// }

// Sends events to the stream to be read.
func (w *PacketStream) Write(data *[]byte) (int, error) {
	err := binary.Write(w.stream, binary.BigEndian, uint32(len(*data)))
	if err != nil {
		logg.Errorf("Failed to write packet length %d. error:%s", len(*data), err)
		return 0, err
	}
	return w.stream.Write(*data)
}

func (p *PacketStream) AsByteBuffer() *bytes.Buffer {
	return bytes.NewBuffer(*p.read())
}

func (p *PacketStream) AsReaderWriter() *bufio.ReadWriter {
	return bufio.NewReadWriter(bufio.NewReader(p.AsByteBuffer()), bufio.NewWriter(p.AsByteBuffer()))
}

func (w *PacketStream) WriteWith(bWriter *bufio.Writer) (int, error) {
	return bWriter.Write(*w.read())
}

func (w *PacketStream) ReadWith(bReader *bufio.Reader) (int, error) {
	return bReader.Read(*w.read())
}

func (w *PacketStream) Read(data []byte) (int, error) {
	err := binary.Read(w.stream, binary.BigEndian, uint32(len(data)))
	if err != nil {
		logg.Errorf("Failed to write packet length %d. error:%s", len(data), err)
		return 0, err
	}
	return w.stream.Read(data)
}
