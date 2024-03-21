package tests

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/gitteamer/libconfig"
	"github.com/jasonkolodziej/go-castv2/sps"
	"github.com/jasonkolodziej/go-castv2/sps/parse"
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
	p := exec.Command("shairport-sync", "-u", "-vv")
	// p := exec.Command("ls", "/usr/local/bin")
	out, err := p.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	errno, err := p.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(out)
	escanner := bufio.NewScanner(errno)
	err = p.Start()
	if err != nil {
		t.Fatal(err)
	}
	for scanner.Scan() {
		// Do something with the line here.
		t.Fatal(scanner.Text())
	}
	go func() {
		for escanner.Scan() {
			// Do something with the line here.
			t.Log(escanner.Text())
		}
	}()
	if scanner.Err() != nil {
		p.Process.Kill()
		p.Wait()
		t.Fatalf("Output Error: %s", scanner.Err())
	}
	if escanner.Err() != nil {
		p.Process.Kill()
		p.Wait()
		t.Fatalf("Error err: %s", escanner.Err())
	}
	p.Process.Kill()
	p.Wait()
	// t.Logf("%s", out)

}

// func startup(t *testing.T, useScanner bool) {
// 	file, ferr := os.Open("../tests/shairport-sync.conf")
// 	if ferr != nil {
// 		t.Error(ferr)
// 	}
// 	defer file.Close()
// 	if useScanner {
// 		tmplScanner = bufio.NewScanner(file)
// 		return
// 	}
// 	if ferr = file.Close(); ferr != nil {
// 		t.Error(ferr)
// 	}
// 	reader, ferr = os.ReadFile("../tests/shairport-sync.conf")
// 	if ferr != nil {
// 		t.Error(ferr)
// 	}
// }

func Test_Sps_Parser(t *testing.T) {
	const tmplFile = "shairport-sync.conf2.tmpl"
	confFile, _ := loadTestFile(t, "shairport-syncKitchenSpeaker.conf", false)
	reader, err := io.ReadAll(confFile)
	if err != nil {
		t.Fatal(err)
	}
	defer confFile.Close()

	reading := string(reader)
	kvTempl := parse.KeyValue{}
	kvTempl.SetDelimiters("=", ";", "/ ")

	sections := parse.Parse(&reading, &kvTempl, "{", " =", "};")
	for i, section := range sections {
		t.Logf("Section id: %v, Name: %s, Number of Keys: %v", i, section.Name, len(section.KeyValues))
		for _, kv := range section.KeyValues {
			if !kv.KvIsCommented() {
				t.Logf("Key: %s, found uncommented with value: %v, type of: %s", kv.KeyName, kv.KeysValue, kv.Type())
			}
			if kv.KeyName == "name" {
				if err := kv.SetValue("New Name"); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	parse.WriteOut(sections, "", "newConf.conf")
	t.Log("Done")
}

func Test_PipeFilled(t *testing.T) {
	// out, _, err := sps.SpawnProcessConfig()
	p := exec.Command("shairport-sync", "-vv", "--output", "stdout")
	out, err := p.StdoutPipe() // * io.ReadCloser
	if err != nil {
		t.Fatal(err)
	}
	err = p.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	// var v bool
	peek := bufio.NewReader(out)
	peeker := bufio.NewScanner(peek)
	peeker.Split(bufio.ScanBytes)

	//s :=
	// var bRead = 0
	for peeker.Scan() {
		// for err == nil {
		t.Log(peeker.Text())
		// peeked, err := peek.Peek(4096)
		// le := len(peeked)
		// if err == nil && le == 4096 {
		// 	v = true
		// 	peeker.Text()
		// 	t.Log("Sound")
		// 	t.Log(p)
		// } else if err != nil {
		// 	t.Fatal(err)
		// } else {
		// 	v = false
		// }
		if err = peeker.Err(); err != nil {
			t.Fatal(err)
		}
	}

	t.Log("done ok")
	p.Wait()
	// good := make(chan io.ReadCloser)

	// go func(out io.ReadCloser, retc chan io.ReadCloser) {
	// 	peek := bufio.NewReader(out)
	// 	// peeker := bufio.NewScanner(peek)
	// 	// peeker.Split(bufio.ScanBytes)
	// 	// var bRead = 0
	// 	// for peeker.Scan() {
	// 	for {
	// 		if peeked, err := peek.Peek(1); err != nil && len(peeked) == 1 {
	// 			retc <- out // * there is content in the pipe
	// 		}
	// 		retc <- nil
	// 	}
	// }(out, good)

	// o := <-good

	// if o != nil {
	// 	t.Log("Ok")
	// }

}
