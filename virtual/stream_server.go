package virtual

// ? https://medium.com/@icelain/a-guide-to-building-a-realtime-http-audio-streaming-server-in-go-24e78cf1aa2c
// ? https://ice.lqorg.com/blog/realtime-http-audio-streaming-server-in-go
import (
	"bytes"
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

func GetStream(connectionPool *ConnectionPool, content []byte) {

	buffer := make([]byte, BUFFERSIZE)

	for {

		// clear() is a new builtin function introduced in go 1.21. Just reinitialize the buffer if on a lower version.
		clear(buffer)
		tempfile := bytes.NewReader(content)
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

func main() {

	// fname := flag.String("filename", "file.aac", "path of the audio file")
	// flag.Parse()
	// file, err := os.Open(*fname)
	// if err != nil {

	// 	log.Fatal(err)

	// }

	ctn, err := io.ReadAll(os.Stdin)
	if err != nil {

		log.Fatal(err)

	}

	connPool := NewConnectionPool()

	go GetStream(connPool, ctn)

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
