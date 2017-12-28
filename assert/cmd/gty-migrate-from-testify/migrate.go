package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"log"
	"path"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
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
	pkgInfo     *loader.PackageInfo
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
		var newNode ast.Node
		switch typed := cursor.Node().(type) {
		case *ast.SelectorExpr:
			newNode = getReplacementTestingT(typed, migration.importNames)
		case *ast.CallExpr:
			newNode = getReplacementAssertion(typed, migration)
		}

		if newNode != nil {
			cursor.Replace(newNode)
		}
		return true
	}
}

func getReplacementTestingT(selector *ast.SelectorExpr, names importNames) ast.Node {
	xIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return nil
	}
	if selector.Sel.Name != "TestingT" || !names.matchesTestify(xIdent) {
		return nil
	}
	return &ast.SelectorExpr{
		X:   &ast.Ident{Name: names.assert},
		Sel: selector.Sel,
	}
}

func getReplacementAssertion(callExpr *ast.CallExpr, migration migration) ast.Node {
	tcall, ok := newCallFromNode(callExpr, migration)
	if !ok {
		return nil
	}
	if !migration.importNames.matchesTestify(tcall.xIdent) {
		return nil
	}
	if !callAcceptsTestingT(tcall, migration.importNames) {
		log.Printf("Skipping call, no testing.T as first arg %s", tcall.StringWithFileInfo())
		return nil
	}

	if len(tcall.expr.Args) < 2 {
		return convertTestifySingleArgCall(tcall, migration)
	}
	return convertTestifyAssertion(tcall, migration)
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
	if c.fileset.File(c.expr.Pos()) == nil {
		return fmt.Sprintf("%s at unknown file", c)
	}
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

func callAcceptsTestingT(tcall call, importNames importNames) bool {
	switch typed := tcall.expr.Args[0].(type) {
	case *ast.Ident:
		field, ok := typed.Obj.Decl.(*ast.Field)
		if !ok {
			return false
		}
		return isTestingTArg(field.Type, importNames)

	case *ast.SelectorExpr:
		return isSelectorFieldTypeTestingT(typed, importNames)
	}

	return false
}

func isTestingTArg(objType interface{}, importNames importNames) bool {
	switch typed := objType.(type) {
	case *ast.StarExpr:
		selector, ok := typed.X.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		// TODO: use import names
		return any(
			matchSelector(selector, "testing", "T"),
			matchSelector(selector, "testing", "B"),
			matchSelector(selector, "check", "C"))

	case *ast.SelectorExpr:
		return any(
			matchSelector(typed, importNames.testifyAssert, "TestingT"),
			matchSelector(typed, importNames.testifyRequire, "TestingT"))
	}
	return false
}

func matchSelector(selExpr *ast.SelectorExpr, ident, sel string) bool {
	xIdent, ok := selExpr.X.(*ast.Ident)
	if !ok {
		debugf("unexpected selector.X (%T) %s", selExpr.X, selExpr.X)
		return false
	}
	return selExpr.Sel.Name == sel && xIdent.Name == ident
}

func any(conds ...bool) bool {
	for _, cond := range conds {
		if cond {
			return true
		}
	}
	return false
}

// isSelectorFieldTypeTestingT examines the ast.SelectorExpr and returns
// the package.Type name for the field
// TODO: use walkSelectorExpr
func isSelectorFieldTypeTestingT(selector *ast.SelectorExpr, importNames importNames) bool {
	fieldName := selector.Sel.Name

	xIdent, ok := selector.X.(*ast.Ident)
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
				return isTestingTArg(field.Type, importNames)
			}
		}
	}

	return false
}

func convertTestifySingleArgCall(tcall call, migration migration) ast.Node {
	switch tcall.selExpr.Sel.Name {
	case "TestingT":
		// Already handled as SelectorExpr
		return nil
	case "New":
		// TODO: assert.New() - astutil.Apply() -> get selector.X from lhs of assignment
		log.Printf("not yet implemented: %s", tcall.StringWithFileInfo())
		return nil
	default:
		log.Printf("Skipping unknown %s", tcall.StringWithFileInfo())
		return nil
	}
}

func convertTestifyAssertion(tcall call, migration migration) ast.Node {
	imports := migration.importNames

	switch tcall.selExpr.Sel.Name {
	case "NoError", "NoErrorf":
		return convertNoError(tcall, imports)
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
	log.Printf("Skipping %s", tcall.StringWithFileInfo())
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

func newCallExprWithPosition(tcall call, imports importNames, args []ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.Ident{
				Name:    imports.assert,
				NamePos: tcall.xIdent.NamePos,
			},
			Sel: &ast.Ident{Name: imports.funcNameFromTestifyName(tcall.xIdent.Name)},
		},
		Args: args,
	}
}

func convertNoError(tcall call, imports importNames) ast.Node {
	// use assert.NoError() if there are no extra args
	if len(tcall.expr.Args) == 2 && tcall.xIdent.Name == imports.testifyRequire {
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name:    imports.assert,
					NamePos: tcall.xIdent.NamePos,
				},
				Sel: &ast.Ident{Name: "NilError"},
			},
			Args: tcall.expr.Args,
		}
	}
	return convertOneArgComparison(tcall, imports, "NilError")
}

func convertOneArgComparison(tcall call, imports importNames, cmpName string) ast.Node {
	return newCallExprWithPosition(tcall, imports,
		newCallExprArgs(
			tcall.testingT(),
			newCallExpr(imports.cmp, cmpName, []ast.Expr{tcall.expr.Args[1]}),
			tcall.extraArgs(2)...))
}

func convertTrue(tcall call, imports importNames) ast.Node {
	return newCallExprWithPosition(tcall, imports, tcall.expr.Args)
}

func convertFalse(tcall call, imports importNames) ast.Node {
	return newCallExprWithPosition(tcall, imports,
		newCallExprArgs(
			tcall.testingT(),
			&ast.UnaryExpr{Op: token.NOT, X: tcall.expr.Args[1]},
			tcall.extraArgs(2)...))
}

func convertEqual(tcall call, migration migration) ast.Node {
	imports := migration.importNames

	cmpEquals := convertTwoArgComparison(tcall, imports, "Equal")
	cmpCompare := convertTwoArgComparison(tcall, imports, "Compare")

	gotype := walkForType(migration.pkgInfo, tcall.expr.Args[1])
	if unknownType(gotype) {
		gotype = walkForType(migration.pkgInfo, tcall.expr.Args[2])
	}
	if unknownType(gotype) {
		return cmpCompare
	}

	switch gotype.Underlying().(type) {
	case *types.Basic:
		return cmpEquals
	default:
		return cmpCompare
	}
}

func unknownType(typ types.Type) bool {
	if typ == nil {
		return true
	}
	basic, ok := typ.(*types.Basic)
	return ok && basic.Kind() == types.Invalid
}

func convertTwoArgComparison(tcall call, imports importNames, cmpName string) ast.Node {
	return newCallExprWithPosition(tcall, imports,
		newCallExprArgs(
			tcall.testingT(),
			newCallExpr(imports.cmp, cmpName, tcall.expr.Args[1:3]),
			tcall.extraArgs(3)...))
}

func convertError(tcall call, imports importNames) ast.Node {
	return newCallExprWithPosition(tcall, imports,
		newCallExprArgs(
			tcall.testingT(),
			newCallExpr(
				imports.cmp,
				"ErrorContains",
				append(tcall.expr.Args[1:2], &ast.BasicLit{Kind: token.STRING, Value: `""`})),
			tcall.extraArgs(2)...))
}

func convertEmpty(tcall call, imports importNames) ast.Node {
	return newCallExprWithPosition(tcall, imports,
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
	return newCallExprWithPosition(tcall, imports,
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
	return newCallExprWithPosition(tcall, imports,
		newCallExprArgs(
			tcall.testingT(),
			&ast.BinaryExpr{X: lenExpr, Op: token.NEQ, Y: zeroExpr},
			tcall.extraArgs(2)...))
}
