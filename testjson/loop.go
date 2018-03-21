package testjson

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/pkg/errors"
)

// Action of TestEvent
type Action string

// nolint: unused
const (
	ActionRun    Action = "run"
	ActionPause  Action = "pause"
	ActionCont   Action = "cont"
	ActionPass   Action = "pass"
	ActionBench  Action = "bench"
	ActionFail   Action = "fail"
	ActionOutput Action = "output"
	ActionSkip   Action = "skip"
)

// TestEvent is a structure output by go tool test2json and go test -json.
type TestEvent struct {
	// Time encoded as an RFC3339-format string
	Time    time.Time
	Action  Action
	Package string
	Test    string
	// Elapsed time in seconds
	// TODO: use time.Duration field with deserialization
	Elapsed float64
	// Output of test or benchmark
	Output string
}

// PackageEvent returns true if the event is a package start or end event
func (e TestEvent) PackageEvent() bool {
	return e.Test == ""
}

// ElapsedFormatted returns Elapsed formatted in the go test format, ex (0.00s).
func (e TestEvent) ElapsedFormatted() string {
	return fmt.Sprintf("(%.2f)", e.Elapsed)
}

// Package is the set of TestEvents for a single go package
type Package struct {
	run    int
	failed []TestEvent
	//skipped []TestEvent
	//passed []time.Duration
	output map[string]*bytes.Buffer
}

func newPackage() *Package {
	return &Package{
		output: make(map[string]*bytes.Buffer),
	}
}

// Execution of one or more test packages
type Execution struct {
	started  time.Time
	packages map[string]*Package
	errors   []string
}

// TODO: detect skipped tests
func (e *Execution) add(event TestEvent) {
	pkg, ok := e.packages[event.Package]
	if !ok {
		pkg = newPackage()
		e.packages[event.Package] = pkg
	}
	if event.PackageEvent() {
		return
	}

	switch event.Action {
	case ActionRun:
		pkg.run += 1
	case ActionPass:
		//pkg.passed = append(pkg.passed, testDuration(event))
	case ActionFail:
		pkg.failed = append(pkg.failed, event)
	case ActionOutput, ActionBench:
		if pkg.output[event.Test] == nil {
			pkg.output[event.Test] = new(bytes.Buffer)
		}
		// TODO: limit size of buffered test output
		pkg.output[event.Test].WriteString(event.Output)
	}
}

func testDuration(event TestEvent) time.Duration {
	return time.Duration(event.Elapsed*1000) * time.Millisecond
}

// Output returns the full test output for a test.
func (e *Execution) Output(event TestEvent) string {
	output := e.packages[event.Package].output[event.Test]
	if output == nil {
		return ""
	}
	return output.String()
}

func (e *Execution) Package(event TestEvent) *Package {
	return e.packages[event.Package]
}

var clock = clockwork.NewRealClock()

func (e *Execution) Elapsed() time.Duration {
	return clock.Now().Sub(e.started)
}

func (e *Execution) Failed() []TestEvent {
	failed := []TestEvent{}
	for _, pkg := range e.packages {
		failed = append(failed, pkg.failed...)
	}
	return failed
}

func (e *Execution) Total() int {
	total := 0
	for _, pkg := range e.packages {
		total += pkg.run
	}
	return total
}

func (e *Execution) addError(err string) {
	// TODO: may need locking, or use a channel
	e.errors = append(e.errors, err)
}

func (e *Execution) Errors() []string {
	return e.errors
}

func NewExecution() *Execution {
	return &Execution{
		started:  time.Now(),
		packages: make(map[string]*Package),
	}
}

type HandleEvent func(event TestEvent, output *Execution) (string, error)

type ScanConfig struct {
	Stdout  io.Reader
	Stderr  io.Reader
	Out     io.Writer
	Err     io.Writer
	Handler HandleEvent
}

func ScanTestOutput(config ScanConfig) (*Execution, error) {
	execution := NewExecution()
	waitOnStderr := readStderr(config.Stderr, config.Err, execution)
	scanner := bufio.NewScanner(config.Stdout)

	for scanner.Scan() {
		raw := scanner.Bytes()
		event, err := parseEvent(raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse test output: %s", string(raw))
		}
		execution.add(event)
		line, err := config.Handler(event, execution)
		if err != nil {
			return nil, err
		}
		config.Out.Write([]byte(line))
	}

	if err := <-waitOnStderr; err != nil {
		// TODO: log failure
	}
	return execution, errors.Wrap(scanner.Err(), "failed to scan test output")
}

func readStderr(in io.Reader, out io.Writer, exec *Execution) chan error {
	wait := make(chan error)
	go func() {
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			exec.addError(scanner.Text())
			out.Write(scanner.Bytes())
		}
		wait <- scanner.Err()
		close(wait)
	}()
	return wait
}

func parseEvent(raw []byte) (TestEvent, error) {
	event := TestEvent{}
	err := json.Unmarshal(raw, &event)
	return event, err
}
