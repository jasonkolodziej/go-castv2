package sps

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	zlog "github.com/rs/zerolog"
)

const spss = "shairport-sync"
const txc = "ffmpeg"

/*
* ffmpegArgs https://ffmpeg.org/ffmpeg-protocols.html#toc-pipe
? (e.g. 0 for stdin, 1 for stdout, 2 for stderr).
  - $ ffmpeg -formats | grep PCM
  - DE alaw            PCM A-law
  - DE f32be           PCM 32-bit floating-point big-endian
  - DE f32le           PCM 32-bit floating-point little-endian
  - DE f64be           PCM 64-bit floating-point big-endian
  - DE f64le           PCM 64-bit floating-point little-endian
  - DE mulaw           PCM mu-law
  - DE s16be           PCM signed 16-bit big-endian
  - DE s16le           PCM signed 16-bit little-endian
  - DE s24be           PCM signed 24-bit big-endian
  - DE s24le           PCM signed 24-bit little-endian
  - DE s32be           PCM signed 32-bit big-endian
  - DE s32le           PCM signed 32-bit little-endian
  - DE s8              PCM signed 8-bit
  - DE u16be           PCM unsigned 16-bit big-endian
  - DE u16le           PCM unsigned 16-bit little-endian
  - DE u24be           PCM unsigned 24-bit big-endian
  - DE u24le           PCM unsigned 24-bit little-endian
  - DE u32be           PCM unsigned 32-bit big-endian
  - DE u32le           PCM unsigned 32-bit little-endian
  - DE u8              PCM unsigned 8-bit

Example:

	shairport-sync -c /etc/shairport-syncKitchenSpeaker.conf -o stdout \
		| ffmpeg -f s16le -ar 44100 -ac 2 -i pipe: -ac 2 -bits_per_raw_sample 8 -c:a flac -y flac_test1.flac
*/
var ffmpegArgs = []string{
	// * arguments
	"-f", "s16le",
	"-ar", "44100",
	"-ac", "2",
	// "-re",         // * encode at 1x playback speed, to not burn the CPU
	"-i", "pipe:", // * input from pipe (stdout->stdin)
	// "-ar", "44100", // * AV sampling rate
	"-c:a", "flac", // * audio codec
	// "-sample_fmt", "44100", // * sampling rate
	"-ac", "2", // * audio channels, chromecasts don't support more than two audio channels
	// "-f", "mp4", // * fmt force format
	"-bits_per_raw_sample", "8",
	"-f", "flac",
	"-movflags", "frag_keyframe+faststart",
	"-strict", "-experimental",
	"pipe:1", // * output to pipe (stdout->)
}

var z = zlog.New(os.Stdout).With().Timestamp().Caller().Logger()

func SpawnProcessRC(args ...string) (proc *exec.Cmd, out, errno io.ReadCloser) {
	p := exec.Command(spss, args...)
	// p := exec.Command("ls", "/usr/local/bin")
	out, err := p.StdoutPipe() // * io.ReadCloser

	if err != nil {
		z.Err(err)
	}
	errno, err = p.StderrPipe()
	if err != nil {
		z.Err(err)
	}
	// var er error
	// outS = bufio.NewScanner(out)
	// errnoS = bufio.NewScanner(errno)
	err = p.Start()
	if err != nil {
		z.Err(err)
	}
	return p, out, errno

	p.Wait()
	z.Info().Msg("exiting")
	// t.Logf("%s", out)
	return
}

func SpawnProcess(args ...string) (outS, errnoS *bufio.Scanner) {
	p := exec.Command("shairport-sync", args...)
	out, err := p.StdoutPipe() // * io.ReadCloser
	if err != nil {
		z.Err(err)
	}
	errno, err := p.StderrPipe()
	if err != nil {
		z.Err(err)
	}
	outS = bufio.NewScanner(out)
	errnoS = bufio.NewScanner(errno)
	err = p.Start()
	if err != nil {
		z.Err(err)
	}
	go func() {
		for outS.Scan() {
			// Do something with the line here.
			fmt.Println(outS.Text())
		}
	}()
	go func() {
		for errnoS.Scan() {
			// Do something with the line here.
			// er = fmt.Errorf("%s%s", er, escanner.Text())
			fmt.Println(errnoS.Text())
		}
	}()
	if outS.Err() != nil {
		p.Process.Kill()
		p.Wait()
		z.Err(err).Msg("stdOutpipe Scanner")
	}
	if errnoS.Err() != nil {
		p.Process.Kill()
		p.Wait()
		z.Err(err).Msg("stdErrPipe Scanner")
	}
	p.Wait()
	z.Info().Msg("exiting")
	return
}

func SpawnProcessWConfig(configPath string) (out io.ReadCloser, errno io.ReadCloser, p *exec.Cmd, err error) {
	p = exec.Command("shairport-sync", "-c", configPath)
	out, err = p.StdoutPipe() // * io.ReadCloser
	if err != nil {
		z.Err(err).Send()
	}
	errno, err = p.StderrPipe()
	if err != nil {
		z.Err(err).Send()
	}
	// outS = bufio.NewScanner(out)
	// errnoS = bufio.NewScanner(errno)
	err = p.Start()
	if err != nil {
		z.Err(err).Send()
	}

	z.Info().Msg("exiting")
	return out, errno, p, nil
}

func SpawnProcessConfig(configPath ...string) (out io.ReadCloser, errno io.ReadCloser, err error) {
	p := exec.Command("shairport-sync", configPath...)
	out, err = p.StdoutPipe() // * io.ReadCloser
	if err != nil {
		z.Err(err)
	}
	errno, err = p.StderrPipe()
	if err != nil {
		z.Err(err)
	}
	err = p.Start()
	if err != nil {
		z.Err(err)
	}
	z.Info().Msg("returning")
	return out, errno, p.Wait()
}

// func SpawnFfMpegWith(in <-chan io.ReadCloser, args ...string) (output io.ReadCloser, errno io.ReadCloser, err error) {
// 	var input io.ReadCloser = <-in
// 	return SpawnFfMpeg(input, ffmpegArgs...)
// }

func SpawnFfMpeg(input io.ReadCloser, args ...string) (output io.ReadCloser, errno io.ReadCloser, p *exec.Cmd, err error) {
	defer input.Close()
	if len(args) == 0 {
		args = ffmpegArgs
	}
	cmd := exec.Command(
		txc,
		args...,
	)
	cmd.Stdin = input
	output, err = cmd.StdoutPipe()
	if err != nil {
		z.Err(err).Send()
	}
	errno, err = cmd.StderrPipe()
	if err != nil {
		z.Err(err).Send()
	}
	// go z.Error()
	err = cmd.Start()
	if err != nil {
		z.Err(err).Send()
	}
	return output, errno, cmd, err
}

// func RunPiping(config string) (encoded io.ReadCloser, spsErr io.ReadCloser, txcErr io.ReadCloser, cErr error) {
// 	out, spsErr, cErr := SpawnProcessWConfig(config)
// 	if cErr != nil {
// 		z.Err(cErr).Msg("shairport-sync Wait():")
// 		return nil, nil, nil, cErr
// 	}
// 	defer out.Close()
// 	defer spsErr.Close()
// 	encoded, txcErr, cErr = SpawnFfMpeg(out)
// 	if cErr != nil {
// 		z.Err(cErr).Msg("FFMpeg Wait():")
// 		return nil, nil, nil, cErr
// 	}
// 	return encoded, spsErr, txcErr, nil
// }

func createPipe() error {
	// ? Equivalent: $ ls /usr/local/bin | grep pip
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer r.Close()
	ls := exec.Command("ls", "/usr/local/bin")
	ls.Stdout = w
	err = ls.Start()
	if err != nil {
		return err
	}
	defer ls.Wait()
	w.Close()
	grep := exec.Command("grep", "pip")
	grep.Stdin = r
	grep.Stdout = os.Stdout
	return grep.Run()
}

func PipePeeker(r io.ReadCloser) (content bool) {
	// defer r.Close()
	peek := bufio.NewReader(r)
	// peeker := bufio.NewScanner(peek)
	// peeker.Split(bufio.ScanBytes)
	// var bRead = 0
	// for peeker.Scan() {
	for {
		if peeked, err := peek.Peek(1); err != nil && len(peeked) == 1 {
			return true // * there is content in the pipe
		}
		return false
	}
	// tt := bytes.Trim(peeked, "\x00")
	// tt = bytes.Trim(tt, "\xff")
	// }
}

func PerformWhenContent(maybe io.ReadCloser, f func(io.ReadCloser, ...string) (io.ReadCloser, io.ReadCloser, error)) {
	defer maybe.Close()
	if PipePeeker(maybe) {
		go f(maybe)
	}
}
