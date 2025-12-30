package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/sanjanabharath/redis-golang/cmd"
	"github.com/sanjanabharath/redis-golang/configs"
)

func readCommand(c io.ReadWriter) (*cmd.RedisCMD, error) {
	// TODO: Max read in one shot is 512 bytes
	// To allow input > 512 bytes, then repeated read until
	// we get EOF or designated delimiter
	var buf []byte = make([]byte, 512)
	n, err := c.Read(buf[:])
	if err != nil {
		return nil, err
	}

	tokens, err := cmd.DecodeArrayString(buf[:n])
	if err != nil {
		return nil, err
	}

	return &cmd.RedisCMD{
		Cmd:  strings.ToUpper(tokens[0]),
		Args: tokens[1:],
	}, nil
}

func respondError(err error, c io.ReadWriter) {
	c.Write([]byte(fmt.Sprintf("-%s\r\n", err)))
}

func respond(cmd *cmd.RedisCMD, c io.ReadWriter) {
	err := cmd.EvalAndRespond(cmd, c)
	if err != nil {
		respondError(err, c)
	}
}

func RunSyncTCPServer() {
	log.Println("starting a synchronous TCP server on", configs.Host, configs.Port)

	// Start the background cleanup routine for expired keys
	cmd.StartCleanupRoutine()

	var con_clients int = 0

	// listening to the configured host:port
	lsnr, err := net.Listen("tcp", configs.Host+":"+strconv.Itoa(configs.Port))
	if err != nil {
		log.Println("err", err)
		return
	}

	for {
		// blocking call: waiting for the new client to connect
		c, err := lsnr.Accept()
		if err != nil {
			log.Println("err", err)
		}

		// increment the number of concurrent clients
		con_clients += 1

		for {
			// over the socket, continuously read the command and print it out
			cmd, err := readCommand(c)
			if err != nil {
				c.Close()
				con_clients -= 1
				if err == io.EOF {
					break
				}
			}
			respond(cmd, c)
		}
	}
}
