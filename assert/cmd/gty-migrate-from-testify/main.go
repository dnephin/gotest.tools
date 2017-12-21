package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"golang.org/x/tools/imports"
)

type options struct {
	dirs   []string
	dryRun bool
	debug  bool
}

func main() {
	name := os.Args[0]
	flags, opts := setupFlags(name)
	handleExitError(name, flags.Parse(os.Args[1:]))
	setupLogging(opts)
	opts.dirs = flags.Args()
	handleExitError(name, run(opts))
}

func setupLogging(opts *options) {
	log.SetFlags(0)
	enableDebug = opts.debug
}

var enableDebug = false

func debugf(msg string, args ...interface{}) {
	if enableDebug {
		log.Printf("DEBUG: "+msg, args...)
	}
}

func setupFlags(name string) (*pflag.FlagSet, *options) {
	opts := options{}
	flags := pflag.NewFlagSet(name, pflag.ContinueOnError)
	flags.BoolVar(&opts.dryRun, "dry-run", false, "don't write to file")
	flags.BoolVar(&opts.debug, "debug", false, "enable debug logging")
	// TODO: set usage func to print more usage
	return flags, &opts
}

func handleExitError(name string, err error) {
	switch {
	case err == nil:
		return
	case err == pflag.ErrHelp:
		os.Exit(0)
	default:
		log.Println(name + ": Error: " + err.Error())
		os.Exit(3)
	}
}

func run(opts *options) error {
	debugf("package count: %d", len(opts.dirs))

	fileset := token.NewFileSet()
	for _, dir := range opts.dirs {
		pkgs, err := parser.ParseDir(fileset, dir, nil, parser.AllErrors|parser.ParseComments)
		if err != nil {
			return errors.Wrapf(err, "failed to parse %s", dir)
		}
		for _, pkg := range pkgs {
			for _, astFile := range pkg.Files {
				absFilename := fileset.File(astFile.Pos()).Name()
				filename := relativePath(absFilename)
				importNames := newImportNames(astFile.Imports)
				if !importNames.hasTestifyImports() {
					debugf("skipping file %s, no imports", filename)
					continue
				}

				debugf("migrating %s with imports: %#v", filename, importNames)
				m := migration{
					file:        astFile,
					fileset:     fileset,
					importNames: importNames,
				}
				migrateFile(m)
				if opts.dryRun {
					continue
				}

				raw, err := formatFile(astFile, fileset)
				if err != nil {
					return errors.Wrapf(err, "failed to format %s", filename)
				}

				if err := ioutil.WriteFile(absFilename, raw, 0); err != nil {
					return errors.Wrapf(err, "failed to write file %s", filename)
				}
			}
		}
	}

	return nil
}

func relativePath(p string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return p
	}
	rel, err := filepath.Rel(cwd, p)
	if err != nil {
		return p
	}
	return rel
}

type importNames struct {
	testifyAssert  string
	testifyRequire string
	assert         string
	cmp            string
}

func (p importNames) hasTestifyImports() bool {
	return p.testifyAssert != "" || p.testifyRequire != ""
}

func (p importNames) matchesTestify(ident *ast.Ident) bool {
	return ident.Name == p.testifyAssert || ident.Name == p.testifyRequire
}

func (p importNames) funcNameFromTestifyName(name string) string {
	if name == p.testifyAssert {
		return "Check"
	}
	return "Assert"
}

func newImportNames(imports []*ast.ImportSpec) importNames {
	importNames := importNames{
		assert: path.Base(pkgAssert),
		cmp:    path.Base(pkgCmp),
	}
	for _, spec := range imports {
		switch strings.Trim(spec.Path.Value, `"`) {
		case pkgTestifyAssert, pkgGopkgTestifyAssert:
			importNames.testifyAssert = identOrDefault(spec.Name, "assert")
		case pkgTestifyRequire, pkgGopkgTestifyRequire:
			importNames.testifyRequire = identOrDefault(spec.Name, "require")
		default:
			if importedAs(spec, "assert") {
				importNames.assert = "gtyassert"
			}
			if importedAs(spec, "cmp") {
				importNames.cmp = "gtycmp"
			}
		}
	}
	return importNames
}

func importedAs(spec *ast.ImportSpec, pkg string) bool {
	if path.Base(strings.Trim(spec.Path.Value, `""`)) == pkg {
		return true
	}
	return spec.Name != nil && spec.Name.Name == pkg
}

func identOrDefault(ident *ast.Ident, def string) string {
	if ident != nil {
		return ident.Name
	}
	return def
}

func formatFile(astFile *ast.File, fileset *token.FileSet) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := format.Node(buf, fileset, astFile)
	if err != nil {
		return nil, err
	}
	return imports.Process(fileset.File(astFile.Pos()).Name(), buf.Bytes(), nil)
}
