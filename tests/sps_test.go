package tests

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/gitteamer/libconfig"
	"github.com/jasonkolodziej/go-castv2/sps"
)

const configFile = "data/example.conf"

var emptyKeyArr = []string{}

func Test_FileParser(t *testing.T) {
	pwd, _ := os.Getwd()
	t.Log(pwd)
	v := sps.FileParser(pwd + "/" + configFile)
	if !v.Exists("general", "airplay_device_id") {
		t.Error("airplay_device_id key does not exist")
	}
	val := v.Get("general").Get("airplay_device_id")
	t.Logf("%s", val)
	gen := v.Get("general")
	air := gen.Get("airplay_device_id")
	t.Logf("current value: %s, type: %s", air, air.Type().String())
	// libconfig.GetHex()
	gen.Set("airplay_device_id", libconfig.MustParse(`=0xF4L;`).Get(""))
	// gen.Set("airplay_device_id", libconfig.MustParse(`=0xF4L;`).Get(""))
	// val = gen.Get("airplay_device_id")
	t.Logf("set to: %s", gen)
}

func readPipe(p io.ReadCloser, t *testing.T) {
	reader := bufio.NewReader(p)
	line, err := reader.ReadString('\n')
	for err == nil {
		t.Log(line)
		line, err = reader.ReadString('\n')
	}
}

// ? https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/
func Test_SpawnProcess(t *testing.T) {
	p := exec.Command("shairport-sync", "-h")
	out, _ := p.CombinedOutput()
	t.Logf("%s", out)

}
