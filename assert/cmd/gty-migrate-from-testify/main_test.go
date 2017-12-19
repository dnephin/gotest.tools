package main

import (
	"log"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
)

func TestRun(t *testing.T) {
	log.SetFlags(0)
	err := run(&options{
		dirs:   []string{"github.com/gotestyourself/gotestyourself/foo"},
		dryRun: true,
	})
	assert.NoError(t, err)
}
