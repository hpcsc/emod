package cli

import (
	"slices"
	"strings"

	urfave "github.com/urfave/cli/v2"
)

// Run executes the emod CLI over args, which are the process arguments with the
// program name still at the front.
func Run(args []string) error {
	return RunApp(NewApp(), args)
}

// RunApp is Run over a caller-supplied app, so a test can substitute an exit
// handler and still go through the argument reordering the process does.
func RunApp(app *urfave.App, args []string) error {
	return app.Run(hoistFlags(app, args))
}

// hoistFlags moves a command's flags ahead of its file argument. Go's flag
// package stops parsing at the first argument that is not a flag and urfave/cli
// offers no way to turn that off, so `emod export model.emod --format cue`
// reaches the action with format still at its default and writes JSON — the
// wrong format, quietly and with a zero exit code.
func hoistFlags(app *urfave.App, args []string) []string {
	if len(args) < 2 {
		return args
	}

	names := []string{args[0]}
	rest := args[1:]
	flags, commands := app.Flags, app.Commands

	for len(rest) > 0 {
		command := commandNamed(commands, rest[0])
		if command == nil {
			break
		}

		names = append(names, rest[0])
		rest = rest[1:]
		flags, commands = command.Flags, command.Subcommands
	}

	hoisted, positional := partitionArgs(flags, rest)

	reordered := make([]string, 0, len(args))
	reordered = append(reordered, names...)
	reordered = append(reordered, hoisted...)
	return append(reordered, positional...)
}

func commandNamed(commands []*urfave.Command, name string) *urfave.Command {
	for _, command := range commands {
		if command.HasName(name) {
			return command
		}
	}

	return nil
}

// partitionArgs splits args into the flags a command declares, each still
// followed by its value, and everything else in the order it was written.
func partitionArgs(flags []urfave.Flag, args []string) (hoisted, positional []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Everything after a bare -- is an argument by definition, including
		// words that look like flags.
		if arg == "--" {
			return hoisted, append(positional, args[i:]...)
		}

		if len(arg) < 2 || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}

		hoisted = append(hoisted, arg)

		name := strings.TrimLeft(arg, "-")
		if strings.Contains(name, "=") {
			continue
		}

		// A flag the command does not declare takes nothing, so the word after
		// it stays an argument and urfave reports the flag itself as undefined.
		if i+1 < len(args) && takesValue(flags, name) {
			i++
			hoisted = append(hoisted, args[i])
		}
	}

	return hoisted, positional
}

func takesValue(flags []urfave.Flag, name string) bool {
	for _, flag := range flags {
		if !slices.Contains(flag.Names(), name) {
			continue
		}

		documented, ok := flag.(urfave.DocGenerationFlag)
		return !ok || documented.TakesValue()
	}

	return false
}
