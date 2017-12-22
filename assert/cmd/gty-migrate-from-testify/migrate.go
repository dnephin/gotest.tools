package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"path"

	"golang.org/x/tools/go/ast/astutil"
)

const (
	pkgTestifyAssert       = "github.com/stretchr/testify/assert"
	pkgGopkgTestifyAssert  = "gopkg.in/stretchr/testify.v1/assert"
	pkgTestifyRequire      = "github.com/stretchr/testify/require"
	pkgGopkgTestifyRequire = "gopkg.in/stretchr/testify.v1/require"
	pkgAssert              = "github.com/gotestyourself/gotestyourself/assert"
	pkgCmp                 = "github.com/gotestyourself/gotestyourself/assert/cmp"
)

type migration struct {
	file        *ast.File
	fileset     *token.FileSet
	importNames importNames
}

func migrateFile(migration migration) {
	astutil.Apply(migration.file, nil, replaceCalls(migration))
	updateImports(migration)
}

func updateImports(migration migration) {
	for _, remove := range []string{
		pkgTestifyAssert,
		pkgTestifyRequire,
		pkgGopkgTestifyAssert,
		pkgGopkgTestifyRequire,
	} {
		astutil.DeleteImport(migration.fileset, migration.file, remove)
	}

	var alias string
	if migration.importNames.assert != path.Base(pkgAssert) {
		alias = migration.importNames.assert
	}
	astutil.AddNamedImport(migration.fileset, migration.file, alias, pkgAssert)

	if migration.importNames.cmp != path.Base(pkgCmp) {
		alias = migration.importNames.cmp
	}
	astutil.AddNamedImport(migration.fileset, migration.file, alias, pkgCmp)
}

func replaceCalls(migration migration) func(cursor *astutil.Cursor) bool {
	return func(cursor *astutil.Cursor) bool {
		tcall, ok := isTestifyCall(cursor.Node(), migration)
		if !ok {
			return true
		}

		if !isPackageCall(tcall) {
			// TODO: also convert assert.New() ...
			log.Printf("Skipping call without t %s", tcall.StringWithFileInfo())
			return true
		}

		newNode := convert(tcall, migration)
		if newNode == nil {
			log.Printf("Skipping %s", tcall.StringWithFileInfo())
			return true
		}

		cursor.Replace(newNode)
		return true
	}
}

func isTestifyCall(node ast.Node, migration migration) (call, bool) {
	c := call{}
	callExpr, ok := node.(*ast.CallExpr)
	if !ok {
		return c, false
	}
	selector, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return c, false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return c, false
	}
	if !migration.importNames.matchesTestify(ident) {
		return c, false
	}
	return call{
		fileset: migration.fileset,
		expr:    callExpr,
		xIdent:  ident,
		selExpr: selector,
	}, true
}

type call struct {
	fileset *token.FileSet
	expr    *ast.CallExpr
	xIdent  *ast.Ident
	selExpr *ast.SelectorExpr
}

func (c call) String() string {
	args := new(bytes.Buffer)
	format.Node(args, token.NewFileSet(), c.expr)
	return args.String()
}

func (c call) StringWithFileInfo() string {
	return fmt.Sprintf("%s at %s:%d", c,
		relativePath(c.fileset.File(c.expr.Pos()).Name()),
		c.fileset.Position(c.expr.Pos()).Line)
}

func (c call) testingT() ast.Expr {
	return c.expr.Args[0]
}

func (c call) extraArgs(index int) []ast.Expr {
	if len(c.expr.Args) <= index {
		return nil
	}
	return c.expr.Args[index:]
}

func isPackageCall(tcall call) bool {
	if len(tcall.expr.Args) < 2 {
		return false
	}

	switch typed := tcall.expr.Args[0].(type) {
	case *ast.Ident:
		if typed.Name == "t" {
			return true
		}
		field, ok := typed.Obj.Decl.(*ast.Field)
		if !ok {
			return false
		}
		return isTestingTStarExpr(field.Type)

	case *ast.SelectorExpr:
		return isSelectorFieldTypeTestingT(typed)
	}

	return false
}

func isTestingTStarExpr(objType interface{}) bool {
	starExpr, ok := objType.(*ast.StarExpr)
	if !ok {
		return false
	}

	selector, ok := starExpr.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	xIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	switch xIdent.Name {
	case "testing":
		return selector.Sel.Name == "T" || selector.Sel.Name == "B"
	case "check":
		return selector.Sel.Name == "C"
	}
	return false
}

// isSelectorFieldTypeTestingT examines the ast.SelectorExpr and returns
// the package.Type name for the field
func isSelectorFieldTypeTestingT(selectorExpr *ast.SelectorExpr) bool {
	fieldName := selectorExpr.Sel.Name

	xIdent, ok := selectorExpr.X.(*ast.Ident)
	if !ok {
		return false
	}

	xIdentField, ok := xIdent.Obj.Decl.(*ast.Field)
	if !ok {
		return false
	}

	objType, ok := xIdentField.Type.(*ast.Ident)
	if !ok {
		return false
	}

	typeSpec, ok := objType.Obj.Decl.(*ast.TypeSpec)
	if !ok {
		return false
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return false
	}

	for _, field := range structType.Fields.List {
		for _, nameIdent := range field.Names {
			if nameIdent.Name == fieldName {
				return isTestingTStarExpr(field.Type)
			}
		}
	}

	return false
}

func convert(tcall call, migration migration) ast.Node {
	imports := migration.importNames

	switch tcall.selExpr.Sel.Name {
	case "NoError", "NoErrorf":
		// use assert.NoError() if there are no extra args
		if len(tcall.expr.Args) == 2 && tcall.xIdent.Name == imports.testifyRequire {
			return newCallExpr(imports.assert, "NoError", tcall.expr.Args)
		}
		return convertOneArgComparison(tcall, imports, "NoError")
	case "True", "Truef":
		return convertTrue(tcall, imports)
	case "False", "Falsef":
		return convertFalse(tcall, imports)
	case "Equal", "Equalf", "Exactly", "Exactlyf", "EqualValues", "EqualValuesf":
		return convertEqual(tcall, migration)
	case "Contains", "Containsf":
		return convertTwoArgComparison(tcall, imports, "Contains")
	case "Len", "Lenf":
		return convertTwoArgComparison(tcall, imports, "Len")
	case "Panics", "Panicsf":
		return convertOneArgComparison(tcall, imports, "Panics")
	case "EqualError", "EqualErrorf":
		return convertTwoArgComparison(tcall, imports, "Error")
	case "Error", "Errorf":
		return convertError(tcall, imports)
	case "Empty", "Emptyf":
		return convertEmpty(tcall, imports)
	case "Nil", "Nilf":
		return convertOneArgComparison(tcall, imports, "Nil")
	case "NotNil", "NotNilf":
		return convertNegativeComparison(tcall, imports, &ast.Ident{Name: "nil"}, 2)
	case "NotEqual", "NotEqualf":
		return convertNegativeComparison(tcall, imports, tcall.expr.Args[2], 3)
	case "Fail", "Failf":
		return convertFail(tcall, "Error")
	case "FailNow", "FailNowf":
		return convertFail(tcall, "Fatal")
	case "NotEmpty", "NotEmptyf":
		return convertNotEmpty(tcall, imports)
	}
	return nil
}

func newCallExpr(x, sel string, args []ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: x},
			Sel: &ast.Ident{Name: sel},
		},
		Args: args,
	}
}

func newCallExprArgs(t ast.Expr, cmp ast.Expr, extra ...ast.Expr) []ast.Expr {
	return append(append([]ast.Expr{t}, cmp), extra...)
}

func convertOneArgComparison(tcall call, imports importNames, cmpName string) ast.Node {
	return newCallExpr(
		imports.assert,
		imports.funcNameFromTestifyName(tcall.xIdent.Name),
		newCallExprArgs(
			tcall.testingT(),
			newCallExpr(imports.cmp, cmpName, []ast.Expr{tcall.expr.Args[1]}),
			tcall.extraArgs(2)...))
}

func convertTrue(tcall call, imports importNames) ast.Node {
	return newCallExpr(
		imports.assert,
		imports.funcNameFromTestifyName(tcall.xIdent.Name),
		tcall.expr.Args)
}

func convertFalse(tcall call, imports importNames) ast.Node {
	return newCallExpr(
		imports.assert,
		imports.funcNameFromTestifyName(tcall.xIdent.Name),
		newCallExprArgs(
			tcall.testingT(),
			&ast.UnaryExpr{Op: token.NOT, X: tcall.expr.Args[1]},
			tcall.extraArgs(2)...))
}

func convertEqual(tcall call, migration migration) ast.Node {
	imports := migration.importNames

	cmpEquals := convertTwoArgComparison(tcall, imports, "Equal")
	cmpCompare := convertTwoArgComparison(tcall, imports, "Compare")

	switch typed := tcall.expr.Args[1].(type) {
	case *ast.BasicLit:
		return cmpEquals
	case *ast.CompositeLit:
		return cmpCompare
	case *ast.Ident:
		if typed.Obj == nil || typed.Obj.Decl == nil {
			return cmpCompare
		}
		switch declTyped := typed.Obj.Decl.(type) {
		case *ast.AssignStmt:
			switch declTyped.Rhs[0].(type) {
			case *ast.BasicLit:
				return cmpEquals
			case *ast.CompositeLit:
				return cmpCompare
			case *ast.CallExpr:
				// TODO: share with other CallExpr branch
			default:
				// TODO: struct type
			}
		}
	case *ast.CallExpr:
		// TODO:
	}

	return cmpCompare
}

func convertTwoArgComparison(tcall call, imports importNames, cmpName string) ast.Node {
	return newCallExpr(
		imports.assert,
		imports.funcNameFromTestifyName(tcall.xIdent.Name),
		newCallExprArgs(
			tcall.testingT(),
			newCallExpr(imports.cmp, cmpName, tcall.expr.Args[1:3]),
			tcall.extraArgs(3)...))
}

func convertError(tcall call, imports importNames) ast.Node {
	return newCallExpr(
		imports.assert,
		imports.funcNameFromTestifyName(tcall.xIdent.Name),
		newCallExprArgs(
			tcall.testingT(),
			newCallExpr(
				imports.cmp,
				"ErrorContains",
				append(tcall.expr.Args[1:2], &ast.BasicLit{Kind: token.STRING, Value: `""`})),
			tcall.extraArgs(2)...))
}

func convertEmpty(tcall call, imports importNames) ast.Node {
	return newCallExpr(
		imports.assert,
		imports.funcNameFromTestifyName(tcall.xIdent.Name),
		newCallExprArgs(
			tcall.testingT(),
			newCallExpr(
				imports.cmp,
				"Len",
				append(tcall.expr.Args[1:2], &ast.BasicLit{Kind: token.INT, Value: "0"})),
			tcall.extraArgs(2)...))
}

func convertNegativeComparison(
	tcall call,
	imports importNames,
	right ast.Expr,
	extra int,
) ast.Node {
	return newCallExpr(
		imports.assert,
		imports.funcNameFromTestifyName(tcall.xIdent.Name),
		newCallExprArgs(
			tcall.testingT(),
			&ast.BinaryExpr{X: tcall.expr.Args[1], Op: token.NEQ, Y: right},
			tcall.extraArgs(extra)...))
}

func convertFail(tcall call, selector string) ast.Node {
	extraArgs := tcall.extraArgs(1)
	if len(extraArgs) > 1 {
		selector = selector + "f"
	}

	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   tcall.testingT(),
			Sel: &ast.Ident{Name: selector},
		},
		Args: extraArgs,
	}
}

func convertNotEmpty(tcall call, imports importNames) ast.Node {
	lenExpr := &ast.CallExpr{
		Fun:  &ast.Ident{Name: "len"},
		Args: tcall.expr.Args[1:2],
	}
	zeroExpr := &ast.BasicLit{Kind: token.INT, Value: "0"}
	return newCallExpr(
		imports.assert,
		imports.funcNameFromTestifyName(tcall.xIdent.Name),
		newCallExprArgs(
			tcall.testingT(),
			&ast.BinaryExpr{X: lenExpr, Op: token.NEQ, Y: zeroExpr},
			tcall.extraArgs(2)...))
}
