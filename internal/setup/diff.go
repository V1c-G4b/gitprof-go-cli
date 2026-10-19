package setup

import (
	"fmt"
	"io"
	"strings"
)

type diffOp struct {
	kind byte
	text string
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

func lineDiff(a, b []string) []diffOp {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var ops []diffOp
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{' ', a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, diffOp{'-', a[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{'-', a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{'+', b[j]})
	}
	return ops
}

func writeDiff(w io.Writer, oldText, newText string, context int) {
	ops := lineDiff(splitLines(oldText), splitLines(newText))
	show := make([]bool, len(ops))
	for k, op := range ops {
		if op.kind == ' ' {
			continue
		}
		for c := max(0, k-context); c <= min(len(ops)-1, k+context); c++ {
			show[c] = true
		}
	}
	gap := false
	for k, op := range ops {
		if !show[k] {
			gap = true
			continue
		}
		if gap {
			fmt.Fprintln(w, "  ...")
			gap = false
		}
		fmt.Fprintf(w, "%c %s\n", op.kind, op.text)
	}
}
