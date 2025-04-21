package server

import (
	"errors"
)

var (
	ErrShutdownStarted = errors.New("The server shutdown process has been already started")
)
