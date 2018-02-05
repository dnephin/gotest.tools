package testjson

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

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

// Package is the set of TestEvents for a single go package
type Package struct {
	run    int
	failed []TestEvent
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
}

// TODO: detect skipped tests
func (e *Execution) add(event TestEvent) {
	pkg, ok := e.packages[event.Package]
	if !ok {
		pkg = newPackage()
		e.packages[event.Package] = pkg
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

func (e *Execution) Elapsed() time.Duration {
	return time.Now().Sub(e.started)
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

func NewExecution() *Execution {
	return &Execution{
		started:  time.Now(),
		packages: make(map[string]*Package),
	}
}

type HandleEvent func(event TestEvent, output *Execution) error

func ScanTestOutput(in io.Reader, handler HandleEvent) (*Execution, error) {
	execution := NewExecution()
	scanner := bufio.NewScanner(in)

	for scanner.Scan() {
		raw := scanner.Bytes()
		event, err := parseEvent(raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse test output: %s", string(raw))
		}
		execution.add(event)
		if err := handler(event, execution); err != nil {
			return nil, err
		}
	}
	return execution, errors.Wrap(scanner.Err(), "failed to scan test output")
}

func parseEvent(raw []byte) (TestEvent, error) {
	event := TestEvent{}
	err := json.Unmarshal(raw, &event)
	return event, err
}

type proc struct {
	cmd    *exec.Cmd
	stdout io.Reader
}

func StartGoTest(args []string) (proc, error) {
	p := proc{
		cmd: exec.Command("go", prepend("test", args...)...),
	}
	// TODO: how to link stderr to a test?
	p.cmd.Stderr = os.Stderr
	var err error
	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return p, err
	}
	return p, p.cmd.Start()
}

func prepend(first string, rest ...string) []string {
	return append([]string{first}, rest...)
}

type Options struct {
	Format []string
}

func Run(opts *Options) error {
	// TODO: args
	args := []string{"-json", "./..."}
	proc, err := StartGoTest(args)
	if err != nil {
		return errors.Wrapf(err, "failed to run %s %s",
			proc.cmd.Path,
			strings.Join(proc.cmd.Args, " "))
	}
	printer := NewHandler(opts.Format)
	exec, err := ScanTestOutput(proc.stdout, printer)
	if err != nil {
		return err
	}
	// TODO: make an interface based on format
	if err := PrintExecution(exec); err != nil {
		return err
	}
	// TODO: return errNonZeroExit instead of "exit status 1"
	return proc.cmd.Wait()
}
