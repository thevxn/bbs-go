package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"bbs-go/internal/config"
)

type Message struct {
	User      string
	Timestamp string
	Content   string
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	messageStore []Message
	messageMutex sync.Mutex
	userDB       = "users.json"
	users        = map[string]string{}
	usersMutex   sync.Mutex
	messagesFile = "messages.txt"
	motdFile     = "motd.txt"
)

type Handler struct {
	conn   net.Conn
	output io.Writer
	wg     *sync.WaitGroup
	done   chan struct{}
	user   string
}

func (h *Handler) Handle() {
	defer h.wg.Done()
	defer h.conn.Close()

	h.done = make(chan struct{})
	h.loadUsers()
	h.loadMessages()

	h.logf("< Incoming connection: %s", h.conn.RemoteAddr().String())

	h.showMOTD()
	h.authenticate()

	h.conn.Write([]byte("> "))
	pktChan := make(chan string)
	errChan := make(chan error)

	h.wg.Add(2)
	go h.route(pktChan, errChan)
	go h.read(pktChan, errChan)

	<-errChan
	h.logf("> Connection closed: %s", h.conn.RemoteAddr().String())
}

func (h *Handler) loadUsers() {
	data, err := os.ReadFile(userDB)
	if err == nil {
		json.Unmarshal(data, &users)
	}
}

func (h *Handler) saveUsers() {
	usersMutex.Lock()
	defer usersMutex.Unlock()

	data, _ := json.MarshalIndent(users, "", "  ")
	_ = os.WriteFile(userDB, data, 0644)
}

func (h *Handler) authenticate() {
	h.conn.Write([]byte("Enter username (or type 'register'): "))
	username := h.readLine()
	if username == "register" {
		h.conn.Write([]byte("Choose a username: "))
		username = h.readLine()
		h.conn.Write([]byte("Choose a password: "))
		password := h.readLine()

		usersMutex.Lock()
		users[username] = password
		usersMutex.Unlock()
		h.saveUsers()
		h.conn.Write([]byte("Registration successful.\n"))
	} else {
		h.conn.Write([]byte("Enter password: "))
		password := h.readLine()

		if stored, ok := users[username]; !ok || stored != password {
			h.conn.Write([]byte("Login failed. Goodbye.\n"))
			h.conn.Close()
			return
		}
		h.conn.Write([]byte(fmt.Sprintf("Welcome back, %s!\n", username)))
	}

	h.user = username
}

func (h *Handler) showMOTD() {
	motd, err := os.ReadFile(motdFile)
	if err == nil {
		h.conn.Write(motd)
		h.conn.Write([]byte("\n"))
	}
}

func (h *Handler) readLine() string {
	buf := make([]byte, 256)
	n, _ := h.conn.Read(buf)
	return strings.TrimSpace(string(buf[:n]))
}

func (h *Handler) read(pktChan chan string, errChan chan error) {
	defer h.wg.Done()
	defer h.debugf("Handler: read closed")

	buf := make([]byte, 512)

	for {
		select {
		case <-h.done:
			h.conn.Write([]byte("Shutdown\n"))
			sendErr(errChan, nil)
			return

		default:
			n, err := h.conn.Read(buf)
			if err != nil {
				sendErr(errChan, err)
				return
			}
			input := strings.TrimSpace(string(buf[:n]))
			if input != "" {
				pktChan <- input
			}
		}
	}
}

func (h *Handler) route(pktChan chan string, errChan chan error) {
	defer h.wg.Done()
	defer h.debugf("Handler: route closed")
	defer sendErr(errChan, nil)

	for {
		select {
		case <-h.done:
			return

		case pkt := <-pktChan:
			cmd := strings.ToLower(pkt)

			switch {
			case cmd == "exit":
				h.conn.Write([]byte("Bye!\n"))
				return

			case cmd == "help":
				h.conn.Write([]byte("*** Commands:\n"))
				h.conn.Write([]byte("    help           --- show this help message\n"))
				h.conn.Write([]byte("    post <message> --- post a message to the board\n"))
				h.conn.Write([]byte("    read           --- read recent messages\n"))
				h.conn.Write([]byte("    exit           --- quit the session\n\n"))

			case strings.HasPrefix(cmd, "post "):
				content := strings.TrimSpace(pkt[5:])
				if content == "" {
					h.conn.Write([]byte("Usage: post <message>\n"))
				} else {
					h.saveMessage(h.user, content)
					h.conn.Write([]byte("Message posted.\n"))
				}

			case cmd == "read":
				messages := h.getLastMessages(config.MaxReadMessages)
				if len(messages) == 0 {
					h.conn.Write([]byte("No messages yet.\n"))
				} else {
					for _, msg := range messages {
						h.conn.Write([]byte(fmt.Sprintf("[%s] %s: %s\n", msg.Timestamp, msg.User, msg.Content)))
					}
				}

			default:
				h.conn.Write([]byte("*** Invalid command, try 'help'\n"))
			}

			h.conn.Write([]byte("> "))
		}
	}
}

func (h *Handler) saveMessage(user, content string) {
	msg := Message{
		User:      user,
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Content:   content,
	}

	messageMutex.Lock()
	messageStore = append(messageStore, msg)

	f, err := os.OpenFile(messagesFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.WriteString(fmt.Sprintf("[%s] %s: %s\n", msg.Timestamp, user, content))
	}

	messageMutex.Unlock()
}

func (h *Handler) getLastMessages(n int) []Message {
	messageMutex.Lock()
	defer messageMutex.Unlock()

	if len(messageStore) <= n {
		return messageStore
	}
	return messageStore[len(messageStore)-n:]
}

func sendErr(ch chan error, err error) {
	if ch != nil {
		ch <- err
		// Do NOT close the channel here!
	}
}

func (h *Handler) debugf(format string, args ...interface{}) {
	if h.output == nil || !config.Debug {
		return
	}
	fmt.Fprintf(h.output, "(dbg) "+format+"\n", args...)
}

func (h *Handler) logf(format string, args ...interface{}) {
	if h.output == nil {
		return
	}
	fmt.Fprintf(h.output, format+"\n", args...)
}

func (h *Handler) loadMessages() {
	messageMutex.Lock()
	defer messageMutex.Unlock()

	messageStore = nil // Clear current store

	data, err := os.ReadFile(messagesFile)
	if err != nil {
		return // No messages file yet
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Expected format: [YYYY-MM-DD HH:MM:SS] user: content
		if len(line) < 22 || line[0] != '[' {
			continue
		}
		endIdx := strings.Index(line, "]")
		if endIdx == -1 {
			continue
		}
		timestamp := line[1:endIdx]
		rest := strings.TrimSpace(line[endIdx+1:])
		colonIdx := strings.Index(rest, ":")
		if colonIdx == -1 {
			continue
		}
		user := strings.TrimSpace(rest[:colonIdx])
		content := strings.TrimSpace(rest[colonIdx+1:])
		messageStore = append(messageStore, Message{
			User:      user,
			Timestamp: timestamp,
			Content:   content,
		})
	}
}
