package testjson

import (
	"bytes"
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func debugEvent(event TestEvent, _ *Execution) error {
	fmt.Printf("%s %s %s (%.3f) [%d] %s\n",
		event.Package,
		event.Test,
		event.Action,
		event.Elapsed,
		event.Time.Unix(),
		event.Output)
	return nil
}

// go test -v
func standardVerboseFormat(event TestEvent, _ *Execution) error {
	if event.Action == ActionOutput && !event.PackageEvent() {
		fmt.Print(event.Output)
	}
	return nil
}

// go test
func standardQuietFormat(event TestEvent, _ *Execution) error {
	if isPackageEndEvent(event) {
		fmt.Print(event.Output)
	}
	return nil
}

// TODO: handler that only shows output of failed tests
//func allTestsFormat(event TestEvent, _ *Execution) error {
//	if event.Test == "" {
//		return nil
//	}
//
//	switch event.Action {
//	case ActionRun:
//
//	}
//	fmt.Printf("%s %s",
//		relativePackagePath(event.Package),
//		event.Test)
//	}
//	return nil
//}

func summaryPackageFormat(event TestEvent, exec *Execution) error {
	if !isPackageEndEvent(event) {
		return nil
	}

	pkg := exec.packages[event.Package]
	switch {
	case pkg.run == 0:
		fmt.Printf("%s [no tests]\n", relativePackagePath(event.Package))
	default:
		fmt.Printf("%s [%d tests%s]\n",
			relativePackagePath(event.Package),
			pkg.run, // TODO: count can be off because of parallel runs?
			formatFailedCount(len(pkg.failed), " %d failed"))
	}
	return nil
}

func formatFailedCount(count int, format string) string {
	if count == 0 {
		return ""
	}
	return fmt.Sprintf(format, count)
}

func isPackageEndEvent(event TestEvent) bool {
	if event.Action != ActionOutput || !event.PackageEvent() {
		return false
	}
	return strings.HasPrefix(event.Output, "ok ") || strings.HasPrefix(event.Output, "? ")
}

// TODO: fix newlines
func testDotsFormat(event TestEvent, exec *Execution) error {
	pkg := exec.Package(event)

	switch {
	case event.Action == ActionRun && pkg.run == 1:
		fmt.Print(relativePackagePath(event.Package) + " ")
	case event.Action == ActionPass:
		fmt.Print(".")
	case event.Action == ActionFail:
		fmt.Print("x")
	case isPackageEndEvent(event):
		fmt.Println()
	}
	return nil
}

func PrintExecution(execution *Execution) error {
	// TODO: only show failed if != 0
	// TODO: show skipped
	fmt.Printf("Summary: Total %d Failed %d (%v)\n",
		execution.Total(),
		len(execution.Failed()),
		execution.Elapsed())
	return nil
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
		return debugEvent
	case "standard":
		return standardVerboseFormat
	case "quiet":
		return standardQuietFormat
	case "summary":
		return summaryPackageFormat
	case "dots":
		return testDotsFormat
	default:
		return nil
	}
}

func multiHandler(handlers []HandleEvent) HandleEvent {
	return func(event TestEvent, exec *Execution) error {
		errs := new(bytes.Buffer)
		for _, handler := range handlers {
			if err := handler(event, exec); err != nil {
				errs.WriteString(err.Error() + "\n")
			}
		}
		if errs.Len() == 0 {
			return nil
		}
		return errors.Errorf("some printers failed: %s", errs)
	}
}
