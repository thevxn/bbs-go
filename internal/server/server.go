package server

import (
	"fmt"
	"net"
	"sync"
)

// Server struct holds server configurations and handlers.
type Server struct {
	Host    string
	Port    int
	Handler *Handler
	wg      *sync.WaitGroup
}

// NewServer initializes and returns a new Server instance.
func NewServer(address string) *Server {
	host := "0.0.0.0"
	port := 2323

	// Initialize the server
	return &Server{
		Host:    host,
		Port:    port,
		wg:      &sync.WaitGroup{},
		Handler: &Handler{},
	}
}

// Run starts the server and listens for incoming connections.
func (s *Server) Run() {
	listenAddress := fmt.Sprintf("%s:%d", s.Host, s.Port)
	ln, err := net.Listen("tcp", listenAddress)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
	defer ln.Close()

	fmt.Printf("Server is running on %s\n", listenAddress)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		// Handle the incoming connection with the Handler
		h := &Handler{
			conn: conn,
			wg:   s.wg,
		}
		s.wg.Add(1)
		go h.Handle()
	}
}
