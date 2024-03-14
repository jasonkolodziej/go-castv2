package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"

	logger "github.com/sirupsen/logrus"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	// logger.SetFormatter(&logger.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logger.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logger.SetLevel(logger.ErrorLevel)

}

var linform = logger.Infof
var ldebug = logger.Debugf
var lwarn = logger.Warnf
var lerr = logger.Errorf

func main() {
	p := exec.Command("shairport-sync", "-u", "-vv")
	// p := exec.Command("ls", "/usr/local/bin")
	out, err := p.StdoutPipe()
	if err != nil {
		lerr("", err)
	}
	errno, err := p.StderrPipe()
	if err != nil {
		lerr("", err)
	}
	// var er error
	scanner := bufio.NewScanner(out)
	escanner := bufio.NewScanner(errno)
	err = p.Start()
	if err != nil {
		lerr("", err)
	}
	// go func() {
	for scanner.Scan() {
		// Do something with the line here.
		// fmt.Println(scanner.Text())
	}
	// }()
	// go func() {
	for escanner.Scan() {
		// Do something with the line here.
		// er = fmt.Errorf("%s%s", er, escanner.Text())
		fmt.Println(escanner.Text())
	}
	// }()
	if scanner.Err() != nil {
		p.Process.Kill()
		p.Wait()
		lerr("Output Error: %s", scanner.Err())
	}
	if escanner.Err() != nil {
		p.Process.Kill()
		p.Wait()
		lerr("Error err: %s", escanner.Err())
	}
	// p.Process.Kill()
	p.Wait()
	fmt.Println("exiting")
	// t.Logf("%s", out)

}
