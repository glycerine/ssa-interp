// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa2

// This file implements the CREATE phase of SSA construction.
// See builder.go for explanation.

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"strings"

	"code.google.com/p/go.tools/go/types"
	"code.google.com/p/go.tools/importer"
)

// BuilderMode is a bitmask of options for diagnostics and checking.
type BuilderMode uint

const (
	LogPackages          BuilderMode = 1 << iota // Dump package inventory to stderr
	LogFunctions                                 // Dump function SSA code to stderr
	LogSource                                    // Show source locations as SSA builder progresses
	SanityCheckFunctions                         // Perform sanity checking of function bodies
	NaiveForm                                    // Build naïve SSA form: don't replace local loads/stores with registers
	BuildSerially                                // Build packages serially, not in parallel.
	DebugInfo                                    // Include DebugRef Globally
)

// NewProgram returns a new SSA Program initially containing no
// packages.
//
// fset specifies the mapping from token positions to source location
// that will be used by all ASTs of this program.
//
// mode controls diagnostics and checking during SSA construction.
//
func NewProgram(fset *token.FileSet, mode BuilderMode) *Program {
	prog := &Program{
		Fset:                fset,
		PackagesByPath:      make(map[string]*Package),
		PackagesByName:      make(map[string]*Package),
		packages:            make(map[*types.Package]*Package),
		builtins:            make(map[*types.Builtin]*Builtin),
		boundMethodWrappers: make(map[*types.Func]*Function),
		ifaceMethodWrappers: make(map[*types.Func]*Function),
		mode:                mode,
	}

	// Create Values for built-in functions.
	for _, name := range types.Universe.Names() {
		if obj, ok := types.Universe.Lookup(name).(*types.Builtin); ok {
			// FIXME: end position is not right
			prog.builtins[obj] = &Builtin{obj, obj.Pos()}
		}
	}

	return prog
}

// memberFromObject populates package pkg with a member for the
// typechecker object obj.
//
// For objects from Go source code, syntax is the associated syntax
// tree (for funcs and vars only); it will be used during the build
// phase.
//
func memberFromObject(pkg *Package, obj types.Object, syntax ast.Node) {
	name := obj.Name()
	switch obj := obj.(type) {
	case *types.TypeName:
		pkg.values[obj] = nil // for needMethods
		pkg.Members[name] = &Type{
			object: obj,
			pkg:    pkg,
		}

	case *types.Const:
		pos  := obj.Pos()
		endP := obj.Pos()
		if syntax != nil {
			if try := syntax.Pos(); try != token.NoPos { pos = try }
			if try := syntax.End(); try != token.NoPos { endP = try }
		}
		c := &NamedConst{
			object: obj,
			Value:  NewConst(obj.Val(), obj.Type(), pos, endP),
			pkg:    pkg,
		}
		pkg.values[obj] = c.Value
		pkg.Members[name] = c

	case *types.Var:
		spec, _ := syntax.(*ast.ValueSpec)
		pos  := obj.Pos()
		endP := obj.Pos()
		if syntax != nil {
			// if try := syntax.Pos(); try != token.NoPos { pos = try }
			if try := syntax.End(); try != token.NoPos { endP = try }
		}
		g := &Global{
			Pkg:    pkg,
			name:   name,
			object: obj,
			typ:    types.NewPointer(obj.Type()), // address
			pos:    pos,
			endP:   endP,
			spec:   spec,
		}
		pkg.values[obj] = g
		pkg.Members[name] = g

	case *types.Func:
		var fs *funcSyntax
		var scope *Scope = nil
		if decl, ok := syntax.(*ast.FuncDecl); ok {
			fs = &funcSyntax{
				functype:  decl.Type,
				recvField: decl.Recv,
				body:      decl.Body,
			}
			scope = pkg.TypeScope2Scope[pkg.info.Scopes[decl.Type]]
			scope.node = &syntax
		}
		pos  := obj.Pos()
		endP := obj.Pos()
		if syntax != nil {
			// if try := syntax.Pos(); try != token.NoPos { pos = try }
			if try := syntax.End(); try != token.NoPos { endP = try }
		}
		fn := &Function{
			name:      name,
			object:    obj,
			Signature: obj.Type().(*types.Signature),
			syntax:    syntax,
			pos:       pos,
			endP:      endP,
			Pkg:       pkg,
			Prog:      pkg.Prog,
			Breakpoint: false,
			Scope     : scope,
			LocalsByName: make(map[NameScope]uint),
		}
		if syntax == nil {
			fn.Synthetic = "loaded from gc object file"
		}

		if fs != nil && fs.body != nil {
			fn.endP =  fs.body.End()
		}
		pkg.values[obj] = fn
		if fn.Signature.Recv() == nil {
			pkg.Members[name] = fn // package-level function
		}

	default: // (incl. *types.Package)
		panic("unexpected Object type: " + obj.String())
	}
}

// membersFromDecl populates package pkg with members for each
// typechecker object (var, func, const or type) associated with the
// specified decl.
//
func membersFromDecl(pkg *Package, decl ast.Decl) {
	switch decl := decl.(type) {
	case *ast.GenDecl: // import, const, type or var
		switch decl.Tok {
		case token.CONST:
			for _, spec := range decl.Specs {
				for _, id := range spec.(*ast.ValueSpec).Names {
					if !isBlankIdent(id) {
						memberFromObject(pkg, pkg.objectOf(id), nil)
					}
				}
			}

		case token.VAR:
			for _, spec := range decl.Specs {
				for _, id := range spec.(*ast.ValueSpec).Names {
					if !isBlankIdent(id) {
						memberFromObject(pkg, pkg.objectOf(id), spec)
					}
				}
			}

		case token.TYPE:
			for _, spec := range decl.Specs {
				id := spec.(*ast.TypeSpec).Name
				if !isBlankIdent(id) {
					memberFromObject(pkg, pkg.objectOf(id), nil)
				}
			}
		}

	case *ast.FuncDecl:
		id := decl.Name
		if decl.Recv == nil && id.Name == "init" {
			return // no object
		}
		if !isBlankIdent(id) {
			memberFromObject(pkg, pkg.objectOf(id), decl)
		}
	}
}

// CreatePackage constructs and returns an SSA Package from an
// error-free package described by info, and populates its Members
// mapping.
//
// Repeated calls with the same info return the same Package.
//
// The real work of building SSA form for each function is not done
// until a subsequent call to Package.Build().
//
func (prog *Program) CreatePackage(info *importer.PackageInfo) *Package {
	if info.Err != nil {
		panic(fmt.Sprintf("package %s has errors: %s", info, info.Err))
	}
	if p := prog.packages[info.Pkg]; p != nil {
		return p // already loaded
	}

	p := &Package{
		Prog:    prog,
		Members: make(map[string]Member),
		values:  make(map[types.Object]Value),
		Object:  info.Pkg,
		info:    info, // transient (CREATE and BUILD phases)
		locs:    make([] LocInst, 0),
		TypeScope2Scope: make(map[*types.Scope]*Scope),
	}

	// 0 scope number is pkg init function
	scope    := assignScopeId(info.Pkg.Scope(), 0)

	// FIXME: rocky I think this in there some other way now in the types interface
	// p.Ast2Scope[scope.Node()] = scope

	scopeId := ScopeId(1)
	AssignScopeIds(p, info.Pkg.Scope(), &scopeId)

	// Add init() function.
	p.Init = &Function{
		name:      "init",
		Signature: new(types.Signature),
		Synthetic: "package initializer",
		LocalsByName: make(map[NameScope]uint),
		Breakpoint: false,
		Scope:     scope,
		Pkg:       p,
		Prog:      prog,
	}
	p.Members[p.Init.name] = p.Init

	// CREATE phase.
	// Allocate all package members: vars, funcs, consts and types.
	if len(info.Files) > 0 {
		// Go source package.
		for _, file := range info.Files {
			for _, decl := range file.Decls {
				membersFromDecl(p, decl)
			}
		}
	} else {
		// GC-compiled binary package.
		// No code.
		// No position information.
		scope := p.Object.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			memberFromObject(p, obj, nil)
			if obj, ok := obj.(*types.TypeName); ok {
				named := obj.Type().(*types.Named)
				for i, n := 0, named.NumMethods(); i < n; i++ {
					memberFromObject(p, named.Method(i), nil)
				}
			}
		}
	}

	// Add initializer guard variable.
	initguard := &Global{
		Pkg:  p,
		name: "init$guard",
		typ:  types.NewPointer(tBool),
	}
	p.Members[initguard.Name()] = initguard

	if prog.mode&LogPackages != 0 {
		p.DumpTo(os.Stderr)
	}

	if info.Importable {
		prog.PackagesByPath[info.Pkg.Path()] = p
	}
	prog.PackagesByName[p.Object.Name()] = p
	prog.packages[p.Object] = p

	if prog.mode&SanityCheckFunctions != 0 {
		sanityCheckPackage(p)
	}

	return p
}

// CreatePackages creates SSA Packages for all error-free packages
// loaded by the specified Importer.
//
// If all packages were error-free, it is safe to call
// prog.BuildAll(), and nil is returned.  Otherwise an error is
// returned.
//
func (prog *Program) CreatePackages(imp *importer.Importer) error {
	var errpkgs []string
	for _, info := range imp.AllPackages() {
		if info.Err != nil {
			errpkgs = append(errpkgs, info.Pkg.Path())
		} else {
			prog.CreatePackage(info)
		}
	}
	if errpkgs != nil {
		return fmt.Errorf("couldn't create these SSA packages due to type errors: %s",
			strings.Join(errpkgs, ", "))
	}
	return nil
}

// AllPackages returns a new slice containing all packages in the
// program prog in unspecified order.
//
func (prog *Program) AllPackages() []*Package {
	pkgs := make([]*Package, 0, len(prog.packages))
	for _, pkg := range prog.packages {
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

// ImportedPackage returns the importable SSA Package whose import
// path is path, or nil if no such SSA package has been created.
//
// Not all packages are importable.  For example, no import
// declaration can resolve to the x_test package created by 'go test'
// or the ad-hoc main package created 'go build foo.go'.
//
func (prog *Program) ImportedPackage(path string) *Package {
	return prog.PackagesByPath[path]
}
