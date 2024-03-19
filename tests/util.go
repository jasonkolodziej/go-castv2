package tests

//go:+doc
import (
	"encoding/json"
	"os"
	"testing"

	"github.com/go-audio/chunk"
)

const remoteSoundFile = "https://samplelib.com/lib/preview/mp3/sample-12s.mp3"

func printJson(t *testing.T, val any) {
	b, err := json.Marshal(val)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", b)
}

func loadTestSound(t *testing.T, filename string) (f *os.File, size int64) {
	pwd, _ := os.Getwd()
	t.Log(pwd)
	f, err := os.Open(pwd + "/data/" + filename)
	if err != nil {
		t.Error(err)
	}
	fInfo, _ := f.Stat()
	size = fInfo.Size()
	return
}

func FileIntoBuffer(t *testing.T, f *os.File) *chunk.Reader {
	fInfo, err := f.Stat()
	if err != nil {
		t.Error(err)
	}
	// buf := make([]byte, fSize) // * make a new buffer with a fixed size
	//_, err = f.Read(buf) // * file f will conduct a read Op up to the side of the buffer
	// if err != nil {
	// 	t.Error(err)
	// }
	// w, err := os.Create() // * used to generate a writer
	return &chunk.Reader{Size: int(fInfo.Size()), R: f}
	// aBuffer := audio.Buffer(reader)
}

func loadTestFile(t *testing.T, filename string, useScanner bool) (f *os.File, size int64) {
	pwd, _ := os.Getwd()
	t.Log(pwd)
	f, err := os.Open(pwd + "/data/" + filename)
	if err != nil {
		t.Error(err)
	}
	fInfo, _ := f.Stat()
	size = fInfo.Size()
	return
}
