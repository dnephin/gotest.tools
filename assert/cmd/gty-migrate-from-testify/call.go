package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
)

type call struct {
	fileset *token.FileSet
	expr    *ast.CallExpr
	xIdent  *ast.Ident
	selExpr *ast.SelectorExpr
	assert  string
}

func (c call) String() string {
	args := new(bytes.Buffer)
	format.Node(args, token.NewFileSet(), c.expr)
	return args.String()
}

func (c call) StringWithFileInfo() string {
	if c.fileset.File(c.expr.Pos()) == nil {
		return fmt.Sprintf("%s at unknown file", c)
	}
	return fmt.Sprintf("%s at %s:%d", c,
		relativePath(c.fileset.File(c.expr.Pos()).Name()),
		c.fileset.Position(c.expr.Pos()).Line)
}

func (c call) testingT() ast.Expr {
	if len(c.expr.Args) == 0 {
		return nil
	}
	return c.expr.Args[0]
}

func (c call) extraArgs(index int) []ast.Expr {
	if len(c.expr.Args) <= index {
		return nil
	}
	return c.expr.Args[index:]
}

func (c call) args(from, to int) []ast.Expr {
	return c.expr.Args[from:to]
}

func (c call) arg(index int) ast.Expr {
	return c.expr.Args[index]
}

func (c call) assertionName() string {
	if c.assert == "" {
		log.Printf("WARN: unknown assertion name for %s", c)
		return "Assert"
	}
	return c.assert
}

func newCallFromNode(callExpr *ast.CallExpr, migration migration) (call, bool) {
	c := call{}
	selector, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return c, false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return c, false
	}

	return call{
		fileset: migration.fileset,
		expr:    updateCallExprForMissingT(*callExpr, migration),
		xIdent:  ident,
		selExpr: selector,
		assert:  migration.importNames.funcNameFromTestifyName(ident.Name),
	}, true
}

// TODO: move call of updateCallExprForMissingT to this function
func newTestifyCallFromNode(callExpr *ast.CallExpr, migration migration) (call, bool) {
	tcall, ok := newCallFromNode(callExpr, migration)
	if !ok {
		return tcall, ok
	}

	assertionName, ok := isTestifyCall(tcall, migration)
	if !ok {
		return tcall, false
	}
	// TODO: not clean
	tcall.assert = assertionName
	return tcall, true
}

const (
	pkgGocheck      = "github.com/go-check/check"
	pkgGopkgGocheck = "gopkg.in/check.v1"
)

// update calls that use assert := assert.New(t), but make a copy of the node
// so that unrelated calls are not modified.
// TODO: check for testify.Assertions as type, instead of first arg
func updateCallExprForMissingT(callExpr ast.CallExpr, migration migration) *ast.CallExpr {
	update := func() *ast.CallExpr {
		// TODO: lookup proper ident for t
		callExpr.Args = append([]ast.Expr{&ast.Ident{Name: "t"}}, callExpr.Args...)
		return &callExpr
	}

	if len(callExpr.Args) < 1 {
		return &callExpr
	}

	gotype := walkForType(migration.pkgInfo, callExpr.Args[0])
	if gotype == nil {
		return update()
	}
	switch gotype.String() {
	case "*testing.T", "*testing.B":
		return &callExpr
	case pkgTestifyAssert + ".TestingT", pkgGopkgTestifyAssert + ".TestingT":
		return &callExpr
	case pkgTestifyRequire + ".TestingT", pkgGopkgTestifyRequire + ".TestingT":
		return &callExpr
	case "*" + pkgGopkgGocheck + ".C", "*" + pkgGocheck + ".C":
		return &callExpr
	}

	return update()
}

// TODO: check if type is import declaration instead of assuming import is always
// correct
func isTestifyCall(tcall call, migration migration) (string, bool) {
	fromPkgName := func() (string, bool) {
		if migration.importNames.matchesTestify(tcall.xIdent) {
			return tcall.assertionName(), true
		}
		return "", false
	}

	//gotype := walkForType(migration.pkgInfo, tcall.xIdent)
	//fmt.Printf("Type is %s\n", gotype)

	if tcall.xIdent.Obj == nil {
		return fromPkgName()
	}

	assignStmt, ok := tcall.xIdent.Obj.Decl.(*ast.AssignStmt)
	if !ok {
		return fromPkgName()
	}

	if assertionName, ok := isAssignmentFromAssertNew(assignStmt, migration); ok {
		return assertionName, ok
	}
	return fromPkgName()
}
