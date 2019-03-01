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
	t.SkipNow()
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
						level: logging.WARNING,
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
	t.SkipNow()
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
	t.SkipNow()
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
	t.SkipNow()
	var testCases = []struct {
		in  string
		out logging.Level
		err error
	}{
		{
			in:  "0",
			out: logging.WARNING,
			err: nil,
		},
		{
			in:  "1",
			out: logging.INFO,
			err: nil,
		},
		{
			in:  "2",
			out: logging.DEBUG,
			err: nil,
		},
		{
			in:  "3",
			out: logging.CRITICAL,
			err: fmt.Errorf("'3' is not a valid value. Valid values: 0,1,2"),
		},
		{
			in:  "-12",
			out: logging.CRITICAL,
			err: fmt.Errorf("'-12' is not a valid value. Valid values: 0,1,2"),
		},
		{
			in:  "a",
			out: logging.CRITICAL,
			err: fmt.Errorf("'a' is not an integer"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.in, func(t *testing.T) {
			l, err := parseLogLevel(tc.in)
			assert.Equal(t, tc.err, err)
			if err == nil {
				assert.Equal(t, tc.out, l)
			}
		})
	}
}
