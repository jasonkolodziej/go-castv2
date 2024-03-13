package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/jasonkolodziej/go-castv2/hls"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

func Test_Encoder(t *testing.T) {
	f, _ := loadTestSound(t, "PinkPanther60.wav")
	if f == nil {
		t.Fatal("Error file was not loaded correctly")
	}
	defer f.Close()

	dec := wav.NewDecoder(f) // * Create the Decoder
	if !dec.IsValidFile() {
		t.Errorf("invalid WAV file %s", f.Name())
	}
	sampleRate, nchannels, bps := int(dec.SampleRate), int(dec.NumChans), int(dec.BitDepth)
	t.Logf("Initialized Decoder SampleRate: %v, NChannels: %v, BitsPerSample: %v, File: %s", sampleRate, nchannels, bps, f.Name())
	t.Logf("Determined Number of Channels from Decoder, %v", nchannels)

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
		SampleRate: uint32(sampleRate),
		// Number of channels; between 1 and 8 channels.
		NChannels: uint8(nchannels),
		// Sample size in bits-per-sample; between 4 and 32 bits.
		BitsPerSample: uint8(bps),
		// Total number of inter-channel samples in the stream. One second of
		// 44.1 KHz audio will have 44100 samples regardless of the number of
		// channels. A 0 value implies unknown.
		//NSamples // set by encoder.
		// MD5 checksum of the unencoded audio data.
		//MD5sum // set by encoder.
	}
	t.Logf("Created flac.StreamInfo, %v", eInfo)

	// outPath := sourcePath[:len(sourcePath)-len(filepath.Ext(sourcePath))] + ".aif"
	pwd, _ := os.Getwd()
	of, err := os.Create(pwd + "/data/PinkPanther60.flac")
	if err != nil {
		fmt.Println("Failed to create", pwd+"/PinkPanther60.flac")
		t.Fatal(err)
	}
	defer of.Close()

	enc, err := flac.NewEncoder(of, eInfo) // * temperarily passes a new buffer created
	if err != nil {
		t.Error(err)
	}
	defer enc.Close() // * End of initializing the Encoder
	t.Logf("Initialized flac.Encoder")

	if err := dec.FwdToPCM(); err != nil { // * Forward audio frames into the PCM
		t.Error(err)
	}
	t.Logf("Forwarding Decoder Frames to PCM")

	buf := hls.AudioBuffer(nil, nchannels, sampleRate, bps)
	bb := audio.Buffer(buf)
	t.Logf("Number of frames: %v", bb.NumFrames())
	t.Logf("Initialized an audio.IntBuffer for audio.Buffer(): %v", *buf)

	const nsamplesPerChannel = 16 // * Number of samples per channel and block
	subframes := hls.CalculateSubFrames(nil, nchannels)
	t.Logf("Initialized []frame.Subframe size: %v; with .[]Samples size: %v", nchannels, nsamplesPerChannel)

	// var n int
	// for err == nil {
	// 	n, err = dec.PCMBuffer(buf)
	// 	if err != nil {
	// 		break
	// 	}
	// 	if n == 0 {
	// 		break
	// 	}
	// 	if n != len(buf.Data) {
	// 		buf.Data = buf.Data[:n]
	// 	}
	// 	if err := enc.WriteFrame(buf); err != nil {
	// 		panic(err)
	// 	}
	// }

	t.Logf("Performing decoding until EOF...")
	for frameNum := 0; !dec.EOF(); frameNum++ { // * Decode WAV samples by obtaining the PCM Data Packets
		t.Log("frame number:", frameNum)
		nBlockSize := nsamplesPerChannel
		n, err := dec.PCMBuffer(buf)

		if err != nil {
			t.Error(err)
			break
		}
		if n == 0 {
			break
		}
		if n != len(buf.Data) { // * Decoder has read the some blocks before EOF
			buf.Data = buf.Data[:n]
			nBlockSize = n
		}
		// t.Log("Initializing SubFrame.SubHeader")
		// for _, subframe := range subframes {
		// 	subHdr := frame.SubHeader{
		// 		Pred:   frame.PredVerbatim, // * Specifies the prediction method used to encode the audio sample of the subframe.
		// 		Order:  0,                  // * Prediction order used by fixed and FIR linear prediction decoding.
		// 		Wasted: 0,                  //* Wasted bits-per-sample.
		// 	}
		// 	subframe.SubHeader = subHdr
		// 	subframe.NSamples = n / nchannels
		// 	subframe.Samples = subframe.Samples[:subframe.NSamples]
		// }
		hls.UpdateSamplesField(&subframes, &buf.Data, n, nchannels)

		// t.Log("Converting buf.Data (# of Samples / Block)")
		// for i, sample := range buf.Data {
		// 	subframe := subframes[i%nchannels]
		// 	subframe.Samples[i/nchannels] = int32(sample) // ! This line panics at frameNum == 82687
		// }
		// t.Log("Checking if all Samples in SubFrames are the same")
		// for _, subframe := range subframes { //*  Check if the subframe may be encoded as constant; when all samples are the same
		// 	sample := subframe.Samples[0]
		// 	constant := true
		// 	for _, s := range subframe.Samples[1:] {
		// 		if sample != s {
		// 			constant = false
		// 		}
		// 	}
		// 	if constant {
		// 		// t.Log("subframe was encoded with a constant method")
		// 		subframe.SubHeader.Pred = frame.PredConstant
		// 	}
		// }
		// Encode FLAC frame.
		// channels, err := getChannels(nchannels)
		// if err != nil {
		// 	t.Fatal(err)
		// }
		// t.Log("channels: ", channels)
		// var hdr = frame.Header{
		// 	// Specifies if the block size is fixed or variable.
		// 	HasFixedBlockSize: false,
		// 	// Block size in inter-channel samples, i.e. the number of audio samples
		// 	// in each subframe.
		// 	BlockSize: uint16(nBlockSize),
		// 	// Sample rate in Hz; a 0 value implies unknown, get sample rate from
		// 	// StreamInfo.
		// 	SampleRate: uint32(sampleRate),
		// 	// Specifies the number of channels (subframes) that exist in the frame,
		// 	// their order and possible inter-channel decorrelation.
		// 	Channels: channels,
		// 	// Sample size in bits-per-sample; a 0 value implies unknown, get sample
		// 	// size from StreamInfo.
		// 	BitsPerSample: uint8(bps),
		// 	// Specifies the frame number if the block size is fixed, and the first
		// 	// sample number in the frame otherwise. When using fixed block size, the
		// 	// first sample number in the frame can be derived by multiplying the
		// 	// frame number with the block size (in samples).
		// 	//Num // set by encoder.
		// }
		// t.Log(hdr)
		// f := &frame.Frame{
		// 	Header:    hdr,
		// 	Subframes: subframes,
		// }
		f := hls.NewFrame(
			hls.NewFrameHeaderBasedOnNBytes(nil, nBlockSize, nBlockSize, sampleRate, nchannels, bps),
			subframes)

		//nf, err := frame.New(dec.PCMChunk)
		// t.Logf("Initialized flac frame.Frame with Header: %v", hdr)
		if err := enc.WriteFrame(f); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("flac.Encoder wrote all frames :)")
}
