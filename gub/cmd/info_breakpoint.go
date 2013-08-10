// Copyright 2013 Rocky Bernstein.
// Debugger info breakpoint command

package gubcmd
import 	"github.com/rocky/ssa-interp/gub"

func init() {
	parent := "info"
	gub.AddSubCommand(parent, &gub.SubcmdInfo{
		Fn: InfoScopeSubcmd,
		Help: `info breakpoint

Show status of user-settable breakpoints.
`,
		Min_args: 1,
		Max_args: 2,
		Short_help: "Status of user-settable breakpoints",
		Name: "breakpoint",
	})
}

func InfoBreakpointSubcmd() {
	if gub.IsBreakpointEmpty() {
		gub.Msg("No breakpoints set")
		return
	}
	if len(gub.Breakpoints) - gub.BrkptsDeleted == 0 {
		gub.Msg("No breakpoints.")
	}
	gub.Section("Num Type          Disp Enb Where")
	for _, bp := range gub.Breakpoints {
		if bp.Deleted { continue }
		gub.Bpprint(*bp)
	}
}