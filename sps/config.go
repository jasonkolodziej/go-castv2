package sps

import (
	"log"

	"github.com/gitteamer/libconfig"
)

var p libconfig.Parser

func FileParser(optFilepath string) *libconfig.Value {
	v, err := p.ParseFile(optFilepath)
	if err != nil {
		log.Fatal(err)
	}
	return v
}
