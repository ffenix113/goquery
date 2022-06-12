package internal

import (
	"go/ast"
)

var packageIdentGenerators = map[string]map[string]typedGenerator[*ast.SelectorExpr]{}

func addPackageIdentGenerator(packageName, identName string, generator typedGenerator[*ast.SelectorExpr]) {
	mp := packageIdentGenerators[packageName]
	if mp == nil {
		mp = map[string]typedGenerator[*ast.SelectorExpr]{}
		packageIdentGenerators[packageName] = mp
	}

	mp[identName] = generator
}

func addPackageIdentGenerators() {
	addGenerator(func(p *whereBodyParser, s *ast.SelectorExpr, args map[string]int) Addable {
		ident, ok := s.X.(*ast.Ident)
		if !ok || ident.Obj != nil {
			return nil
		}

		identGenerators, ok := packageIdentGenerators[ident.Name]
		if !ok {
			return nil
		}

		generator, ok := identGenerators[s.Sel.Name]
		if !ok {
			return nil
		}

		return generator(p, s, args)
	})
}

func timeIdentsGenerator(_ *whereBodyParser, s *ast.SelectorExpr, _ map[string]int) Addable {
	var duration string
	switch s.Sel.Name {
	case "Microsecond":
		duration = "1 microsecond"
	case "Millisecond":
		duration = "1 millisecond"
	case "Second":
		duration = "1 second"
	case "Minute":
		duration = "1 minute"
	case "Hour":
		duration = "1 hour"
	}

	return NewSimple("INTERVAL '" + duration + "'")
}
