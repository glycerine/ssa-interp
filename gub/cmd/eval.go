// Copyright 2013 Rocky Bernstein.
package gubcmd

import (
	"github.com/rocky/ssa-interp/gub"
	"github.com/rocky/ssa-interp/interp"
)

func init() {
	name := "eval"
	gub.Cmds[name] = &gub.CmdInfo{
		Fn: EvalCommand,
		Help: `eval *expr*

Evaluate go expression *expr*.
`,

		Min_args: 1,
		Max_args: -1,
	}
	gub.AddToCategory("data", name)
}

func EvalCommand(args []string) {

	// Don't use args, but gub.CmdArgstr which preserves blanks inside quotes
	if expr, err := gub.EvalExpr(gub.CmdArgstr); err == nil {
		if expr == nil {
			gub.Msg("nil")
		} else {
			expr := *expr
			if len(expr) == 1 {
				gub.Msg("%s", interp.ToInspect(expr[0].Interface()))
			} else {
				if len(expr) == 0 {
					gub.Errmsg("Something is weird. Result has length 0")
				} else {
					gub.MsgNoCr("(")
					size := len(expr)
					for i, v := range expr {
						gub.MsgNoCr("%v", v.Interface())
						if i < size-1 { gub.MsgNoCr(", ") }
					}
					gub.Msg(")")
				}
			}
		}
	}
}
