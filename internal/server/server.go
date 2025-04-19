package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"go.vxn.dev/bbs-go/internal/config"
)

const (
	ConnectionTimeout time.Duration = 5 * time.Second
	ShutdownTimeout   time.Duration = 5 * time.Second
)

type Server struct {
	ctx          context.Context
	done         chan struct{}
	once         sync.Once
	outputTarget io.Writer
	wg           sync.WaitGroup
}

func NewServer(ctx context.Context) *Server {
	return &Server{
		ctx: ctx,
	}
}

// Run acts like the Server's API function to watch main's context cancelation to autostart the shutdown process.
func (s *Server) Run() error {
	s.init()

	for {
		select {
		// A signal notifying that the Shutdown procedure has been initiated.
		case <-s.done:
			return ErrShutdownStarted

		// A signal from the parent task supervising the daemon (via the main() function),
		// meaning the Shutdown procedure is to be started now.
		case <-s.ctx.Done():
			s.logf("\n--- App's context canceled: starting a graceful shutdown cleanup...")

			// Create a new shutdown context with timeout to close the server by force after deadline.
			sctx, scancel := context.WithTimeout(context.Background(), ShutdownTimeout)
			defer scancel()

			if err := s.Shutdown(sctx); err != nil {
				s.logf("Shutdown error: %s", err.Error())
				return err
			}

			if s.ctx.Err() != context.Canceled {
				return s.ctx.Err()
			}

			return nil
		}
	}
}

// Shutdown takes care of the system resources closing and cleanup.
func (s *Server) Shutdown(sctx context.Context) (err error) {
	s.init()

	defer func() {
		if r := recover(); r != nil {
			err = ErrShutdownStarted
		}
	}()

	// This will panic when closed by any previous function invokation => to be recovered.
	close(s.done)

	// This code section should only be access once in app's lifetime!
	s.logf("--- Shutdown invoked...")
	shch := make(chan struct{})

	go func() {
		// Wait for all goroutines to exit gracefully.
		s.wg.Wait()
		close(shch)
	}()

	// Wait for the context cancelation, or a timeout.
	select {
	case <-shch:
		s.logf("Graceful cleanup done.")

	case <-sctx.Done():
		if sctx.Err() != context.DeadlineExceeded {
			err = sctx.Err()
			return
		}

		return
	}

	return
}

// init gets called from any exported function to ensure that the Server instance
// is initialized before a Shutdown routine is started.
func (s *Server) init() {
	if s.ctx == nil {
		panic("Server's context missing, use NewServer(...) to initialize a new Server instance properly")
	}

	s.once.Do(func() {
		if s.outputTarget == nil {
			s.outputTarget = os.Stdout
		}

		s.logf("*** Initializing new Server instance...")

		s.done = make(chan struct{})

		// Run the TCP listener.
		s.wg.Add(1)
		go s.listen()

		s.logf("Initialization complete.")
	})
}

//
//  Logging methods.
//

// Common debug info logging wrapper function.
func (s *Server) debugf(format string, args ...any) {
	if s.outputTarget == nil || !config.Debug {
		return
	}

	// Prepend the debug info prefix.
	f := fmt.Sprintf("(dbg) %s\n", format)

	// Write to the output according to running configuration.
	fmt.Fprintf(s.outputTarget, f, args...)
}

// Common basic logging wrapper function.
func (s *Server) logf(format string, args ...any) {
	if s.outputTarget == nil {
		return
	}

	// Write to the output according to running configuration.
	fmt.Fprintf(s.outputTarget, format+"\n", args...)
}

// Main TCP listener.
func (s *Server) listen() {
	defer s.wg.Done()

	// Start listening on a port from daemon's running configuration.
	l, err := net.Listen("tcp4", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		panic(err)
	}

	defer func() {
		s.logf("TCP listener closed.")
		l.Close()
	}()

	// Ensure the welcome message contains carriage return + line feed
	// Needed to display properly on Windows terminals
	WelcomeMessage = strings.ReplaceAll(WelcomeMessage, "\n", "\r\n")

	s.logf("%s", WelcomeMessage)
	s.logf("*** Listening on port TCP/%d...", config.Port)

	go func() {
		for {
			// Wait for a new connection.
			conn, err := l.Accept()
			if err != nil {
				// Check the main context for an error (may be already canceled).
				if s.ctx.Err() != nil {
					return
				}
				s.logf("Accept error: %s", err.Error())
				break
			}

			// Set the conn deadline.
			//conn.SetDeadline(time.Now().Add(ConnectionTimeout))

			// Handle the connection in a new goroutine.
			// --> internal/server/handler.go
			handler := &Handler{
				done:   s.done,
				conn:   conn,
				output: s.outputTarget,
				wg:     &s.wg,
			}

			s.wg.Add(1)
			go handler.Handle()
		}
	}()

	// Wait until the shutdown is invoked.
	<-s.done
}
