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
	fmt.Printf("%s:%s %s (%.3f) [%d] %s\n",
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
	if event.Action == ActionOutput && event.Test != "" {
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

func summaryPackageFormat(event TestEvent, exec *Execution) error {
	if !isPackageEndEvent(event) {
		return nil
	}

	pkg := exec.packages[event.Package]
	switch {
	case pkg.run == 0:
		fmt.Printf("%s [no test files]\n", event.Package)
	default:
		fmt.Printf("%s Total=%d Failed=%d\n",
			event.Package, pkg.run, len(pkg.failed))
	}
	return nil
}

func isPackageEndEvent(event TestEvent) bool {
	if event.Action != ActionOutput || event.Test != "" {
		return false
	}
	return strings.HasPrefix(event.Output, "ok ") || strings.HasPrefix(event.Output, "? ")
}

// TODO: fix
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
	return strings.TrimPrefix(pkgpath, pkgPathPrefix)
}

// TODO: might not work on windows
func getPkgPathPrefix() string {
	cwd, _ := os.Getwd()
	gopaths := strings.Split(build.Default.GOPATH, string(filepath.ListSeparator))
	for _, gopath := range gopaths {
		gosrcpath := gopath + "/src/"
		if strings.HasPrefix(cwd, gosrcpath) {
			return strings.TrimPrefix(cwd, gosrcpath) + "/"
		}
	}
	return ""
}

var pkgPathPrefix = getPkgPathPrefix()

func NewHandler(formats []string) HandleEvent {
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
