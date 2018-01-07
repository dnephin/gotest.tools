package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
)

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
		expr:    updateCallExprForMissingT(*callExpr),
		xIdent:  ident,
		selExpr: selector,
		assert:  migration.importNames.funcNameFromTestifyName(ident.Name),
	}, true
}

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
