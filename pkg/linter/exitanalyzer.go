// linter - пакет с анализатором проверяющим использование os.Exit .
package linter

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Переменная для харнения анализатора ExitAnalyzer.
var ExitAnalyzer = &analysis.Analyzer{
	Name:     "exitcheck",
	Doc:      "check for use Exit in main",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Run - функция для запуска анализатора.
func run(pass *analysis.Pass) (interface{}, error) {
	isMain := func(x *ast.File) bool {
		return x.Name.Name == "main"
	}

	isMainFunc := func(x *ast.FuncDecl) bool {
		return x.Name.Name == "main"
	}

	isExit := func(x *ast.SelectorExpr, isMain bool) bool {
		if !isMain || x.X == nil {
			return false
		}
		ident, ok := x.X.(*ast.Ident)
		if !ok {
			return false
		}

		if ident.Name == "os" && x.Sel.Name == "Exit" {
			pass.Reportf(ident.NamePos, "Exit called in main package")
			return true
		}
		return false
	}

	i := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.SelectorExpr)(nil),
	}
	mainInspect := false
	i.Preorder(nodeFilter, func(n ast.Node) {
		switch x := n.(type) {
		case *ast.File:
			if !isMain(x) {
				return
			}
		case *ast.FuncDecl:
			f := isMainFunc(x)
			if mainInspect && !f {
				mainInspect = false
				return
			}
			mainInspect = f
		case *ast.SelectorExpr:
			if isExit(x, mainInspect) {
				return
			}
		}
	})
	return nil, nil
}
