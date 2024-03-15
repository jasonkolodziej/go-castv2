package hls

import (
	"os"

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

// var server *async.Server

var linform = logger.Infof
var ldebug = logger.Debugf
var lwarn = logger.Warnf
var lerr = logger.Errorf
