package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/op/go-logging"
)

var errorLevel = map[int]logging.Level{
	0: logging.WARNING,
	1: logging.INFO,
	2: logging.DEBUG,
}

func initWriterLogger(logLevel int, writer io.Writer, moduleName string, format string) (logging.LeveledBackend, error) {
	formatter := logging.MustStringFormatter(format)
	backend := logging.NewLogBackend(writer, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, formatter)
	module := logging.AddModuleLevel(backendFormatter)
	module.SetLevel(errorLevel[logLevel], moduleName)

	return module, nil
}

func initFileLogger(logLevel int, fileName string, moduleName string, format string) (*os.File, logging.Backend, error) {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		if _, err := os.Create(fileName); err != nil {
			return nil, nil, fmt.Errorf("error creating log file: %s", err)
		}
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening log file: %s", err)
	}
	fileWriter := io.Writer(file)
	module, err := initWriterLogger(logLevel, fileWriter, moduleName, format)
	return file, module, err
}

// LoggerInit Initializes the logger
func LoggerInit(config *Config) []*os.File {
	var backends []logging.Backend
	var files []*os.File

	const moduleName = "grimd"

	logConfig, err := ParseLogConfig(config.LogConfig)
	if err != nil {
		panic(fmt.Sprintf("Cannot parse LogConfig: %s -  %v", config.LogConfig, err))
	}
	if logConfig.stderr {
		stderrBackend, err := initWriterLogger(config.LogLevel, os.Stderr, moduleName, `%{color}%{time:15:04:05.000} %{level:.4s} %{shortfile} ▶ %{id:03x}%{color:reset} %{message}`)
		if err == nil {
			backends = append(backends, stderrBackend)
		}
	}
	for _, logFile := range logConfig.files {
		file, fileBackend, err := initFileLogger(config.LogLevel, logFile, moduleName, `%{time:15:04:05.000} %{level:.4s} %{shortfile} ▶ %{id:03x} %{message}`)
		if err == nil {
			backends = append(backends, fileBackend)
			files = append(files, file)
		}
	}
	logging.SetBackend(backends...)
	return files
}

// LogConfig type
type LogConfig struct {
	files  []string
	syslog bool
	stderr bool
}

// ParseLogConfig parses a log config string and returns a LogConfig structure
func ParseLogConfig(config string) (LogConfig, error) {
	var c = LogConfig{}
	options := strings.Split(config, ",")
	for _, option := range options {
		parts := strings.Split(option, ":")
		switch parts[0] {
		case "file":
			c.files = append(c.files, parts[1])
			break
		case "syslog":
			c.syslog = true
			break
		case "stderr":
			c.stderr = true
			break
		}
	}
	return c, nil
}

var (
	// Initialize to a dummy but functional logger so that tested code has something to write to
	logger = logging.MustGetLogger("test")
)
