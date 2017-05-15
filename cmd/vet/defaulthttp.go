// This file contains the defaulthttp checker.

package main

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"
)

func init() {
	register("defaulthttp",
		"check use of the http.DefaultClient in Chain packages",
		checkHttpCall,
		callExpr)
}

var prohibitedFunctions = map[string]bool{
	"net/http.Get":      true,
	"net/http.Head":     true,
	"net/http.Post":     true,
	"net/http.PostForm": true,
}

var prohibitedMethods = map[string]bool{
	"(*net/http.Client).Do":       true,
	"(*net/http.Client).Get":      true,
	"(*net/http.Client).Head":     true,
	"(*net/http.Client).Post":     true,
	"(*net/http.Client).PostForm": true,
}

var checkCmds = map[string]bool{
	"cored":   true,
	"corectl": true,
	"signerd": true,
}

// checkHttpCall checks that the chain packages do not use the
// http DefaultClient which does not include TLS settings.
//
// TODO(jackson): use pointer analysis to catch more uses:
// https://github.com/chain/chain/pull/1124#discussion_r115578973
func checkHttpCall(f *File, node ast.Node) {
	// Skip test files.
	if strings.HasSuffix(f.name, "_test.go") {
		return
	}
	// Skip commands besides the ones we explicitly want to check.
	if f.pkg.path == "main" && !checkCmds[filepath.Dir(f.name)] {
		return
	}

	call, ok := node.(*ast.CallExpr)
	if !ok {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	fun, ok := f.pkg.uses[sel.Sel].(*types.Func)
	if !ok {
		return
	}
	// Check for uses of the http convenience functions that implicitly use
	// the net/http.DefaultClient.
	if prohibitedFunctions[fun.FullName()] {
		f.Badf(call.Pos(), "use of %s", fun.FullName())
	}

	// It might still be an invocation of a method with the
	// http.DefaultClient as a receiver.
	if !prohibitedMethods[fun.FullName()] {
		return
	}
	clientSel, ok := sel.X.(*ast.SelectorExpr)
	if !ok {
		return
	}
	obj, ok := f.pkg.uses[clientSel.Sel]
	if !ok {
		return
	}
	if !isPackageLevel(obj) {
		return
	}
	if obj.Pkg().Path() != "net/http" || obj.Name() != "DefaultClient" {
		return
	}
	f.Badf(call.Pos(), "use of net/http.DefaultClient.%s", fun.Name())
}

func isPackageLevel(obj types.Object) bool {
	return obj.Pkg() != nil && obj.Pkg().Scope().Lookup(obj.Name()) == obj
}
