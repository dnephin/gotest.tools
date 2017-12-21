package main

import (
	"log"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
)

func TestRun(t *testing.T) {
	log.SetFlags(0)
	err := run(&options{
		dirs:   []string{"./testdata/full"},
		dryRun: true,
	})
	assert.NoError(t, err)
}
