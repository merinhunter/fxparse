package fxparse

import "fxsym"

type Builtin struct {
	name string
	kind int
	args []string
}

var (
	builtins = map[string]Builtin{
		"circle": {"circle", fxsym.SFunc, []string{"x", "y", "r", "color"}},
		"rect":   {"rect", fxsym.SFunc, []string{"p", "angle", "color"}},
	}
)
