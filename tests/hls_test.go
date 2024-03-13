package tests

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
	"github.com/pkg/errors"
)

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

func Test_Encoder(t *testing.T) {
	f, fSize := loadTestSound(t, "PinkPanther60.wav")
	if f == nil {
		t.Fatal("Error file was not loaded correctly")
	}
	dec := wav.NewDecoder(f) // * Create the Decoder
	if !dec.IsValidFile() {
		t.Errorf("invalid WAV file %q", f.Name())
	}

	nchannels := int(dec.NumChans)
	var eInfo = &meta.StreamInfo{ // * Start the initialization of the Encoder
		SampleRate:    44100,
		NChannels:     2,
		BitsPerSample: 16, // dec.NumChannels
		// ! compression_level should result to 8
	}
	enc, err := flac.NewEncoder(bytes.NewBuffer(make([]byte, fSize)), eInfo) // * temperarily passes a new buffer created
	if err != nil {
		t.Error(err)
	}
	defer enc.Close()                      // * End of initializing the Encoder
	if err := dec.FwdToPCM(); err != nil { // * Forward audio frames into the PCM
		t.Error(err)
	}
	const nsamplesPerChannel = 16 // * Number of samples per channel and block
	nsamplesPerBlock := eInfo.NChannels * nsamplesPerChannel
	buf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: int(eInfo.NChannels),
			SampleRate:  int(eInfo.SampleRate),
		},
		Data:           make([]int, nsamplesPerBlock),
		SourceBitDepth: int(eInfo.BitsPerSample),
	}
	subframes := make([]*frame.Subframe, eInfo.NChannels) // * Calculate the subframes for the given number of channels
	for i := range subframes {
		subframe := &frame.Subframe{
			Samples: make([]int32, nsamplesPerChannel),
		}
		subframes[i] = subframe
	} // * End of initializing the SubFrame buffer
	for frameNum := 0; !dec.EOF(); frameNum++ { // * Decode WAV samples by obtaining the PCM Data Packets
		fmt.Println("frame number:", frameNum)
		n, err := dec.PCMBuffer(buf)
		if err != nil {
			t.Error(err)
		}
		if n == 0 {
			break
		}
		for _, subframe := range subframes {
			subHdr := frame.SubHeader{
				Pred:   frame.PredVerbatim, // * Specifies the prediction method used to encode the audio sample of the subframe.
				Order:  0,                  // * Prediction order used by fixed and FIR linear prediction decoding.
				Wasted: 0,                  //* Wasted bits-per-sample.
			}
			subframe.SubHeader = subHdr
			subframe.NSamples = n / nchannels
			subframe.Samples = subframe.Samples[:subframe.NSamples]
		}
		for i, sample := range buf.Data {
			subframe := subframes[i%nchannels]
			subframe.Samples[i/nchannels] = int32(sample)
		}
		for _, subframe := range subframes { //*  Check if the subframe may be encoded as constant; when all samples are the same
			sample := subframe.Samples[0]
			constant := true
			for _, s := range subframe.Samples[1:] {
				if sample != s {
					constant = false
				}
			}
			if constant {
				fmt.Println("constant method")
				subframe.SubHeader.Pred = frame.PredConstant
			}
		}
		channels, err := getChannels(nchannels) //* Encode FLAC frame.
		if err != nil {
			t.Error(err)
		}
		hdr := frame.Header{
			// Specifies if the block size is fixed or variable.
			HasFixedBlockSize: false,
			// Block size in inter-channel samples, i.e. the number of audio samples
			// in each subframe.
			BlockSize: uint16(nsamplesPerChannel),
			// Sample rate in Hz; a 0 value implies unknown, get sample rate from
			// StreamInfo.
			SampleRate: uint32(eInfo.SampleRate),
			// Specifies the number of channels (subframes) that exist in the frame,
			// their order and possible inter-channel decorrelation.
			Channels: channels,
			// Sample size in bits-per-sample; a 0 value implies unknown, get sample
			// size from StreamInfo.
			BitsPerSample: uint8(eInfo.BitsPerSample),
			// Specifies the frame number if the block size is fixed, and the first
			// sample number in the frame otherwise. When using fixed block size, the
			// first sample number in the frame can be derived by multiplying the
			// frame number with the block size (in samples).
			//Num // set by encoder.
		}
		f := &frame.Frame{
			Header:    hdr,
			Subframes: subframes,
		}
		if err := enc.WriteFrame(f); err != nil {
			t.Error(err)
		}
	}
}
