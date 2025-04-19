package server

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"go.vxn.dev/bbs-go/internal/config"
)

const prompt = "> "
const crlf = "\r\n"
const shutdownMessage = "Shutdown" + crlf
const byeMessage = "*** Bye" + crlf
const helpMessage = "*** Commands" + crlf +
	"    exit --- quit the session" + crlf + crlf
const invalidCommandMessage = "*** Invalid command, try 'help'" + crlf

// stripTelnetNegotiation removes Telnet command sequences from incoming data.
// These sequences begin with the IAC (Interpret As Command) byte (decimal 255),
// followed by one or more bytes indicating Telnet negotiation options (e.g., WILL, DO, etc.).
// See RFC 854 for details: https://datatracker.ietf.org/doc/html/rfc854
func stripTelnetNegotiation(data []byte) []byte {
	result := []byte{}
	i := 0
	for i < len(data) {
		if data[i] == 255 { // IAC
			// Telnet command structure is typically IAC <command> <option>
			// So skip the next two bytes (if available)
			if i+2 < len(data) {
				i += 3
			} else {
				break
			}
		} else {
			result = append(result, data[i])
			i++
		}
	}
	return result
}

type Handler struct {
	//ctx context.Context
	done   chan struct{}
	conn   net.Conn
	output io.Writer
	wg     *sync.WaitGroup
}

// write writes data to the Handler's client connection. Any error returned is logged.
func (h *Handler) write(data string) {
	if _, err := h.conn.Write([]byte(data)); err != nil {
		h.debugf("Write error: %s", err.Error())
	}
}

func (h *Handler) Handle() {
	defer h.wg.Done()

	h.logf("< Incoming connection: %s", h.conn.RemoteAddr().String())
	h.write(WelcomeMessage)
	h.write(prompt)

	defer h.conn.Close()

	pktChan := make(chan string)
	errChan := make(chan error)

	h.wg.Add(2)
	go h.route(pktChan, errChan)
	go h.read(pktChan, errChan)

	<-errChan

	h.logf("> Connection closed: %s", h.conn.RemoteAddr().String())
}

func (h *Handler) read(pktChan chan string, errChan chan error) {
	defer h.wg.Done()
	defer h.debugf("Handler: read closed")

	// Run the read loop.
	for {

		select {
		case <-h.done:
			h.write(shutdownMessage)
			h.logf("> Connection closed (daemon's shutdown): %s", h.conn.RemoteAddr().String())

			if errChan != nil {
				errChan <- nil
				close(errChan)
			}

			return

		case <-errChan:
			return

		default:
			// Prepare packet's byte allocation.
			tmp := make([]byte, 512)
			buf := strings.Builder{}

			// Read bytes from the remote conterpart.
			for {
				n, err := h.conn.Read(tmp)

				// Handle error from the read loop.
				if err != nil {
					switch err {
					case io.EOF:
						// End-of-file
						h.debugf("< EOF")

					case err.(net.Error):
						h.write("Too slow\n")
						h.debugf("> Connection closed (read timeout): %s", h.conn.RemoteAddr().String())
						return

					default:
						h.debugf("< Unexpected read error: %s", err.Error())
						return
					}
				}

				for i := range n {
					b := tmp[i]

					// CR or LF indicates end of line
					if b == '\n' {
						line := stripTelnetNegotiation([]byte(buf.String()))
						if pktChan != nil && len(line) > 0 {
							pktChan <- string(line)
						}
						buf.Reset()
					} else if b != '\r' {
						buf.WriteByte(b)
					}
				}
			}
		}
	}
}

func (h *Handler) route(pktChan chan string, errChan chan error) {
	defer h.wg.Done()
	defer h.debugf("Handler: route closed")

	defer func() {
		if errChan != nil {
			errChan <- nil
			close(errChan)
		}
	}()

	for {
		select {
		case <-h.done:
			h.write(shutdownMessage)
			h.logf("> Connection closed (daemon's shutdown): %s", h.conn.RemoteAddr().String())
			return

		case <-errChan:
			return

		default:
			// Preprocess the string for switch.
			pkt := <-pktChan
			// h.debugf("pkt: %s", pkt)
			// h.debugf("pkt: %v", []byte(pkt))
			parts := strings.Split(pkt, "\r\n")

			switch strings.TrimSpace(parts[0]) {
			case "":

			case "exit":
				h.write(byeMessage)
				return

			case "help":
				h.write(helpMessage)

			default:
				h.debugf("Invalid command")
				h.write(invalidCommandMessage)
			}

			h.write(prompt)
		}
	}
}

// Common debug info logging wrapper function.
func (h *Handler) debugf(format string, args ...any) {
	if h.output == nil || !config.Debug {
		return
	}

	// Prepend the debug info prefix.
	f := fmt.Sprintf("(dbg) %s\n", format)

	// Write to the output according to running configuration.
	fmt.Fprintf(h.output, f, args...)
}

// Common basic logging wrapper function.
func (h *Handler) logf(format string, args ...any) {
	if h.output == nil {
		return
	}

	// Write to the output according to running configuration.
	fmt.Fprintf(h.output, format+"\n", args...)
}
