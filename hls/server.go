package hls

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasonkolodziej/go-castv2"
	"github.com/jasonkolodziej/go-castv2/async"
	logger "github.com/sirupsen/logrus"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	logger.SetFormatter(&logger.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logger.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logger.SetLevel(logger.DebugLevel)
}

type AsyncHandler func([]byte) error
type SyncHandler func([]byte) ([]byte, error)

var server *async.Server

var linform = logger.Infof
var ldebug = logger.Debugf
var lwarn = logger.Warnf
var lerr = logger.Errorf

func StartServer(port uint) {
	// create new http server with max async 20 goroutines
	server = async.NewServer("mytest", int(port)).SetAsyncNum(20) // * Should listen on port specificied across all connected Addresses
	// handler sync http request
	// server.HandlerRequst("POST", "/sync", syncDemo)

	// handler async http request
	server.HandlerAsyncRequst("POST", "/async", asyncDemo)

	go func() {
		if err := server.Start(); err != nil {
			lerr("server failed: %v", err)
		}
	}()
}

// simple handler for async request
// return task info in response data when performing request handler asynchronously
var asyncDemo AsyncHandler = func(jsonIn []byte) error {
	time.Sleep(5 * time.Second)
	linform("[asyncDemo] jsonIn: %v", string(jsonIn[:]))

	return nil
}

var syncDemo SyncHandler = func(jsonIn []byte) ([]byte, error) {
	time.Sleep(5 * time.Second)
	linform("[asyncDemo] jsonIn: %v", string(jsonIn[:]))

	return []byte("ok"), nil
}

// Will apply paths for extUrl if needed
func AssignAsyncHandlerPath(cb AsyncHandler, method, baseUrl, extUrl string) {
	var url = baseUrl
	if extUrl != "" {
		url += "/" + extUrl
	}
	server.HandlerAsyncRequst(method, url, cb)
}

func AssignSyncHandlerPath(cb SyncHandler, method, baseUrl, extUrl string) {
	var url = baseUrl
	if extUrl != "" {
		url += "/" + extUrl
	}
	server.HandlerRequst(method, url, cb)
}

func CleanStopServer() {
	// you can stop server using Stop() method which could await completion for all requests
	// finishing off some extra-works by a system signal is recommended
	EndChannel := make(chan os.Signal, 1) //? EndChannel should be buffered?
	signal.Notify(EndChannel, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	select { // ! select maybe meaningless with one case
	case output := <-EndChannel:
		linform("end http server process by: %s", output)
		server.Stop()
	}
	close(EndChannel)
}

func GenerateHandleForDevice(d *castv2.Device, async bool) {
	if async {
		AssignAsyncHandlerPath(asyncDemo, "GET", d.Info.Id.String(), "stream.flac")
	} else {
		AssignSyncHandlerPath(syncDemo, "GET", d.Info.Id.String(), "stream.flac")
	}
	// server.SetTaskManager()
}
