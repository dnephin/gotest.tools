package main

import (
	"go/ast"
	"go/token"
	"reflect"
)

// walk possible values for assert.Equal() args:
// * a literal
// * a variable -> walk to type decl
// * a selector (struct field, struct method, import)
// * a function
func walk(node ast.Node) reflect.Kind {
	var result reflect.Kind
	visit := func(node ast.Node) bool {
		switch typed := node.(type) {
		case nil:

		case *ast.BasicLit:
			result = reflectKindFromToken(typed.Kind)
		case *ast.CompositeLit:
			result = reflectKindFromNodeType(typed.Type)

		case *ast.Ident:
			if typed.Obj == nil || typed.Obj.Decl == nil {
				debugf("missing obj.decl for %s", typed)
				return false
			}
			result = getKindFromIdentObjDecl(typed.Obj.Decl.(ast.Node))

		case *ast.SelectorExpr:
			result = walkSelectorExpr(typed)

		default:
			return true
		}
		return false
	}
	ast.Inspect(node, visit)
	return result
}

func reflectKindFromToken(k token.Token) reflect.Kind {
	switch k {
	case token.INT:
		return reflect.Int
	case token.FLOAT:
		return reflect.Float64
	case token.IMAG:
		return reflect.Float64 // ?
	case token.CHAR:
		return reflect.Uint8
	case token.STRING:
		return reflect.String
	}
	return reflect.Invalid
}

func reflectKindFromNodeType(node ast.Node) reflect.Kind {
	switch node.(type) {
	case *ast.ArrayType:
		return reflect.Array
	case *ast.MapType:
		return reflect.Map
	case *ast.ChanType:
		return reflect.Chan
	case *ast.FuncType:
		return reflect.Func
	case *ast.StructType:
		return reflect.Struct
	case *ast.InterfaceType:
		return reflect.Interface
	}
	return reflect.Invalid
}

func getKindFromIdentObjDecl(node ast.Node) reflect.Kind {
	var result reflect.Kind
	visit := func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.AssignStmt:
			if len(typed.Rhs) > 1 {
				debugf("multi assignment not yet support: %s", typed)
				return false
			}
			result = walk(typed.Rhs[0])
		case *ast.FuncDecl:
			if len(typed.Type.Results.List) > 1 {
				debugf("multi return values not yet support: %s", typed)
				return false
			}
			result = getKindFromFunctionReturn(typed.Type.Results.List[0].Type)
		default:
			return true
		}
		return false
	}
	ast.Inspect(node, visit)
	return result
}

// TODO: look at isSelectorFieldTypeTestingT
func walkSelectorExpr(selector *ast.SelectorExpr) reflect.Kind {
	// selector.X -> struct or import
	// selecter.Sel -> method/function or field
	return reflect.Invalid
}

func getKindFromFunctionReturn(node ast.Node) reflect.Kind {
	switch typed := node.(type) {
	case *ast.SelectorExpr:
		// type from other package most likely?
		return reflect.Invalid
	case *ast.Ident:
		switch typed.Name {
		case "bool":
			return reflect.Bool
		case "int", "int8", "int16", "int32", "int64":
			return reflect.Int
		case "uint", "uint8", "uint16", "uint32", "uint64":
			return reflect.Uint
		case "uintptr":
			return reflect.Uintptr
		case "float32", "float64":
			return reflect.Float64
		case "comlpex64", "complex128":
			return reflect.Complex128
		case "string":
			return reflect.String
		}
		debugf("unexpected identifier in function return: %s", typed.Name)
		return reflect.Invalid
	default:
		return reflectKindFromNodeType(node)
	}
}
