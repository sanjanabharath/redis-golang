package main

import (
	"flag"
	"log"

	"github.com/sanjanabharath/redis-golang/configs"
	"github.com/sanjanabharath/redis-golang/server"
)

func setupFlags() {
	flag.StringVar(&configs.Host, "host", "0.0.0.0", "host for the redis server")
	flag.Int64Var(&configs.Port, "port", int64(6379), "port for the redis server")
	flag.Parse()
}

func main() {
	setupFlags()
	log.Println("starting the redis server..")
	server.RunAsyncTCPServer()
}