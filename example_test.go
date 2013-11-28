// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa2_test

import (
	"fmt"
	"go/build"
	"go/parser"
	"os"

	"code.google.com/p/go.tools/importer"
	"github.com/rocky/ssa-interp"
)

// This program demonstrates how to run the SSA builder on a "Hello,
// World!" program and shows the printed representation of packages,
// functions and instructions.
//
// Within the function listing, the name of each BasicBlock such as
// ".0.entry" is printed left-aligned, followed by the block's
// Instructions.
//
// For each instruction that defines an SSA virtual register
// (i.e. implements Value), the type of that value is shown in the
// right column.
//
// Build and run the tortoise.go program in this package if you want a
// standalone tool with similar functionality.
//
func Example() {
	const hello = `
package main

import "fmt"

const message = "Hello, World!"

func main() {
	fmt.Println(message)
}
`
	// Construct an importer.  Imports will be loaded as if by 'go build'.
	imp := importer.New(&importer.Config{Build: &build.Default})

	// Parse the input file.
	file, err := parser.ParseFile(imp.Fset, "hello.go", hello, 0)
	if err != nil {
		fmt.Print(err) // parse error
		return
	}

	// Create single-file main package and import its dependencies.
	mainInfo := imp.CreatePackage("main", file)

	// Create SSA-form program representation.
	var mode ssa2.BuilderMode
	prog := ssa2.NewProgram(imp.Fset, mode)
	if err := prog.CreatePackages(imp); err != nil {
		fmt.Print(err) // type error in some package
		return
	}
	mainPkg := prog.Package(mainInfo.Pkg)

	// Print out the package.
	mainPkg.DumpTo(os.Stdout)

	// Build SSA code for bodies of functions in mainPkg.
	mainPkg.Build()

	// Print out the package-level functions.
	mainPkg.Func("init").DumpTo(os.Stdout)
	mainPkg.Func("main").DumpTo(os.Stdout)

	// Output:
	//
	// package main:
	//   func  init       func()
	//   var   init$guard bool
	//   func  main       func()
	//   const message    message = "Hello, World!":untyped string
	//
	// # Name: main.init
	// # Package: main
	// # Synthetic: package initializer
	// func init():
	// # scope: 0
	// .0.entry:                                                               P:0 S:2
	// 	t0 = *init$guard                                                   bool
	// 	if t0 goto 2.init.done else 1.init.start
	// # scope: 0
	// .1.init.start:                                                          P:1 S:1
	// 	*init$guard = true:bool
	// 	t1 = fmt.init()                                                      ()
	// 	jump 2.init.done
	// # scope: 0
	// .2.init.done:                                                           P:2 S:0
	// 	return
	//
	// # Name: main.main
	// # Package: main
	// # Located at hello.go:8:6-10:2
	// func main():
	// .0.entry:                                                               P:0 S:0
	//	trace <STATEMENT in list> at hello.go:9:2-22
	//	t0 = new [1]interface{} (varargs)                       *[1]interface{}
	//	t1 = &t0[0:untyped integer]                                *interface{}
	//	t2 = make interface{} <- string ("Hello, World!":string)    interface{}
	//	*t1 = t2
	//	t3 = slice t0[:]                                          []interface{}
	//	t4 = fmt.Println(t3)                                 (n int, err error)
	//	trace <Block End> at hello.go:10:2
	// 	return
}
