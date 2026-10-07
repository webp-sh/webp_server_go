package main_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFiberAppIsBuiltAfterLoadConfig guards the startup order in webp-server.go.
// fiber.New copies ReadBufferSize, Concurrency, and DisableKeepalive by value.
// config.json currently matches NewWebPConfig defaults, so a running server cannot
// show the mistake; the call order in source is what has to stay intact.
func TestFiberAppIsBuiltAfterLoadConfig(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "webp-server.go", nil, 0)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		ast.Inspect(gen, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if ok && callName(call) == "fiber.New" {
				t.Fatal("fiber.New is in a package-level variable; move it to after config.LoadConfig()")
			}
			return true
		})
	}

	initFn := findFunc(file, "init")
	require.NotNil(t, initFn, "init() must create the Fiber app after loading config")

	var calls []string
	var fiberNew *ast.CallExpr
	ast.Inspect(initFn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		name := callName(call)
		switch name {
		case "config.LoadConfig", "fiber.New", "app.Use", "setupLogger":
			calls = append(calls, name)
			if name == "fiber.New" && fiberNew == nil {
				fiberNew = call
			}
		}
		return true
	})

	loadAt := indexOf(calls, "config.LoadConfig")
	newAt := indexOf(calls, "fiber.New")
	if loadAt < 0 {
		t.Fatal("init() must call config.LoadConfig()")
	}
	if newAt < 0 || newAt < loadAt {
		t.Fatal("init() must call fiber.New after config.LoadConfig(), otherwise Fiber keeps the default ReadBufferSize, Concurrency, and DisableKeepalive")
	}

	for _, name := range calls[:newAt] {
		require.NotContains(t, []string{"app.Use", "setupLogger"}, name, "%s runs before fiber.New; app is still nil", name)
	}

	require.NotNil(t, fiberNew)
	fields := compositeFields(fiberNew)
	require.Equal(t, "config.Config.ReadBufferSize", fields["ReadBufferSize"])
	require.Equal(t, "config.Config.Concurrency", fields["Concurrency"])
	require.Equal(t, "config.Config.DisableKeepalive", fields["DisableKeepalive"])
}

func findFunc(file *ast.File, name string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == name {
			return fn
		}
	}
	return nil
}

func indexOf(items []string, target string) int {
	for i, item := range items {
		if item == target {
			return i
		}
	}
	return -1
}

func callName(call *ast.CallExpr) string {
	return exprName(call.Fun)
}

func exprName(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprName(v.X) + "." + v.Sel.Name
	default:
		return ""
	}
}

func compositeFields(call *ast.CallExpr) map[string]string {
	fields := map[string]string{}
	if len(call.Args) == 0 {
		return fields
	}
	lit, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return fields
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		fields[key.Name] = exprName(kv.Value)
	}
	return fields
}
