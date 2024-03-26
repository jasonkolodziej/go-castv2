package virtual

// ? https://medium.com/@icelain/a-guide-to-building-a-realtime-http-audio-streaming-server-in-go-24e78cf1aa2c
// ? https://ice.lqorg.com/blog/realtime-http-audio-streaming-server-in-go
import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net"
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
	Id            *net.IP
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

func (cp *ConnectionPool) HasConnection(connection *Connection) bool {

	defer cp.mu.Unlock()
	cp.mu.Lock()

	_, ok := cp.ConnectionMap[connection]
	return ok
}

func (cp *ConnectionPool) HasConnectionWithId(netIp net.IP) *Connection {

	defer cp.mu.Unlock()
	cp.mu.Lock()
	for c := range cp.ConnectionMap {
		if c.Id == &netIp {
			z.Debug().Any("HasConnectionWithId", netIp).Msg("a matching connection was found")
			return c
		}
	}
	return nil
}

func NewConnectionWithId(connId net.IP) *Connection {
	return &Connection{bufferChannel: make(chan []byte), buffer: make([]byte, BUFFERSIZE), Id: &connId}
}

func NewConnection() *Connection {
	return &Connection{bufferChannel: make(chan []byte), buffer: make([]byte, BUFFERSIZE), Id: nil}
}

func (c *Connection) BufferCh() <-chan []byte {
	return c.bufferChannel

}

func (c *Connection) StreamWriter(connPool *ConnectionPool) func(w *bufio.Writer) {
	return func(w *bufio.Writer) {
		for {
			buf := <-c.BufferCh()
			if _, err := w.Write(buf); err != nil {
				connPool.DeleteConnection(c)
				z.Info().Err(err).Msgf("connection to the audio stream has been closed\n")
				return
			}
			if err := w.Flush(); err != nil {
				z.Warn().Err(err).Msg("calling writer.Flush")
			}
			c.ClearBuffer()
		}
	}
}

func (c *Connection) ClearBuffer() {
	// z.Info().Msgf("Clearing Connection.Buffer")
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
			z.Debug().Any("ConnectionPool.Broadcast", "broadcasted buffer")
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
				z.Debug().AnErr("GetStream", err)
				ticker.Stop()
				break

			} else if err != nil {
				z.Debug().AnErr("GetStream", err)
			}

			connectionPool.Broadcast(buffer)

		}

	}

}

func GetStreamFromReader(connectionPool *ConnectionPool, content io.ReadCloser) {
	defer content.Close()
	buffer := make([]byte, BUFFERSIZE)

	for {
		// clear() is a new builtin function introduced in go 1.21. Just reinitialize the buffer if on a lower version.
		clear(buffer)
		tempfile := bufio.NewReader((content))
		ticker := time.NewTicker(time.Millisecond * DELAY)

		for range ticker.C {

			n, err := tempfile.Read(buffer)
			z.Debug().Msgf("Read %d bytes", n)

			if err == io.EOF {

				ticker.Stop()
				break

			} else if err != nil {
				z.Err(err).Msg("GetStreamFromReader")
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
