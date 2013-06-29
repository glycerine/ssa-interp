// Copyright 2013 Rocky Bernstein.
// Debugger callback hook. Contains the main command loop.
package gub

import (
	"fmt"
	"strings"

	"gnureadline"
	"ssa-interp"
	"ssa-interp/interp"
)

var cmdCount int = 0
var traceEvent ssa2.TraceEvent

// Call-back hook from interpreter. Contains top-level statement breakout
func GubTraceHook(fr *interp.Frame, instr *ssa2.Instruction, event ssa2.TraceEvent) {
	traceEvent = event
	frameInit(fr)
	printLocInfo(topFrame, event)

	line := ""
	var err error
	for inCmdLoop := true; err == nil && inCmdLoop; cmdCount++ {
		line, err = gnureadline.Readline(fmt.Sprintf("gub[%d]: ", cmdCount),
			true)
		args  := strings.Split(line, " ")
		if len(args) == 0 {
			fmt.Println("Empty line skipped")
			continue
		}

		cmd := args[0]

		switch cmd {
		case "s":
			fmt.Println("Stepping...")
			interp.SetStepIn(curFrame)
			inCmdLoop = false
			break
		case "h", "?", "help":
			HelpCommand(args)
		case "c":
			interp.SetStepOff(topFrame)
			fmt.Println("Continuing...")
			inCmdLoop = false
			break
		case "finish", "fin":
 			FinishCommand(args)
			inCmdLoop = false
			break
		case "next", "n":
 			NextCommand(args)
			inCmdLoop = false
			break
		case "env":
			for i, p := range topFrame.Env() {
				fmt.Println(i, p)
			}
		case "+":
			fmt.Println("Setting Instruction Trace")
			interp.SetInstTracing()
		case "-":
			fmt.Println("Clearing Instruction Trace")
			interp.ClearInstTracing()
		case "frame":
			FrameCommand(args)
		case "gl", "global", "globals":
			GlobalsCommand(args)
		case "lo", "local", "locals":
			LocalsCommand(args)
		case "param", "parameters":
			ParametersCommand(args)
		case "q", "quit", "exit":
			QuitCommand(args)
		case "bt", "T", "backtrace", "where":
			BacktraceCommand(args)
		case "v":
			VariableCommand(args)
		default:
			fmt.Printf("Unknown command %s\n", cmd)
		}
	}
}
