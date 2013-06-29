// Copyright 2013 Rocky Bernstein.
// Debugger commands
package gub

import (
	"fmt"
	"os"
	"strconv"

	"go/token"
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"


	"ssa-interp"
	"ssa-interp/interp"
	"gnureadline"
)

func argCountOK(min int, max int, args [] string) bool {
	l := len(args)-1 // strip command name from count
	if (l < min) {
		errmsg("Too few args; need at least %d, got %d", min, l)
		return false
	} else if (l > max) {
		errmsg("Too many args; need at most %d, got %d", max, l)
		return false
	}
	return true
}

func BacktraceCommand(args []string) {
	if !argCountOK(0, 1, args) { return }
	// FIXME: should get limit from args
	fr := topFrame
	for i:=0; fr !=nil; fr = fr.Caller(0) {
		pointer := "   "
		if fr == curFrame {
			pointer = "=> "
		}
		msg("%s#%d %s", pointer, i, StackLocation(fr))
		i++
	}
}

func FinishCommand(args []string) {
	interp.SetStepOut(topFrame)
	msg("Continuing until return...")
}

func FrameCommand(args []string) {
	if !argCountOK(1, 1, args) { return }
	frameIndex, ok := strconv.Atoi(args[1])
	if ok != nil {
		errmsg("Expecting integer frame number; got %s",
			args[1])
		return
	}
	adjustFrame(frameIndex, true)

}

func UpCommand(args []string) {
	if !argCountOK(0, 1, args) { return }
	var frameIndex int
	var ok error
	if len(args) == 1 {
		frameIndex = 1
	} else {
		frameIndex, ok = strconv.Atoi(args[1])
		if ok != nil {
			fmt.Printf("Expecting integer frame number; got %s",
				args[1])
			return
		}
	}
	adjustFrame(frameIndex, false)

}

func DownCommand(args []string) {
	if !argCountOK(0, 1, args) { return }
	var frameIndex int
	var ok error
	if len(args) == 1 {
		frameIndex = 1
	} else {
		frameIndex, ok = strconv.Atoi(args[1])
		if ok != nil {
			errmsg("Expecting integer frame number; got %s", args[1])
			return
		}
	}
	adjustFrame(-frameIndex, false)

}

func NextCommand(args []string) {
	interp.SetStepOver(topFrame)
	fmt.Println("Step over...")
}

func HelpCommand(args []string) {
	fmt.Println(`List of commands:
Execution running --
  s: step in
  n: next or step over
  fin: finish or step out
  c: continue

Variables --
  local [*name*]:  show local variable info
  global [*name*]: show global variable info
  param [*name*]: show function parameter info

Tracing --
  +: add instruction tracing
  -: remove instruction tracing

Stack:
  bt: print a backtrace
  frame *num*: switch stack frame
  up *num*: switch to a newer frame
  down *num*: switch to a older frame

Other:
  ?: this help
  q: quit
`)
}

func GlobalsCommand(args []string) {
	argc := len(args) - 1
	if argc == 0 {
		for k, v := range curFrame.I().Globals() {
			if v == nil {
				fmt.Printf("%s: nil\n")
			} else {
				// FIXME: figure out why reflect.lookupCache causes
				// an panic on a nil pointer or invalid address
				if fmt.Sprintf("%s", k) == "reflect.lookupCache" {
					continue
				}
				msg("%s: %s", k, interp.ToString(*v))
			}
		}
	} else {
		// This doesn't work and I don't know how to fix it.
		for i:=1; i<=argc; i++ {
			vv := ssa2.NewLiteral(exact.MakeString(args[i]),
				types.Typ[types.String], token.NoPos, token.NoPos)
			// fmt.Println(vv, "vs", interp.ToString(vv))
			v, ok := curFrame.I().Globals()[vv]
			if ok {
				msg("%s: %s", vv, interp.ToString(*v))
			}
		}

		// This is ugly, but I don't know how to turn a string into
		// a ssa2.Value.
		globals := make(map[string]*interp.Value)
		for k, v := range curFrame.I().Globals() {
			globals[fmt.Sprintf("%s", k)] = v
		}

		for i:=1; i<=argc; i++ {
			vv := args[i]
			v, ok := globals[vv]
			if ok {
				msg("%s: %s", vv, interp.ToString(*v))
			}
		}
	}
}

func ParametersCommand(args []string) {
	argc := len(args) - 1
	if !argCountOK(0, 1, args) { return }
	if argc == 0 {
		for i, p := range curFrame.Fn().Params {
			fmt.Println(curFrame.Fn().Params[i], curFrame.Env()[p])
		}
	} else {
		varname := args[1]
		for i, p := range curFrame.Fn().Params {
			if varname == curFrame.Fn().Params[i].Name() {
				fmt.Println(curFrame.Fn().Params[i], curFrame.Env()[p])
				break
			}
		}
	}
}

func LocalsCommand(args []string) {
	argc := len(args) - 1
	if !argCountOK(0, 2, args) { return }
	if argc == 0 {
		i := 0
		for _, v := range curFrame.Locals() {
			name := curFrame.Fn().Locals[i].Name()
			fmt.Printf("%s: %s\n", name, interp.ToString(v))
			i++
		}
	} else {
		varname := args[1]
		i := 0
		for _, v := range curFrame.Locals() {
			if args[1] == curFrame.Fn().Locals[i].Name() {
				msg("%s %s: %s", varname, curFrame.Fn().Locals[i], interp.ToString(v))
				break
			}
			i++
		}

	}
}

// quit [exit-code]
//
// Terminates program. If an exit code is given, that is the exit code
// for the program. Zero (normal termination) is used if no
// termintation code.
func QuitCommand(args []string) {
	if !argCountOK(0, 1, args) { return }
	rc := 0
	if len(args) == 2 {
		new_rc, ok := strconv.Atoi(args[1])
		if ok == nil { rc = new_rc } else {
			errmsg("Expecting integer return code; got %s. Ignoring",
				args[1])
		}
	}
	msg("That's all folks...")
	gnureadline.Rl_reset_terminal(term)
	os.Exit(rc)

}

func VariableCommand(args []string) {
	if !argCountOK(1, 1, args) { return }
	fn := curFrame.Fn()
	varname := args[1]
	for _, p := range fn.Locals {
		if varname == p.Name() { break }
	}

}
