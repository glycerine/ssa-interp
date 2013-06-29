// Copyright 2013 Rocky Bernstein.
// Things dealing with locations

package gub

import (
	"fmt"

	"go/token"
	"ssa-interp"
	"ssa-interp/interp"
)

var Event2Icon map[ssa2.TraceEvent]string

func init() {
	Event2Icon = map[ssa2.TraceEvent]string{
		ssa2.OTHER      : "???",
		ssa2.ASSIGN_STMT: ":= ",
		ssa2.BLOCK_END  : "}  ",
		ssa2.BREAK_STMT : "<-X",
		ssa2.CALL_ENTER : "-> ",
		ssa2.CALL_RETURN: "<- ",
		ssa2.EXPR       : "(.)",
		ssa2.IF_INIT    : "if:",
		ssa2.IF_COND    : "if?",
		ssa2.FOR_INIT   : "lo:",
		ssa2.FOR_COND   : "lo?",
		ssa2.FOR_ITER   : "lo+",
		ssa2.MAIN       : "m()",
		ssa2.RANGE_STMT : "...",
		ssa2.SELECT_TYPE: "sel",
		ssa2.STMT_IN_LIST: "---",
	}
}

func StackLocation(fr *interp.Frame) string {
	fn := fr.Fn()
	s := fmt.Sprintf("%s(", fn.Name())
	params :=""
	if len(fn.Params) > 0 {
		params = fn.Params[0].Name()
		for i:=1; i<len(fn.Params); i++ {
			params += ", " + fn.Params[i].Name()
		}
	}
	s += params + ")"
	return s
}

func fmtLocation(start token.Position, end token.Position) string {
	return fmt.Sprintf("%s", interp.PositionRange(start, end))
}

func printLocInfo(fr *interp.Frame, event ssa2.TraceEvent) {
	fset := fr.Fn().Prog.Fset
	startP := fset.Position(fr.StartP())
	endP   := fset.Position(fr.EndP())
	s := Event2Icon[event] + " "
	if len(fr.Fn().Name()) > 0 {
		s += fr.Fn().Name() + "() "
	}
	fmt.Println(s)
	if (event == ssa2.CALL_RETURN) {
		fmt.Printf("return: %s\n", interp.ToString(fr.Result()))
	} else if (event == ssa2.CALL_ENTER) {
		for i, p := range fr.Fn().Params {
			fmt.Println(fr.Fn().Params[i], fr.Env()[p])
		}
	}

	fmt.Println(fmtLocation(startP, endP))
}