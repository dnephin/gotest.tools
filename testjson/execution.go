package testjson

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
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
	return fmt.Sprintf("(%.2fs)", e.Elapsed)
}

// Package is the set of TestEvents for a single go package
type Package struct {
	run     int
	failed  []TestCase
	skipped []TestCase
	//passed []time.Duration
	output map[string][]string
}

// TestCase stores the name and elapsed time for a test case.
type TestCase struct {
	Package string
	Test    string
	Elapsed time.Duration
}

func newPackage() *Package {
	return &Package{output: make(map[string][]string)}
}

// Execution of one or more test packages
type Execution struct {
	started  time.Time
	packages map[string]*Package
	errors   []string
}

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
		pkg.run++
	case ActionPass:
		//pkg.passed = append(pkg.passed, testDuration(event))
	case ActionFail:
		pkg.failed = append(pkg.failed, TestCase{
			Package: event.Package,
			Test:    event.Test,
			Elapsed: elapsedDuration(event),
		})
	case ActionSkip:
		pkg.skipped = append(pkg.skipped, TestCase{
			Package: event.Package,
			Test:    event.Test,
			Elapsed: elapsedDuration(event),
		})
	case ActionOutput, ActionBench:
		// TODO: only store output for failed and skipped tests
		// TODO: limit size of buffered test output
		pkg.output[event.Test] = append(pkg.output[event.Test], event.Output)
	}
}

func elapsedDuration(event TestEvent) time.Duration {
	return time.Duration(event.Elapsed*1000) * time.Millisecond
}

// Output returns the full test output for a test.
func (e *Execution) Output(pkg, test string) []string {
	return e.packages[pkg].output[test]
}

// Package returns the Package for a TestEvent.
func (e *Execution) Package(event TestEvent) *Package {
	return e.packages[event.Package]
}

var clock = clockwork.NewRealClock()

// Elapsed returns the time elapsed since the execution started.
func (e *Execution) Elapsed() time.Duration {
	return clock.Now().Sub(e.started)
}

// Failed returns a list of all the failed test cases.
func (e *Execution) Failed() []TestCase {
	var failed []TestCase
	for _, pkg := range sortedKeys(e.packages) {
		failed = append(failed, e.packages[pkg].failed...)
	}
	return failed
}

func sortedKeys(pkgs map[string]*Package) []string {
	var keys []string
	for key := range pkgs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Skipped returns a list of all the skipped test cases.
func (e *Execution) Skipped() []TestCase {
	var skipped []TestCase
	for _, pkg := range sortedKeys(e.packages) {
		skipped = append(skipped, e.packages[pkg].skipped...)
	}
	return skipped
}

// Total returns a count of all test cases.
func (e *Execution) Total() int {
	total := 0
	for _, pkg := range e.packages {
		total += pkg.run
	}
	return total
}

func (e *Execution) addError(err string) {
	// Build errors start with a header
	if strings.HasPrefix(err, "# ") {
		return
	}
	// TODO: may need locking, or use a channel
	e.errors = append(e.errors, err)
}

// Errors returns a list of all the errors.
func (e *Execution) Errors() []string {
	return e.errors
}

// NewExecution returns a new Execution and records the current time as the
// time the test execution started.
func NewExecution() *Execution {
	return &Execution{
		started:  time.Now(),
		packages: make(map[string]*Package),
	}
}

// HandleEvent is a function which handles an event and returns a string to
// output for the event.
type HandleEvent func(event TestEvent, output *Execution) (string, error)

// ScanConfig used by ScanTestOutput
type ScanConfig struct {
	Stdout  io.Reader
	Stderr  io.Reader
	Out     io.Writer
	Err     io.Writer
	Handler HandleEvent
}

// ScanTestOutput reads lines from stdout and stderr, creates an Execution,
// calls the Handler for each event, and returns the Execution.
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
			out.Write(append(scanner.Bytes(), '\n'))
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
