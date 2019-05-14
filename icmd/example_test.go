package icmd_test

import (
	"time"

	"gotest.tools/icmd"
	"gotest.tools/internal/examplet"
)

var t = examplet.T

func ExampleRunCommand() {
	result := icmd.RunCommand("bash", "-c", "echo all good")
	result.Assert(t, icmd.Success)
}

func ExampleRunCmd() {
	result := icmd.RunCmd(icmd.Command("cat", "/does/not/exist"))
	result.Assert(t, icmd.Expected{
		ExitCode: 1,
		Err:      "cat: /does/not/exist: No such file or directory",
	})
}

func ExampleRunCmd_failure() {
	result := icmd.RunCmd(icmd.Command("cat", "/does/not/exist"))
	result.Assert(t, icmd.Success)
	// Output:
	// assertion failed: 
	// Command:  cat /does/not/exist
	// ExitCode: 1
	// Error:    exit status 1
	// Stdout:
	// Stderr:   cat: can't open '/does/not/exist': No such file or directory
	// 
	// 
	// Failures:
	// ExitCode was 1 expected 0
	// Expected no error
}

func ExampleWaitOnCmd() {
	result := icmd.RunCmd(icmd.Command("sleep", "200"))
	result = icmd.WaitOnCmd(2*time.Second, result)
	result.Assert(t, icmd.Success)
}
