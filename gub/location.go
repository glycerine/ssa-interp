// Copyright 2013 Rocky Bernstein.
// Things dealing with locations

package gub

import (
	"fmt"
	"github.com/rocky/ssa-interp"
	"github.com/rocky/ssa-interp/interp"
)

var Event2Icon map[ssa2.TraceEvent]string

func init() {
	Event2Icon = map[ssa2.TraceEvent]string{
		ssa2.OTHER           : "???",
		ssa2.ASSIGN_STMT     : ":= ",
		ssa2.BLOCK_END       : "}  ",
		ssa2.BREAK_STMT      : "<-X",
		ssa2.BREAKPOINT      : "xxx",
		ssa2.CALL_ENTER      : "-> ",
		ssa2.CALL_RETURN     : "<- ",
		ssa2.DEFER_ENTER     : "d->",
		ssa2.EXPR            : "(.)",
		ssa2.IF_INIT         : "if:",
		ssa2.IF_COND         : "if?",
		ssa2.STEP_INSTRUCTION: "...",
		ssa2.FOR_INIT        : "lo:",
		ssa2.FOR_COND        : "lo?",
		ssa2.FOR_ITER        : "lo+",
		ssa2.MAIN            : "m()",
		ssa2.PANIC           : "oX ",  // My attempt at skull and cross bones
		ssa2.RANGE_STMT      : "...",
		ssa2.SELECT_TYPE     : "sel",
		ssa2.SWITCH_COND     : "sw?",
		ssa2.STMT_IN_LIST    : "---",
	}
}

func printLocInfo(fr *interp.Frame, inst *ssa2.Instruction,
	event ssa2.TraceEvent) {
	s    := Event2Icon[event] + " "
	fn   := fr.Fn()
	sig  := fn.Signature
	name := fn.Name()

	if fn.Signature.Recv() != nil {
		if len(fn.Params) == 0 {
			panic("Receiver method "+name+" should have at least 1 param. Has 0.")
		}
		s += fmt.Sprintf("(%s).%s()", fn.Params[0].Type(), name)
	} else {
		s += name
		if len(name) > 0 { s += "()" }
	}

	if *terse && (event != ssa2.STEP_INSTRUCTION) {
		Msg(s)
	} else {
		Msg("%s block %d insn %d", s, fr.Block().Index, fr.PC())
	}
	switch event {
	case ssa2.CALL_RETURN:
		if sig.Results() == nil {
			Msg("return void")
		} else {
			Msg("return type: %s", sig.Results())
			Msg("return value: %s", Deref2Str(fr.Result()))
		}
	case ssa2.CALL_ENTER:
		for i, p := range fn.Params {
			if val := fr.Env()[p]; val != nil {
				Msg("%s %s", fn.Params[i], Deref2Str(val))
			} else {
				Msg("%s nil", fn.Params[i])
			}
		}
	case ssa2.PANIC:
		// fmt.Printf("panic arg: %s\n", fr.Get(instr.X))
	}

	Msg(fr.PositionRange())
}
