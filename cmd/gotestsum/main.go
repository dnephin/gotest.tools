package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/gotestyourself/gotestyourself/internal/cmd"
	"github.com/gotestyourself/gotestyourself/testjson"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

func main() {
	name := os.Args[0]
	flags, opts := setupFlags(name)
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}
	opts.args = flags.Args()

	switch err := run(opts).(type) {
	case nil:
	case *exec.ExitError:
		// go test should already report the error to stderr so just exit with
		// the same status code
		os.Exit(cmd.ExitCodeWithDefault(err))
	default:
		fmt.Fprintln(os.Stderr, name+": Error: "+err.Error())
		os.Exit(3)
	}
}

func setupFlags(name string) (*pflag.FlagSet, *options) {
	opts := &options{}
	flags := pflag.NewFlagSet(name, pflag.ContinueOnError)
	flags.SetInterspersed(false)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage:
    %s [flags] [--] [go test flags]

Flags:
`, name)
		flags.PrintDefaults()
		fmt.Fprint(os.Stderr, `
Formats:
    dots              print a character for each test
    short             print a line for each package
    short-verbose     print a line for each test and package
    standard-quiet    default go test format
    standard-verbose  default go test -v format
`)
	}
	flags.BoolVar(&opts.debug, "debug", false, "enabled debug")
	flags.StringVar(&opts.format, "format", "short",
		"print format of test input")
	flags.BoolVar(&opts.rawCommand, "raw-command", false,
		"don't prepend 'go test -json' to the 'go test' command")
	return flags, opts
}

type options struct {
	args       []string
	format     string
	debug      bool
	rawCommand bool
}

// TODO: add flag --max-failures
// TODO: use logrus
func run(opts *options) error {
	ctx := context.Background()
	goTestProc, err := startGoTest(ctx, goTestCmdArgs(opts), opts.debug)
	if err != nil {
		return errors.Wrapf(err, "failed to run %s %s",
			goTestProc.cmd.Path,
			strings.Join(goTestProc.cmd.Args, " "))
	}
	defer goTestProc.cancel()

	out := os.Stdout
	handler := testjson.NewEventHandler(opts.format)
	if handler == nil {
		return errors.Errorf("unknown format %s", opts.format)
	}
	exec, err := testjson.ScanTestOutput(testjson.ScanConfig{
		Stdout:  goTestProc.stdout,
		Stderr:  goTestProc.stderr,
		Handler: handler,
		Out:     out,
		Err:     os.Stderr,
	})
	if err != nil {
		return err
	}
	// TODO: make an interface based on a --summary flag
	if err := testjson.PrintSummary(out, exec); err != nil {
		return err
	}
	return goTestProc.cmd.Wait()
}

func goTestCmdArgs(opts *options) []string {
	args := opts.args
	defaultArgs := []string{"go", "test"}
	switch {
	case opts.rawCommand:
		return args
	case len(args) == 0:
		return append(defaultArgs, "-json", "./...")
	case !hasJSONArg(args):
		defaultArgs = append(defaultArgs, "-json")
	}
	return append(defaultArgs, args...)
}

func hasJSONArg(args []string) bool {
	for _, arg := range args {
		if arg == "-json" || arg == "--json" {
			return true
		}
	}
	return false
}

type proc struct {
	cmd    *exec.Cmd
	stdout io.Reader
	stderr io.Reader
	cancel func()
}

func startGoTest(ctx context.Context, args []string, debug bool) (proc, error) {
	ctx, cancel := context.WithCancel(ctx)
	p := proc{
		cmd:    exec.CommandContext(ctx, args[0], args[1:]...),
		cancel: cancel,
	}
	if debug {
		log.Printf("exec: %s", p.cmd.Args)
	}
	var err error
	p.stdout, err = p.cmd.StdoutPipe()
	if err != nil {
		return p, err
	}
	p.stderr, err = p.cmd.StderrPipe()
	if err != nil {
		return p, err
	}
	return p, p.cmd.Start()
}
