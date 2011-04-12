package main

import (
	"net"
	"flag"
	"log"
	"time"

	afp "github.com/davecheney/afp"
)

func main() {
	addr, err := net.ResolveTCPAddr(flag.Args()[0])
	if err != nil {
		log.Fatal(err)
	}
	_, err = afp.DialDSI(addr)
	if err != nil {
		log.Fatal(err)
	}

	<-time.After(120e9)
}
