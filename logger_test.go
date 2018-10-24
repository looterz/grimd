package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThatCanFindOneFile(t *testing.T) {
	var logConfigString = "file:some.file,null,unknown:thin.g"
	logConfig, err := ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.NotNil(t, logConfig)
	assert.Equal(t, len(logConfig.files), 1)
	assert.Equal(t, logConfig.files[0], "some.file")
	assert.False(t, logConfig.syslog)
}

func TestThatSysLogWorks(t *testing.T) {
	var logConfigString = "whatever"
	logConfig, err := ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.False(t, logConfig.syslog)
	logConfigString = "syslog"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.syslog)
	logConfigString = "syslog,file:some.file"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.syslog)
	logConfigString = "file:some.file,syslog"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.syslog)
	logConfigString = "file:some.file,syslog,file:whaaaaat"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.syslog)
	logConfigString = "file:some.file,file:test,syslog"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.syslog)
	logConfigString = "file:syslog"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.False(t, logConfig.syslog)
	assert.Equal(t, "syslog", logConfig.files[0])
	logConfigString = "wrong:entry,syslog"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.syslog)
	logConfigString = "syslog,wrong:entry"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.syslog)
	logConfigString = "syslog,file:entry,syslog"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.syslog)
}

func TestThatStdErrWorks(t *testing.T) {
	var logConfigString = "whatever"
	logConfig, err := ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.False(t, logConfig.stderr)
	logConfigString = "stderr"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.stderr)
	logConfigString = "stderr,file:some.file"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.stderr)
	logConfigString = "file:some.file,stderr"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.stderr)
	logConfigString = "file:some.file,stderr,file:whaaaaat"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.stderr)
	logConfigString = "file:some.file,file:test,stderr"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.stderr)
	logConfigString = "file:stderr"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.False(t, logConfig.stderr)
	assert.Equal(t, "stderr", logConfig.files[0])
	logConfigString = "wrong:entry,stderr"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.stderr)
	logConfigString = "stderr,wrong:entry"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.stderr)
	logConfigString = "stderr,file:entry,stderr"
	logConfig, err = ParseLogConfig(logConfigString)
	assert.Nil(t, err)
	assert.True(t, logConfig.stderr)
}
