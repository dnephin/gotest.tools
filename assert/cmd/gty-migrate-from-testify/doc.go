/*

Command gty-migration-from-testify migrates one or more packages from
testify/assert and testify/require to gotestyourself/assert.

To run on all packages (including external test packages) use:

	go list \
		-f '{{.ImportPath}} {{if .XTestGoFiles}}{{"\n"}}{{.ImportPath}}_test{{end}}' \
		./... | xargs gty-migrate-from-testify

*/

package main
