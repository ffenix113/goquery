package internal

import (
	"go/ast"
	"go/types"
)

func addConstGenerators() {
	// consts in the form of
	//	const <name> = <value>
	//	const <name> = <package>.<name>
	// for example
	//	const val = 55
	//	const val = math.MaxInt
	addGenerator(func(p *whereBodyParser, s *ast.Ident, args map[string]int) Addable {
		if s.Obj == nil || s.Obj.Kind != ast.Con {
			objVal := p.c.TypeInfo.ObjectOf(s)
			if _, ok := objVal.(*types.Const); ok {
				// If constant is just an Ident - it means that
				// it is defined in the same package
				return NewSimple(param, raw(p.c.exprName(s)))
			}

			return nil
		}

		switch constVal := s.Obj.Decl.(*ast.ValueSpec).Values[0].(type) {
		case *ast.BasicLit:
			// Just use a direct value for now.
			// TODO: support package constants by name, not value.
			return NewSimple(param, getArg(constVal))
		case *ast.SelectorExpr:
			// If it is a selector - use its name.
			// Import for it will be added on template execution.
			return NewSimple(param, raw(p.c.exprName(constVal)))
		}

		return nil
	})

	// consts in the form of
	//	<package>.<name>
	// for example
	//	math.MaxInt
	addGenerator(func(p *whereBodyParser, s *ast.SelectorExpr, args map[string]int) Addable {
		obj := p.c.TypeInfo.ObjectOf(s.Sel)
		if obj == nil {
			return nil
		}

		if _, ok := obj.(*types.Const); ok {
			// Same as in the above const evaluation - if it is a const
			// from another package - it will be imported on the
			// template execution.
			return NewSimple(param, raw(p.c.exprName(s)))
		}

		return nil
	})
}
