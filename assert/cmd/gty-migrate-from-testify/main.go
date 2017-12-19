package main

import (
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"golang.org/x/tools/go/loader"
)

type options struct {
	dirs   []string
	dryRun bool
}

func main() {
	setupLogging()
	name := os.Args[0]
	flags, opts := setupFlags(name)
	handleExitError(name, flags.Parse(os.Args[1:]))
	opts.dirs = flags.Args()
	handleExitError(name, run(opts))
}

func setupLogging() {
	log.SetFlags(0)
}

func setupFlags(name string) (*pflag.FlagSet, *options) {
	opts := options{}
	flags := pflag.NewFlagSet(name, pflag.ContinueOnError)
	flags.BoolVar(&opts.dryRun, "dry-run", false, "don't write to file")
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
	// TODO: use Build.UseAllFiles=true, also defualt GOROOT doesn't work on arch
	conf := loader.Config{
		Fset:       token.NewFileSet(),
		ParserMode: parser.AllErrors | parser.ParseComments,
	}
	for _, dir := range opts.dirs {
		conf.ImportWithTests(dir)
	}
	prog, err := conf.Load()
	if err != nil {
		return errors.Wrapf(err, "failed to load source")
	}
	log.Printf("DEBUG: package count: %d", len(prog.Imported))

	for _, pkginfo := range prog.Imported {
		for _, astFile := range pkginfo.Files {
			file := conf.Fset.File(astFile.Pos())
			importNames := newImportNames(astFile.Imports)
			if !importNames.hasTestifyImports() {
				continue
			}

			m := migration{
				file:        astFile,
				fileset:     conf.Fset,
				importNames: importNames,
			}
			migrateFile(m)
			// TODO: maybe sort imports before write
			if !opts.dryRun {
				if err := writeFile(astFile, conf.Fset); err != nil {
					return errors.Wrapf(err, "failed to write file %s", file.Name())
				}
			}
		}
	}

	return nil
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
			continue
		case pkgTestifyRequire, pkgGopkgTestifyRequire:
			importNames.testifyRequire = identOrDefault(spec.Name, "require")
			continue
		}

		if importedAs(spec, "assert") {
			importNames.assert = "gtyassert"
		}
		if importedAs(spec, "cmp") {
			importNames.cmp = "gtycmp"
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

func writeFile(astFile *ast.File, fileset *token.FileSet) error {
	file := fileset.File(astFile.Pos())
	fh, err := os.OpenFile(file.Name(), os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return errors.Wrapf(err, "failed to open %s for writing", file.Name())
	}
	err = format.Node(fh, fileset, astFile)
	return errors.Wrapf(err, "failed to write source to %s", file.Name())
}
