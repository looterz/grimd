package main

import (
	"fmt"
	"io"
	"os"

	"github.com/op/go-logging"
)

// LoggerInit Initializes the logger
func LoggerInit(logLevel int, logFile string) (*os.File, error) {
	errorLevel := map[int]logging.Level{
		0: logging.WARNING,
		1: logging.INFO,
		2: logging.DEBUG,
	}
	const moduleName = "grimd"
	logger = logging.MustGetLogger(moduleName)
	var format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{level:.4s} %{shortfile} ▶ %{id:03x}%{color:reset} %{message}`)
	stderrBackend := logging.NewLogBackend(os.Stderr, "", 0)
	stderrFormatter := logging.NewBackendFormatter(stderrBackend, format)
	stderrLeveled := logging.AddModuleLevel(stderrFormatter)

	stderrLeveled.SetLevel(errorLevel[logLevel], moduleName)

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		if _, err := os.Create(logFile); err != nil {
			return nil, fmt.Errorf("error creating log file: %s", err)
		}
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening log file: %s", err)
	}

	format = logging.MustStringFormatter(
		`%{time:15:04:05.000} %{level:.4s} %{shortfile} ▶ %{id:03x} %{message}`)
	fileWriter := io.Writer(file)
	fileBackend := logging.NewLogBackend(fileWriter, "", 0)
	fileFormatter := logging.NewBackendFormatter(fileBackend, format)
	fileLeveled := logging.AddModuleLevel(fileFormatter)

	fileLeveled.SetLevel(errorLevel[logLevel], moduleName)

	logging.SetBackend(stderrLeveled, fileLeveled)

	return file, nil
}

var (
	// Initialize to a dummy but functional logger so that tested code has something to write to
	logger = logging.MustGetLogger("test")
)
