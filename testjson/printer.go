package testjson

import (
	"bytes"
	"fmt"
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

func standardVerboseEvent(event TestEvent, _ *Execution) error {
	if event.Action == ActionOutput && event.Test != "" {
		fmt.Print(event.Output)
	}
	return nil
}

func standardQuietEvent(event TestEvent, _ *Execution) error {
	if event.Action == ActionOutput && event.Test == "" {
		fmt.Print(event.Output)
	}
	return nil
}

func summarizedPackageOutput(event TestEvent, exec *Execution) error {
	if event.Action == ActionOutput && event.Test != "" {
		pkg := exec.packages[event.Package]
		fmt.Printf("%s  Total=%d Failed=%d\n",
			event.Output, pkg.run, len(pkg.failed))
	}

	return nil
}

func PrintExecution(execution *Execution) error {
	fmt.Printf("%+v\n", execution)
	return nil
}

func NewHandler(formats []string) HandleEvent {
	if len(formats) == 0 {
		// TODO: better default
		return standardVerboseEvent
	}
	handlers := []HandleEvent{}
	for _, format := range formats {
		handler := handlersFromFormat(format)
		if handler == nil {
			// TODO: error?
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
		return standardVerboseEvent
	case "quiet":
		return standardQuietEvent
	case "summary":
		return summarizedPackageOutput
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
