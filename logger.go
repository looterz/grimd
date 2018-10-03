package main

import (
	"fmt"
	"io"
	"os"

	"github.com/op/go-logging"
)

// LoggerInit Initializes the logger
func LoggerInit(logFile string) (*os.File, error) {
	error_level := map[int]logging.Level{
		0: logging.WARNING,
		1: logging.INFO,
		2: logging.DEBUG,
	}
	const module_name = "grimd"
	logger = logging.MustGetLogger(module_name)
	var format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{level:.4s} %{shortfile} ▶ %{id:03x}%{color:reset} %{message}`)
	stderr_backend := logging.NewLogBackend(os.Stderr, "", 0)
	stderr_formatter := logging.NewBackendFormatter(stderr_backend, format)
	stderr_leveled := logging.AddModuleLevel(stderr_formatter)

	stderr_leveled.SetLevel(error_level[Config.LogLevel], module_name)

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
	file_writer := io.Writer(file)
	file_backend := logging.NewLogBackend(file_writer, "", 0)
	file_formatter := logging.NewBackendFormatter(file_backend, format)
	file_leveled := logging.AddModuleLevel(file_formatter)

	file_leveled.SetLevel(error_level[Config.LogLevel], module_name)

	logging.SetBackend(stderr_leveled, file_leveled)

	return file, nil
}

var (
	logger *logging.Logger = nil
)
