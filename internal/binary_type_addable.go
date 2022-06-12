package internal

import (
	"go/ast"
)

var binaryTypeGenerators = map[[2]string]typedGenerator[*ast.BinaryExpr]{}

func addBinaryTypeGenerator(first, second string, generator typedGenerator[*ast.BinaryExpr]) {
	binaryTypeGenerators[[2]string{first, second}] = generator
}

func addBinaryGenerators() {
	addGenerator(func(p *whereBodyParser, s *ast.BinaryExpr, args map[string]int) Addable {
		leftType, ok := p.exprType(s.X)
		if !ok {
			return nil
		}

		rightType, ok := p.exprType(s.Y)
		if !ok {
			return nil
		}

		key := [2]string{leftType, rightType}
		generator, ok := binaryTypeGenerators[key]
		if !ok {
			key = [2]string{rightType, leftType}
			generator, ok = binaryTypeGenerators[key]
			if !ok {
				return nil
			}
		}

		return generator(p, s, args)
	})
}

func stringBinaryTypeGenerator(p *whereBodyParser, s *ast.BinaryExpr, args map[string]int) Addable {
	return newBinary(
		p.exprToAddable(s.X, args),
		"||", // We can only concat strings, so all good.
		p.exprToAddable(s.Y, args),
	)
}
