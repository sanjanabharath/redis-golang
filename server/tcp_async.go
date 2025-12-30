package server

import (
	"log"
	"net"
	"syscall"

	"github.com/sanjanabharath/redis-golang/cmd"
	"github.com/sanjanabharath/redis-golang/configs"
)

var con_clients int = 0

func RunAsyncTCPServer() error {
	log.Println("starting an asynchronous TCP server on", configs.Host, configs.Port, "(using kqueue)")

	// Start the background cleanup routine for expired keys
	cmd.StartCleanupRoutine()

	max_clients := 20000

	// Create Kevent array to hold events
	var events []syscall.Kevent_t = make([]syscall.Kevent_t, max_clients)

	// Create a socket
	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(serverFD)

	// Set the Socket to operate in a non-blocking mode
	if err = syscall.SetNonblock(serverFD, true); err != nil {
		return err
	}

	// Set SO_REUSEADDR to allow reusing the address
	if err = syscall.SetsockoptInt(serverFD, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}

	// Bind the IP and the port
	ip4 := net.ParseIP(configs.Host)
	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: int(configs.Port),
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		return err
	}

	// Start listening
	if err = syscall.Listen(serverFD, max_clients); err != nil {
		return err
	}

	// AsyncIO starts here!!
	// Create kqueue instance
	kqueueFD, err := syscall.Kqueue()
	if err != nil {
		log.Fatal(err)
	}
	defer syscall.Close(kqueueFD)

	// Register the server socket with kqueue for read events
	var serverEvent syscall.Kevent_t
	syscall.SetKevent(&serverEvent, serverFD, syscall.EVFILT_READ, syscall.EV_ADD)

	if _, err := syscall.Kevent(kqueueFD, []syscall.Kevent_t{serverEvent}, nil, nil); err != nil {
		return err
	}

	for {
		// Wait for events
		nevents, err := syscall.Kevent(kqueueFD, nil, events, nil)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			log.Println("Kevent error:", err)
			continue
		}

		for i := 0; i < nevents; i++ {
			eventFD := int(events[i].Ident)

			// Check for errors
			if events[i].Flags&syscall.EV_ERROR != 0 {
				log.Printf("Error on FD %d: %v", eventFD, syscall.Errno(events[i].Data))
				continue
			}

			// Check for EOF (connection closed)
			if events[i].Flags&syscall.EV_EOF != 0 {
				log.Printf("Connection closed on FD: %d", eventFD)
				if eventFD != serverFD {
					syscall.Close(eventFD)
					con_clients--
				}
				continue
			}

			// If the socket server itself is ready for an IO
			if eventFD == serverFD {
				// Accept the incoming connection from a client
				fd, _, err := syscall.Accept(serverFD)
				if err != nil {
					log.Println("Accept error:", err)
					continue
				}

				// Increase the number of concurrent clients count
				con_clients++
				log.Printf("New client connected. Total clients: %d", con_clients)

				// Set non-blocking
				syscall.SetNonblock(fd, true)

				// Register this new client socket with kqueue
				var clientEvent syscall.Kevent_t
				syscall.SetKevent(&clientEvent, fd, syscall.EVFILT_READ, syscall.EV_ADD)

				if _, err := syscall.Kevent(kqueueFD, []syscall.Kevent_t{clientEvent}, nil, nil); err != nil {
					log.Printf("Failed to register client FD %d: %v", fd, err)
					syscall.Close(fd)
					con_clients--
				}
			} else {
				// Handle client data
				log.Printf("Client data ready on FD: %d", eventFD)
				comm := cmd.FDComm{Fd: eventFD}
				command, err := readCommand(comm)
				if err != nil {
					log.Printf("Client disconnected or error reading command: %v", err)
					syscall.Close(eventFD)
					con_clients--
					continue
				}
				log.Printf("Received command: %s %v", command.Cmd, command.Args)
				respond(command, comm)
			}
		}
	}
}
