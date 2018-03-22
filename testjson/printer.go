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
	// TODO: include elapsed time in package events?
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

// TODO: show skipped
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
		// skip: ↷
	}
	return "", nil
}

// TODO: show skipped
func PrintExecution(out io.Writer, execution *Execution) error {
	failed := execution.Failed()
	errors := execution.Errors()
	fmt.Fprintf(out, "\nDONE %d tests%s%s in %s\n",
		execution.Total(),
		formatTestCount(len(failed), "failure"),
		formatTestCount(len(errors), "error"),
		formatDurationAsSeconds(execution.Elapsed(), 3))

	if len(failed) > 0 {
		fmt.Fprintln(out, "\n=== Failures")
	}
	for _, failure := range failed {
		writeFailureSummary(out, failure, execution.Output(failure.Package, failure.Test))
	}

	if len(errors) > 0 {
		fmt.Fprintln(out, "\n=== Errors")
	}
	for _, err := range errors {
		fmt.Fprintln(out, err)
	}

	return nil
}

func formatTestCount(count int, category string) string {
	switch count {
	case 0:
		return ""
	case 1:
	default:
		category += "s"
	}
	return fmt.Sprintf(", %d %s", count, category)
}

func formatDurationAsSeconds(d time.Duration, precision int) string {
	return fmt.Sprintf("%.[2]*[1]fs", float64(d.Nanoseconds()/1000000)/1000, precision)
}

func writeFailureSummary(out io.Writer, tc testCase, failure []string) {
	fmt.Fprintf(out, "=== FAIL: %s %s (%s)\n",
		relativePackagePath(tc.Package),
		tc.Test,
		formatDurationAsSeconds(tc.Elapsed, 2))
	for _, line := range failure[1:] {
		if isFailLine(line) {
			continue
		}
		fmt.Fprint(out, line)
	}
	fmt.Fprintln(out)
}

func isFailLine(line string) bool {
	return strings.HasPrefix(line, "--- FAIL: Test")
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
