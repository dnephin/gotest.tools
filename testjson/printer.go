package testjson

import (
	"fmt"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func debugFormat(event TestEvent, _ *Execution) (string, error) {
	return fmt.Sprintf("%s %s %s (%.3f) [%d] %s\n",
		event.Package,
		event.Test,
		event.Action,
		event.Elapsed,
		event.Time.Unix(),
		event.Output), nil
}

// go test -v
func standardVerboseFormat(event TestEvent, _ *Execution) (string, error) {
	if event.Action == ActionOutput && !event.PackageEvent() {
		return event.Output, nil
	}
	return "", nil
}

// go test
func standardQuietFormat(event TestEvent, _ *Execution) (string, error) {
	if isPackageEndEvent(event) {
		return event.Output, nil
	}
	return "", nil
}

func shortVerboseFormat(event TestEvent, exec *Execution) (string, error) {
	switch {
	case event.Action == ActionSkip && event.PackageEvent():
		return "EMPTY " + relativePackagePath(event.Package) + "\n", nil
	case event.Action == ActionPass && event.PackageEvent():
		return "PASS " + relativePackagePath(event.Package) + "\n", nil
	case event.Action == ActionFail && event.PackageEvent():
		return "FAIL " + relativePackagePath(event.Package) + "\n", nil
	case event.Action == ActionPass:
		return fmt.Sprintf("--- PASS %s %s %s\n",
			relativePackagePath(event.Package),
			event.Test,
			event.ElapsedFormatted(),
		), nil
	case event.Action == ActionFail:
		return fmt.Sprintf("%s--- FAIL %s %s %s\n",
			strings.Join(exec.Output(event.Package, event.Test), ""),
			relativePackagePath(event.Package),
			event.Test,
			event.ElapsedFormatted(),
		), nil
	}
	return "", nil
}

func shortFormat(event TestEvent, _ *Execution) (string, error) {
	if !event.PackageEvent() {
		return "", nil
	}
	fmtElapsed := func() string {
		d := elapsedDuration(event)
		if d == 0 {
			return ""
		}
		return fmt.Sprintf(" (%s)", d)
	}
	fmtEvent := func(action string) (string, error) {
		return fmt.Sprintf("%s  %s%s\n",
			action, relativePackagePath(event.Package), fmtElapsed()), nil
	}
	switch event.Action {
	case ActionSkip:
		return fmtEvent("∅")
	case ActionPass:
		return fmtEvent("✓")
	case ActionFail:
		return fmtEvent("✖")
	}
	return "", nil
}

func isPackageEndEvent(event TestEvent) bool {
	if event.Action != ActionOutput || !event.PackageEvent() {
		return false
	}
	return strings.HasPrefix(event.Output, "ok ") || strings.HasPrefix(event.Output, "? ")
}

func dotsFormat(event TestEvent, exec *Execution) (string, error) {
	pkg := exec.Package(event)

	switch {
	case event.PackageEvent():
		return "", nil
	case event.Action == ActionRun && pkg.run == 1:
		return "[" + relativePackagePath(event.Package) + "]", nil
	case event.Action == ActionPass:
		return "·", nil
	case event.Action == ActionFail:
		return "✖", nil
	case event.Action == ActionSkip:
		return "↷", nil
	}
	return "", nil
}

func PrintExecution(out io.Writer, execution *Execution) error {
	errors := execution.Errors()
	fmt.Fprintf(out, "\nDONE %d tests%s%s%s in %s\n",
		execution.Total(),
		formatTestCount(len(execution.Skipped()), "skipped", ""),
		formatTestCount(len(execution.Failed()), "failure", "s"),
		formatTestCount(len(errors), "error", "s"),
		formatDurationAsSeconds(execution.Elapsed(), 3))

	writeTestCaseSummary(out, execution, formatSkipped)
	writeTestCaseSummary(out, execution, formatFailures)

	if len(errors) > 0 {
		fmt.Fprintln(out, "\n=== Errors")
	}
	for _, err := range errors {
		fmt.Fprintln(out, err)
	}

	return nil
}

func formatTestCount(count int, category string, pluralize string) string {
	switch count {
	case 0:
		return ""
	case 1:
	default:
		category += pluralize
	}
	return fmt.Sprintf(", %d %s", count, category)
}

func formatDurationAsSeconds(d time.Duration, precision int) string {
	return fmt.Sprintf("%.[2]*[1]fs", float64(d.Nanoseconds()/1000000)/1000, precision)
}

func writeTestCaseSummary(out io.Writer, execution *Execution, conf testCaseFormatConfig) {
	testCases := conf.getter(execution)
	if len(testCases) == 0 {
		return
	}
	fmt.Fprintln(out, "\n"+conf.header)
	for _, tc := range testCases {
		fmt.Fprintf(out, "%s %s %s (%s)\n",
			conf.prefix,
			relativePackagePath(tc.Package),
			tc.Test,
			formatDurationAsSeconds(tc.Elapsed, 2))
		for _, line := range execution.Output(tc.Package, tc.Test) {
			if isRunLine(line) || conf.filter(line) {
				continue
			}
			fmt.Fprint(out, line)
		}
		fmt.Fprintln(out)
	}
}

type testCaseFormatConfig struct {
	header string
	prefix string
	filter func(string) bool
	getter func(*Execution) []testCase
}

var formatFailures = testCaseFormatConfig{
	header: "=== Failures",
	prefix: "=== FAIL:",
	filter: func(line string) bool {
		return strings.HasPrefix(line, "--- FAIL: Test")
	},
	getter: func(execution *Execution) []testCase {
		return execution.Failed()
	},
}

var formatSkipped = testCaseFormatConfig{
	header: "=== Skipped",
	prefix: "=== SKIP:",
	filter: func(line string) bool {
		return strings.HasPrefix(line, "--- SKIP: Test")
	},
	getter: func(execution *Execution) []testCase {
		return execution.Skipped()
	},
}

func isRunLine(line string) bool {
	return strings.HasPrefix(line, "=== RUN   Test")
}

func relativePackagePath(pkgpath string) string {
	if pkgpath == pkgPathPrefix {
		return "."
	}
	return strings.TrimPrefix(pkgpath, pkgPathPrefix+"/")
}

// TODO: might not work on windows
func getPkgPathPrefix() string {
	cwd, _ := os.Getwd()
	gopaths := strings.Split(build.Default.GOPATH, string(filepath.ListSeparator))
	for _, gopath := range gopaths {
		gosrcpath := gopath + "/src/"
		if strings.HasPrefix(cwd, gosrcpath) {
			return strings.TrimPrefix(cwd, gosrcpath)
		}
	}
	return ""
}

var pkgPathPrefix = getPkgPathPrefix()

func NewEventHandler(format string) HandleEvent {
	switch format {
	case "debug":
		return debugFormat
	case "standard-verbose":
		return standardVerboseFormat
	case "standard-quiet":
		return standardQuietFormat
	case "dots":
		return dotsFormat
	case "short-verbose":
		return shortVerboseFormat
	case "short":
		return shortFormat
	default:
		return nil
	}
}
