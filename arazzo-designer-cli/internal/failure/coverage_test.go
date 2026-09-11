package failure_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// packagesThatFail are the source directories where a runtime failure can be constructed.
var packagesThatFail = []string{"../runner", "../runner/executor"}

// A class is only useful if EVERY failure carries one. Otherwise an empty class means two different
// things - "this situation has no name yet" and "whoever added this path forgot" - and a client can
// never tell which, so it has to fall back to matching prose exactly as it did before classes
// existed.
//
// These tests read the source rather than running it, because behaviour tests can only cover the
// paths someone thought to provoke. A new failure path added next year is caught here on the day it
// is written, with no test needing to be remembered.
//
// If one of these fails, the fix is to pass a failure.Class at the reported line - or, if the
// situation genuinely is not in the vocabulary, to add a class for it in failure.go. Silencing it by
// passing an empty class defeats the point.

// Every createFailureResult call must supply a class. The parameter is variadic, so the compiler
// cannot require it; this does.
func TestEveryCreateFailureResultCallSuppliesAClass(t *testing.T) {
	forEachSourceFile(t, func(t *testing.T, fset *token.FileSet, file *ast.File, path string) {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "createFailureResult" {
				return true
			}
			// (stepID, step, state, errMsg) plus at least one class.
			if len(call.Args) < 5 {
				t.Errorf("%s: createFailureResult call supplies no failure class", pos(fset, call.Pos()))
			}
			return true
		})
	})
}

// A failure result built as a struct literal bypasses createFailureResult entirely, so it needs the
// same rule: if it reports an Error, it must report an ErrorClass beside it.
func TestEveryFailureResultLiteralSetsAClass(t *testing.T) {
	carriesFailure := map[string]bool{"StepResult": true, "WorkflowExecutionResult": true}

	forEachSourceFile(t, func(t *testing.T, fset *token.FileSet, file *ast.File, path string) {
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			sel, ok := lit.Type.(*ast.SelectorExpr)
			if !ok || !carriesFailure[sel.Sel.Name] {
				return true
			}
			var hasError, hasClass bool
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch key.Name {
				case "Error":
					hasError = true
				case "ErrorClass":
					hasClass = true
				}
			}
			if hasError && !hasClass {
				t.Errorf("%s: %s literal reports an Error with no ErrorClass beside it",
					pos(fset, lit.Pos()), sel.Sel.Name)
			}
			return true
		})
	})
}

func forEachSourceFile(t *testing.T, fn func(*testing.T, *token.FileSet, *ast.File, string)) {
	t.Helper()
	for _, dir := range packagesThatFail {
		paths, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatalf("glob %s: %v", dir, err)
		}
		if len(paths) == 0 {
			t.Fatalf("no source files found in %s - has the layout moved?", dir)
		}
		for _, path := range paths {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			fn(t, fset, file, path)
		}
	}
}

func pos(fset *token.FileSet, p token.Pos) string {
	at := fset.Position(p)
	return filepath.ToSlash(at.Filename) + ":" + itoa(at.Line)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
