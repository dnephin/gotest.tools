package main

import (
	"fmt"
	"go/build"
	"path"
	"strings"
)

var allTestingTPkgs = append(
	allTestifyPks,
	"github.com/go-check/check",
	"gopkg.in/check.v1",
)

func findPackage(
	bctx *build.Context,
	importPath string,
	fromDir string,
	mode build.ImportMode,
) (*build.Package, error) {
	pkg, err := bctx.Import(importPath, fromDir, mode)
	if err == nil {
		return pkg, err
	}

	for _, pkgName := range allTestingTPkgs {
		if pkgName == importPath {
			return importStubPackage(pkgName)
		}
	}

	fmt.Printf("FindPackage(%s, %s, %v) => %s\n", importPath, fromDir, mode, err)
	return pkg, err
}

func importStubPackage(pkgName string) (*build.Package, error) {
	// TODO: fix paths
	return &build.Package{
		Dir:        "/home/daniel/pers/code/gotestyourself/tmp/pkgs",
		Name:       strings.TrimSuffix(path.Base(pkgName), ".v1"),
		ImportPath: pkgName,
		GoFiles:    []string{"fixtures.go"},
	}, nil
}
