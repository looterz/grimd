package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

// LoggerInit Initializes the logger
func LoggerInit(logFile string) (*os.File, error) {
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		if _, err := os.Create(logFile); err != nil {
			return nil, fmt.Errorf("error creating log file: %s", err)
		}
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening log file: %s", err)
	}

	logWriter := io.MultiWriter(os.Stdout, file)

	log.SetOutput(logWriter)
	log.SetFlags(log.Ldate | log.Ltime)

	return file, nil
}
