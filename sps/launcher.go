package sps

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"

	zlog "github.com/rs/zerolog"
)

const binName = "shairport-sync"

var z = zlog.New(os.Stdout).With().Timestamp().Caller().Logger()

func SpawnProcessRC(args ...string) (proc *exec.Cmd, out, errno io.ReadCloser) {
	p := exec.Command("shairport-sync", args...)
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
	// return defer out.Close()
	// go func() {
	// 	for outS.Scan() {
	// 		// Do something with the line here.
	// 		fmt.Println(outS.Text())
	// 	}
	// }()
	// go func() {
	// 	for errnoS.Scan() {
	// 		// Do something with the line here.
	// 		// er = fmt.Errorf("%s%s", er, escanner.Text())
	// 		fmt.Println(errnoS.Text())
	// 	}
	// }()

	// if outS.Err() != nil {
	// 	p.Process.Kill()
	// 	p.Wait()
	// 	z.Err(err).Msg("stdOutpipe Scanner")
	// }
	// if errnoS.Err() != nil {
	// 	p.Process.Kill()
	// 	p.Wait()
	// 	z.Err(err).Msg("stdErrPipe Scanner")
	// }
	// p.Process.Kill()
	p.Wait()
	z.Info().Msg("exiting")
	// t.Logf("%s", out)
	return
}

func SpawnProcess(args ...string) (outS, errnoS *bufio.Scanner) {
	p := exec.Command("shairport-sync", args...)
	// p := exec.Command("ls", "/usr/local/bin")
	out, err := p.StdoutPipe() // * io.ReadCloser
	if err != nil {
		z.Err(err)
	}
	errno, err := p.StderrPipe()
	if err != nil {
		z.Err(err)
	}
	// var er error
	outS = bufio.NewScanner(out)
	errnoS = bufio.NewScanner(errno)
	err = p.Start()
	if err != nil {
		z.Err(err)
	}
	// return defer out.Close()
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
	// p.Process.Kill()
	p.Wait()
	z.Info().Msg("exiting")
	// t.Logf("%s", out)
	return
}
