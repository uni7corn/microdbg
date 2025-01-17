package debugger

import "golang.org/x/exp/constraints"

func Align[I constraints.Integer](a, b I) I {
	return (a + b - 1) &^ (b - 1)
}
