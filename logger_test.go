package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/op/go-logging"

	"github.com/stretchr/testify/assert"
)

func TestLogConfigParsing(t *testing.T) {
	var fileTests = []struct {
		in  string
		out *logConfig
		err error
	}{
		{
			in: "file:grimd.log@0",
			out: &logConfig{
				files: []fileConfig{
					fileConfig{
						name:  "grimd.log",
						level: logging.WARNING,
					},
				},
			},
			err: nil,
		},
		{
			in: "file:grimd.log@0,file:something.log@1",
			out: &logConfig{
				files: []fileConfig{
					fileConfig{
						name:  "grimd.log",
						level: 0,
					},
					fileConfig{
						name:  "something.log",
						level: logging.INFO,
					},
				},
			},
			err: nil,
		},
		{
			in:  "file:grimd.log@aa",
			out: nil,
			err: fmt.Errorf("Error while parsing 'file:grimd.log@aa': 'aa' is not an integer"),
		},
		{
			in:  "syslog@aa",
			out: nil,
			err: fmt.Errorf("Error while parsing 'syslog@aa': 'aa' is not an integer"),
		},
		{
			in:  "fail:grimd.log@1",
			out: nil,
			err: fmt.Errorf("Error: uknown log config fragment: 'fail:grimd.log@1'"),
		},
		{
			in: "syslog@0",
			out: &logConfig{
				syslog: boolConfig{
					enabled: true,
					level:   logging.WARNING,
				},
			},
			err: nil,
		},
		{
			in: "file:grimd.log@1,syslog@1",
			out: &logConfig{
				files: []fileConfig{
					fileConfig{
						name:  "grimd.log",
						level: logging.INFO,
					},
				},
				syslog: boolConfig{
					enabled: true,
					level:   logging.INFO,
				},
			},
			err: nil,
		},
		{
			in: "stderr@2",
			out: &logConfig{
				stderr: boolConfig{
					enabled: true,
					level:   logging.DEBUG,
				},
			},
			err: nil,
		},
		{
			in: "file:grimd.log@2,syslog@1,stderr@0",
			out: &logConfig{
				files: []fileConfig{
					fileConfig{
						name:  "grimd.log",
						level: logging.DEBUG,
					},
				},
				syslog: boolConfig{
					enabled: true,
					level:   logging.INFO,
				},
				stderr: boolConfig{
					enabled: true,
					level:   logging.WARNING,
				},
			},
			err: nil,
		},
	}

	for _, test := range fileTests {
		t.Run(test.in, func(t *testing.T) {
			result, err := parseLogConfig(test.in)
			if err == nil {
				assert.Equal(t, test.out, result)
			} else {
				assert.Equal(t, test.err, err)
			}
		})
	}

}

func TestCreateLogFile(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	var testCases = []struct {
		in  string
		out *os.File
		err error
	}{
		{
			in:  "",
			err: fmt.Errorf("error creating log file '': open : no such file or directory"),
		},
		{
			in:  fmt.Sprintf("%s/first", dir),
			err: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.in, func(t *testing.T) {
			result, err := createLogFile(test.in)
			assert.Equal(t, test.err, err)
			if err == nil {
				assert.NotNil(t, result)
				result.Close()
				os.Remove(test.in)
			}
		})
	}
}

func TestCreateFileLogger(t *testing.T) {
	dir, err := ioutil.TempDir("", "test")
	assert.Nil(t, err)
	defer os.RemoveAll(dir)

	var testCases = []struct {
		in  string
		err error
	}{
		{
			in:  fmt.Sprintf("%s/one", dir),
			err: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.in, func(t *testing.T) {
			cfg := fileConfig{tc.in, logging.WARNING}
			logger, file, err := createFileLogger(cfg, "module")
			assert.Equal(t, tc.err, err)
			if err == nil {
				assert.NotNil(t, logger)
				assert.NotNil(t, file)
				file.Close()
			}
		})
	}
}

func TestLogLevelParse(t *testing.T) {
	l, err := parseLogLevel("0")
	assert.Nil(t, err)
	assert.Equal(t, logging.WARNING, l)
}
