package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/op/go-logging"
)

type fileConfig struct {
	name  string
	level logging.Level
}

type boolConfig struct {
	enabled bool
	level   logging.Level
}

type logConfig struct {
	files  []fileConfig
	syslog boolConfig
	stderr boolConfig
}

func parseLogLevel(level string) (logging.Level, error) {
	errorLevel := map[int]logging.Level{
		0: logging.WARNING,
		1: logging.INFO,
		2: logging.DEBUG,
	}

	l, err := strconv.Atoi(level)
	if err != nil {
		return logging.CRITICAL, fmt.Errorf("'%s' is not an integer", level)
	}

	if l < 0 || l > 2 {
		return logging.CRITICAL, fmt.Errorf("'%s' is not a valid value. Valid values: 0,1,2", level)
	}

	return errorLevel[l], nil
}

func parseLogConfig(logConfigString string) (*logConfig, error) {
	var result logConfig
	fileRe := regexp.MustCompile(`file:([^@]+)@(\S+)`)
	boolRe := regexp.MustCompile(`(syslog|stderr)@(\S+)`)

	for _, part := range strings.Split(logConfigString, ",") {

		match := fileRe.FindStringSubmatch(part)
		if match != nil {
			l, err := parseLogLevel(match[2])
			if err != nil {
				return nil, fmt.Errorf("Error while parsing '%s': %s", match[0], err.Error())
			}
			result.files = append(result.files, fileConfig{match[1], l})
			continue
		}

		match = boolRe.FindStringSubmatch(part)
		if match != nil {
			l, err := parseLogLevel(match[2])
			if err != nil {
				return nil, fmt.Errorf("Error while parsing '%s': %s", match[0], err.Error())
			}
			switch match[1] {
			case "syslog":
				result.syslog = boolConfig{true, l}
				continue
			case "stderr":
				result.stderr = boolConfig{true, l}
				continue
			}
		}

		return nil, fmt.Errorf("Error: uknown log config fragment: '%s'", part)

	}
	return &result, nil
}

func createLogFile(fileName string) (*os.File, error) {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		if _, err := os.Create(fileName); err != nil {
			return nil, fmt.Errorf("error creating log file '%s': %s", fileName, err)
		}
	}

	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening log file: %s", err)
	}

	return file, nil
}

func decorateBackend(backend logging.Backend, level logging.Level, format string, moduleName string) logging.LeveledBackend {
	stringFormatter := logging.MustStringFormatter(format)
	beFormatter := logging.NewBackendFormatter(backend, stringFormatter)
	leveled := logging.AddModuleLevel(beFormatter)
	leveled.SetLevel(level, moduleName)
	return leveled
}

func createLoggerFromFile(file *os.File, level logging.Level, format string, moduleName string) logging.LeveledBackend {
	writer := io.Writer(file)
	backend := logging.NewLogBackend(writer, "", 0)
	return decorateBackend(backend, level, format, moduleName)
}

func createFileLogger(cfg fileConfig, moduleName string) (*logging.LeveledBackend, *os.File, error) {
	file, err := createLogFile(cfg.name)
	if err != nil {
		return nil, nil, err
	}

	formatString := `%{time:15:04:05.000} %{level:.4s} %{shortfile} ▶ %{id:03x} %{message}`
	backend := createLoggerFromFile(file, cfg.level, formatString, moduleName)
	return &backend, file, nil
}

func createSyslogBackend(cfg boolConfig, moduleName string) (*logging.LeveledBackend, error) {
	backend, err := logging.NewSyslogBackend("Grimd")
	if err != nil {
		return nil, err
	}
	format := `%{time:15:04:05.000} %{level:.4s} %{shortfile} ▶ %{id:03x} %{message}`
	decorated := decorateBackend(backend, cfg.level, format, moduleName)
	return &decorated, nil
}

// LoggerInit Initializes the logger
func LoggerInit(cfg string) ([]*os.File, error) {

	logConfig, err := parseLogConfig(cfg)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("%d", errorLevel[0])
	//fmt.Printf("%#v", logConfig)

	const moduleName = "grimd"
	logger = logging.MustGetLogger(moduleName)

	var backends []logging.Backend
	var openFiles []*os.File

	for _, f := range logConfig.files {
		b, file, err := createFileLogger(f, moduleName)
		if err != nil {
			for _, toClose := range openFiles {
				toClose.Close()
			}
			return nil, err
		}
		backends = append(backends, *b)
		openFiles = append(openFiles, file)
	}

	if logConfig.stderr.enabled {
		var format = `%{color}%{time:15:04:05.000} %{level:.4s} %{shortfile} ▶ %{id:03x}%{color:reset} %{message}`
		stderrLogger := createLoggerFromFile(os.Stderr, logConfig.stderr.level, format, moduleName)
		backends = append(backends, stderrLogger)
	}

	if logConfig.syslog.enabled {
		syslogLogger, err := createSyslogBackend(logConfig.syslog, moduleName)
		if err != nil {
			panic(err)
		}
		backends = append(backends, *syslogLogger)
	}

	logging.SetBackend(backends...)

	return openFiles, nil
}

var (
	// Initialize to a dummy but functional logger so that tested code has something to write to
	logger *logging.Logger = logging.MustGetLogger("test")
)
