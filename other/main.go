package main

// import (
// 	"bufio"
// 	"fmt"
// 	"os"
// 	"os/exec"

// 	logger "github.com/sirupsen/logrus"
// )

// func init() {
// 	// Log as JSON instead of the default ASCII formatter.
// 	// logger.SetFormatter(&logger.JSONFormatter{})

// 	// Output to stdout instead of the default stderr
// 	// Can be any io.Writer, see below for File example
// 	logger.SetOutput(os.Stdout)

// 	// Only log the warning severity or above.
// 	logger.SetLevel(logger.ErrorLevel)

// }

// var linform = logger.Infof
// var ldebug = logger.Debugf
// var lwarn = logger.Warnf
// var lerr = logger.Errorf

// func main() {
// 	p := exec.Command("shairport-sync", "-vv")
// 	// p := exec.Command("ls", "/usr/local/bin")
// 	out, err := p.StdoutPipe()
// 	if err != nil {
// 		lerr("", err)
// 	}
// 	errno, err := p.StderrPipe()
// 	if err != nil {
// 		lerr("", err)
// 	}
// 	// var er error
// 	scanner := bufio.NewScanner(out)
// 	escanner := bufio.NewScanner(errno)
// 	err = p.Start()
// 	if err != nil {
// 		lerr("", err)
// 	}
// 	// go func() {
// 	// for scanner.Scan() {
// 	// 	// Do something with the line here.
// 	// 	fmt.Println(scanner.Text())
// 	// }
// 	// }()
// 	go func() {
// 		for escanner.Scan() {
// 			// Do something with the line here.
// 			// er = fmt.Errorf("%s%s", er, escanner.Text())
// 			fmt.Println(escanner.Text())
// 		}
// 	}()
// 	if scanner.Err() != nil {
// 		p.Process.Kill()
// 		p.Wait()
// 		lerr("Output Error: %s", scanner.Err())
// 	}
// 	if escanner.Err() != nil {
// 		p.Process.Kill()
// 		p.Wait()
// 		lerr("Error err: %s", escanner.Err())
// 	}
// 	// p.Process.Kill()
// 	p.Wait()
// 	fmt.Println("exiting")
// 	// t.Logf("%s", out)

// }

// ? https://medium.com/@icelain/a-guide-to-building-a-realtime-http-audio-streaming-server-in-go-24e78cf1aa2c
// ? https://ice.lqorg.com/blog/realtime-http-audio-streaming-server-in-go
import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	BUFFERSIZE = 8192

	//formula for delay = track_duration * buffer_size / aac_file_size
	DELAY = 150
)

type Connection struct {
	bufferChannel chan []byte
	buffer        []byte
}

type ConnectionPool struct {
	ConnectionMap map[*Connection]struct{}
	mu            sync.Mutex
}

func (cp *ConnectionPool) AddConnection(connection *Connection) {

	defer cp.mu.Unlock()
	cp.mu.Lock()

	cp.ConnectionMap[connection] = struct{}{}

}

func NewConnection() *Connection {
	return &Connection{bufferChannel: make(chan []byte), buffer: make([]byte, BUFFERSIZE)}

}

func (c *Connection) BufferCh() <-chan []byte {
	return c.bufferChannel

}

func (c *Connection) ClearBuffer() {
	clear(c.buffer)
}

func (cp *ConnectionPool) DeleteConnection(connection *Connection) {

	defer cp.mu.Unlock()
	cp.mu.Lock()

	delete(cp.ConnectionMap, connection)

}

func (cp *ConnectionPool) Broadcast(buffer []byte) {

	defer cp.mu.Unlock()
	cp.mu.Lock()

	for connection := range cp.ConnectionMap {

		copy(connection.buffer, buffer)

		select {

		case connection.bufferChannel <- connection.buffer:

		default:

		}

	}

}

func NewConnectionPool() *ConnectionPool {

	connectionMap := make(map[*Connection]struct{})
	return &ConnectionPool{ConnectionMap: connectionMap}

}

func GetStream(connectionPool *ConnectionPool, content io.Reader) {

	buffer := make([]byte, BUFFERSIZE)

	for {
		// clear() is a new builtin function introduced in go 1.21. Just reinitialize the buffer if on a lower version.
		clear(buffer)
		tempfile := bufio.NewReader(content) // bytes.NewReader(content)
		ticker := time.NewTicker(time.Millisecond * DELAY)

		for range ticker.C {

			_, err := tempfile.Read(buffer)

			if err == io.EOF {

				ticker.Stop()
				break

			}

			connectionPool.Broadcast(buffer)

		}
	}

}

func ReadFromStdIn(ctn chan<- *[]byte, r io.Reader) {
	// func ReadAll(r Reader) ([]byte, error) {
	b := make([]byte, 0, 512)
	ctn <- &b
	for {
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				log.Println("ReadFromStdIn: read to EOF")
				err = nil
			}
			if err != nil {
				log.Fatalf("ReadFromStdIn: %s", err.Error())
			}
			break
		}
		// Send before adjusting -- should we rest?
		if len(b) == cap(b) {
			log.Println("ReadFromStdIn: adjusting capacity")
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
	}
}

// ! ffmpeg -y -re -fflags nobuffer -f s16le -ac 2 -ar 44100 -i pipe:0 -f adts pipe:

func main() {
	log.Println("Starting Stream SERVER...")
	// fname := flag.String("filename", "file.aac", "path of the audio file")
	// flag.Parse()
	// file, err := os.Open(*fname)
	// if err != nil {

	// 	log.Fatal(err)

	// }
	// var ctn = make(chan *[]byte)
	// var ctn []byte
	// var err error
	// check if there is somethinig to read on STDIN

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		log.Println("STDIN Ready, scanning...")
		// scanner := bufio.NewScanner(os.Stdin)
		// scanner.Split(bufio.ScanBytes)
		// for scanner.Scan() {
		// 	ctn = append(ctn, scanner.Bytes()...)
		// }
		// go ReadFromStdIn(ctn, os.Stdin)
	} else {
		log.Fatal("Nothing to read from StdIN")
	}

	connPool := NewConnectionPool()

	go GetStream(connPool, os.Stdin)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Content-Type", "audio/aac")
		w.Header().Add("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			log.Println("Could not create flusher")
		}

		connection := &Connection{bufferChannel: make(chan []byte), buffer: make([]byte, BUFFERSIZE)}
		connPool.AddConnection(connection)
		log.Printf("%s has connected to the audio stream\n", r.Host)

		for {

			buf := <-connection.bufferChannel
			if _, err := w.Write(buf); err != nil {

				connPool.DeleteConnection(connection)
				log.Printf("%s's connection to the audio stream has been closed\n", r.Host)
				return

			}
			flusher.Flush()
			clear(connection.buffer)

		}
	})

	log.Println("Listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
