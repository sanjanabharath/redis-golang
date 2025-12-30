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

func respond(cd *cmd.RedisCMD, c io.ReadWriter) {
	err := cmd.EvalAndRespond(cd, c)
	if err != nil {
		respondError(err, c)
	}
}

func RunSyncTCPServer() {
	log.Println("starting a synchronous TCP server on", configs.Host, configs.Port)

	cmd.StartCleanupRoutine()

	var con_clients int = 0

	lsnr, err := net.Listen("tcp", configs.Host+":"+strconv.FormatInt(configs.Port, 10))
	if err != nil {
		log.Println("err", err)
		return
	}

	for {
		// wait for a client to connect
		c, err := lsnr.Accept()
		if err != nil {
			log.Println("err", err)
		}

		// increment the connection count
		con_clients += 1

		for {
			// obtain the command from the client
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
