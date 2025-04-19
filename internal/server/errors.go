package server

import (
	"errors"
)

var (
	ErrShutdownStarted = errors.New("the server shutdown process has been already started")
)
