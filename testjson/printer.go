package testjson

import (
	"bytes"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
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

func condensedFormat(event TestEvent, exec *Execution) (string, error) {
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
			exec.Output(event),
			relativePackagePath(event.Package),
			event.Test,
			event.ElapsedFormatted(),
		), nil
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
		return ".", nil
	case event.Action == ActionFail:
		return "x", nil
	}
	return "", nil
}

// TODO: show skipped
func PrintExecution(out io.Writer, execution *Execution) error {
	failed := execution.Failed()
	fmt.Fprintf(out, "\nDONE %d tests%s in %s\n",
		execution.Total(),
		formatFailedCount(len(failed), " with %d failure(s)"),
		formatDuration(execution.Elapsed()))

	// TODO: include package name in failure summary
	for _, failure := range failed {
		fmt.Fprintf(out, execution.Output(failure))
	}
	return nil
}

func formatFailedCount(count int, format string) string {
	if count == 0 {
		return ""
	}
	return fmt.Sprintf(format, count)
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.3fs", float64(d.Nanoseconds()/1000000)/1000)
}

// TODO: print failed test summary
// TODO: test data with: failed, skipped, empty package, parallel, subtests

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

func NewEventHandler(formats []string) HandleEvent {
	if len(formats) == 0 {
		// TODO: better default
		return standardVerboseFormat
	}
	handlers := []HandleEvent{}
	for _, format := range formats {
		handler := handlersFromFormat(format)
		if handler == nil {
			log.Printf("unknown format %s", format)
			continue
		}
		handlers = append(handlers, handler)
	}
	if len(handlers) == 1 {
		return handlers[0]
	}
	return multiHandler(handlers)
}

func handlersFromFormat(format string) HandleEvent {
	switch format {
	case "debug":
		return debugFormat
	case "standard":
		return standardVerboseFormat
	case "quiet":
		return standardQuietFormat
	case "dots":
		return dotsFormat
	case "condensed":
		return condensedFormat
	default:
		return nil
	}
}

// TODO: support multiple handlers without the extra buffer
func multiHandler(handlers []HandleEvent) HandleEvent {
	return func(event TestEvent, exec *Execution) (string, error) {
		errs := new(bytes.Buffer)
		out := new(bytes.Buffer)
		for _, handler := range handlers {
			lines, err := handler(event, exec)
			switch {
			case err != nil:
				errs.WriteString(err.Error() + "\n")
			default:
				out.WriteString(lines)
			}
		}
		if errs.Len() == 0 {
			return out.String(), nil
		}
		return "", errors.Errorf("some printers failed: %s", errs)
	}
}
