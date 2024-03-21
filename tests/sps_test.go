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
	// sections := parse.SplitUpSections(&reading, "};", &kvTempl)
	// sectionNameDelimiter := " ="
	// for i, section := range sections {
	// 	// t.Logf("Section index: %q\n", i)
	// 	t.Log("Call FindBeginningOfSection")
	// 	sDescription, sectionContent := section.FindBeginningOfSection("{", &sectionNameDelimiter)
	// 	t.Log("Call HandleSection")
	// 	section.HandleSection(sDescription, sectionContent[1], "")
	// 	t.Logf("Section idx: %v, Name: %s, contains %v keys.", i, section.Name, len(section.KeyValues))
	// 	for _, kv := range section.KeyValues {
	// 		if !kv.KvIsCommented() {
	// 			t.Logf("Key: %s, found uncommented with value: %v", kv.KeyName, kv.KeysValue)
	// 		}
	// 	}
	// 	// section.HandleSection(sDescription, sectionContent[1], "", "=", ";")
	// }

	t.Log("Done")
}
