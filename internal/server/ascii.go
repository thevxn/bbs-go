package server

import (
	"fmt"
	"os"
	"strings"

	"bbs-go/internal/config"
)

var WelcomeMessage string

func LoadMOTD(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	msg := string(content)
	msg = strings.ReplaceAll(msg, "{{VERSION}}", config.Version)
	msg = strings.ReplaceAll(msg, "{{HOST}}", config.Host)
	msg = strings.ReplaceAll(msg, "{{PORT}}", fmt.Sprintf("%d", config.Port))

	WelcomeMessage = msg
	return nil
}
