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
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/jasonkolodziej/go-castv2/virtual"
	"github.com/rs/zerolog"
)

var z = zerolog.New(os.Stderr).Level(0).With().Timestamp().Logger()

var fib = fiber.New(fiber.Config{
	Prefork:       true,
	CaseSensitive: true,
	StrictRouting: true,
	ServerHeader:  "Fiber",
	AppName:       "Test App v1.0.1",
	GETOnly:       true,
})

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

/*
shairport-sync -c /etc/shairport-syncKitchenSpeaker.conf | \
ffmpeg -y -re -fflags nobuffer -f s16le -ac 2 -ar 44100 -i pipe:0 -bits_per_raw_sample 8 -f adts ./hlsTest/output.aac | \
./mainn
*/

// func main() {
// 	log.Println("Starting Stream SERVER...")
// 	// fname := flag.String("filename", "file.aac", "path of the audio file")
// 	// flag.Parse()
// 	// file, err := os.Open(*fname)
// 	// if err != nil {

// 	// 	log.Fatal(err)

// 	// }
// 	// var ctn = make(chan *[]byte)
// 	// var ctn []byte
// 	// var err error
// 	// check if there is somethinig to read on STDIN

// 	stat, _ := os.Stdin.Stat()
// 	if (stat.Mode() & os.ModeCharDevice) == 0 {
// 		log.Println("STDIN Ready, scanning...")
// 		// scanner := bufio.NewScanner(os.Stdin)
// 		// scanner.Split(bufio.ScanBytes)
// 		// for scanner.Scan() {
// 		// 	ctn = append(ctn, scanner.Bytes()...)
// 		// }
// 		// go ReadFromStdIn(ctn, os.Stdin)
// 	} else {
// 		log.Fatal("Nothing to read from StdIN")
// 	}

// 	connPool := NewConnectionPool()

// 	go GetStream(connPool, os.Stdin)

// 	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

// 		w.Header().Add("Content-Type", "audio/aac")
// 		w.Header().Add("Connection", "keep-alive")

// 		flusher, ok := w.(http.Flusher)
// 		if !ok {
// 			log.Println("Could not create flusher")
// 		}

// 		connection := &Connection{bufferChannel: make(chan []byte), buffer: make([]byte, BUFFERSIZE)}
// 		connPool.AddConnection(connection)
// 		log.Printf("%s has connected to the audio stream\n", r.Host)

// 		for {

// 			buf := <-connection.bufferChannel
// 			if _, err := w.Write(buf); err != nil {

// 				connPool.DeleteConnection(connection)
// 				log.Printf("%s's connection to the audio stream has been closed\n", r.Host)
// 				return

// 			}
// 			flusher.Flush()
// 			clear(connection.buffer)

// 		}
// 	})

// 	log.Println("Listening on port 8080...")
// 	log.Fatal(http.ListenAndServe(":8080", nil))

// }

// func WaitOnStdIn(connectionPool *ConnectionPool, content io.ReadCloser) {
// 	for {
// 		stat, err := os.Stdin.Stat()
// 		if err != nil {
// 			z.Debug().AnErr("WaitonStdIn", err).Send()
// 			return
// 		}
// 		if (stat.Mode() & os.ModeCharDevice) == 0 {
// 			log.Println("STDIN Ready, scanning...")
// 			break
// 		}
// 	}
// 	go GetStream(connectionPool, content)
// }

func main() {
	connPool := virtual.NewConnectionPool()
	// * File - WORKS
	// f, err := os.Open("./hlsTest/output.aac")
	// if err != nil {
	// 	z.Fatal().AnErr("os.Open", err)
	// 	panic(err)
	// }
	// defer f.Close()

	// stat, err := os.Stdin.Stat()
	// if (stat.Mode() & os.ModeCharDevice) == 0 {
	// 	log.Println("STDIN Ready, scanning...")
	// 	// defer os.Stdin.Close()

	// 	// 	// scanner := bufio.NewScanner(os.Stdin)
	// 	// 	// scanner.Split(bufio.ScanBytes)
	// 	// 	// for scanner.Scan() {
	// 	// 	// 	ctn = append(ctn, scanner.Bytes()...)
	// 	// 	// }
	// 	// 	// go ReadFromStdIn(ctn, os.Stdin)
	// } else if err != nil {
	// 	log.Fatalln(err)
	// } else {
	// 	log.Fatal("Nothing to read from StdIN")
	// }

	go virtual.GetStreamFromReader(connPool, os.Stdin)
	fib.Get("/stream", func(c *fiber.Ctx) error {
		// z.Info().Any("CtxId", c.Context().ID()).Send()
		// z.Info().Any("headers", c.Context().Request.String()).Send()
		c.Context().SetContentType("audio/aac")
		c.Set(fiber.HeaderConnection, fiber.HeaderKeepAlive)
		var connection = connPool.HasConnectionWithId(c.Context().RemoteIP())
		// connection, ok := c.Context().Value("connection").(*virtual.Connection)
		if connection == nil {
			// z.Warn().Msg("Assembling a new connection!")
			connection = virtual.NewConnectionWithId(c.Context().RemoteIP())
			connPool.AddConnection(connection)
		} else {
			z.Warn().Msg("Found a existing connection")
		}
		z.Info().Msgf("%s has connected to the audio stream\n", c.Context().RemoteIP().String())
		c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
			for {
				buf := <-connection.BufferCh()
				if _, err := w.Write(buf); err != nil {
					connPool.DeleteConnection(connection)
					z.Info().Err(err).Msgf("connection to the audio stream has been closed\n")
					return
				}
				if err := w.Flush(); err != nil {
					z.Warn().Err(err).Msg("calling writer.Flush")
				}
				connection.ClearBuffer()
			}
		})
		return nil
	})

	z.Fatal().Err(fib.Listen(":8080"))

}
