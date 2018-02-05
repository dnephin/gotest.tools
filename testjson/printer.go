package testjson

import "fmt"

type Printer interface {
	PrintEvent(event TestEvent, output string) error
	PrintExecution(execution Execution) error
}

type defaultPrinter struct {
}

func (p *defaultPrinter) PrintEvent(event TestEvent, output string) error {
	fmt.Printf("%+v\n", event)
	return nil
}

func (p *defaultPrinter) PrintExecution(execution Execution) error {
	fmt.Printf("%+v\n", execution)
	return nil
}

// TODO: options
func NewPrinter() Printer {
	return &defaultPrinter{}
}
