package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsing(t *testing.T) {
	newBlockCache := &MemoryBlockCache{Backend: make(map[string]bool)}
	expected := []string{
		"domain1.some",
		"domain.with.number",
		"domain.2.with.number",
		"domain.with.tab",
		"domain.with.spaces",
		"domain.with.many.spaces",
		"domain.with.comment",
		"domain.with.number.and.comment",
		"domain.with.attached.comment",
		"domain.with.number.and.attached.comment",
	}
	err := parseHostFile("testdata/parser_data.list", newBlockCache)
	assert.Nil(t, err)
	assert.Equal(t, len(expected), newBlockCache.Length())
	for _, host := range expected {
		val, err := newBlockCache.Get(host)
		assert.Nil(t, err)
		if err != nil {
			assert.Equalf(t, true, val, "Unexpected blocked state: %v for: %s", val, host)
		}
	}

}
