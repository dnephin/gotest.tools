package main

import (
	"fmt"
	"os"

	"github.com/gotestyourself/gotestyourself/testjson"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type options struct {
	quiet            bool
	hideFailureRecap bool
	hideRunSummary   bool
}

var errNonZeroExit = errors.New("")

func main() {
	name := os.Args[0]
	flags, opts := setupFlags(name)
	handleExitError(name, flags.Parse(os.Args[1:]))
	handleExitError(name, run(opts))
}

func setupFlags(name string) (*pflag.FlagSet, options) {
	opts := options{}
	flags := pflag.NewFlagSet(name, pflag.ContinueOnError)
	// TODO: set usage func to print more usage
	//flags.BoolVarP(&opts.quiet, "quiet", "q", false,
	//	"hide verbose test log")
	//flags.BoolVar(&opts.hideFailureRecap, "hide-failure-recap", false,
	//	"do not print a recap of test failures")
	//flags.BoolVar(&opts.hideRunSummary, "hide-summary", false,
	//	"do not print test summary")
	return flags, opts
}

func run(opts options) error {
	return testjson.Run()
}

//func getEchoWrite(quiet bool) io.Writer {
//	if quiet {
//		return ioutil.Discard
//	}
//	return os.Stdout
//}

func handleExitError(name string, err error) {
	switch {
	case err == nil:
		return
	case err == pflag.ErrHelp:
		os.Exit(0)
	case err == errNonZeroExit:
		os.Exit(1)
	default:
		fmt.Println(name + ": Error: " + err.Error())
		os.Exit(3)
	}
}
