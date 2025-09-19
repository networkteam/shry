package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func PrettyPrint(diffs []diffmatchpatch.Diff) {
	// TODO use bubbletea view
	fmt.Println(prettify(diffs))
}

const contextLines = 2

type operationRune rune

const (
	operationRuneInsert operationRune = '+'
	operationRuneDelete operationRune = '-'
	operationRuneEqual  operationRune = ' '
)

var (
	insertLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	deleteLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	omitLineStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#808080"))
)

func formatLine(opRune operationRune, text string) string {
	return string(opRune) + "    " + text
}

func renderLine(op diffmatchpatch.Operation, text string) string {
	switch op {
	default:
		panic("Unknown operation type")
	case diffmatchpatch.DiffInsert:
		return insertLineStyle.Render(formatLine(operationRuneInsert, text))
	case diffmatchpatch.DiffDelete:
		return deleteLineStyle.Render(formatLine(operationRuneDelete, text))
	case diffmatchpatch.DiffEqual:
		return formatLine(operationRuneEqual, text)
	}
}

func prettify(hunks []diffmatchpatch.Diff) string {
	var result []string
	for hunkIdx, hunk := range hunks {
		lines := strings.Split(hunk.Text, "\n")

		// remove last element if it is empty due to trailing linebreak
		if lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		nLines := len(lines)

		if hunk.Type == diffmatchpatch.DiffEqual && nLines > contextLines*2+1 {
			lines = compressHunk(lines, hunk.Type, hunkIdx == 0, hunkIdx == len(hunks)-1)
		} else {
			for i, line := range lines {
				lines[i] = renderLine(hunk.Type, line)
			}
		}

		result = append(result, lines...)
	}

	return strings.Join(result, "\n")
}

func compressHunk(lines []string, op diffmatchpatch.Operation, isFirst bool, isLast bool) []string {
	var compressedLines []string
	nLines := len(lines)
	unchangedLines := nLines

	if !isFirst {
		for i := range contextLines {
			compressedLines = append(compressedLines, renderLine(op, lines[i]))
			unchangedLines--
		}
	}

	// add a dummy line and save its index
	compressedLines = append(compressedLines, "")
	infoLineIdx := len(compressedLines) - 1

	if !isLast {
		for i := nLines - contextLines; i < nLines; i++ {
			compressedLines = append(compressedLines, renderLine(op, lines[i]))
			unchangedLines--
		}
	}

	// add info line
	compressedLines[infoLineIdx] = omitLineStyle.Render(fmt.Sprintf("@@ <...> (%d more lines) @@", unchangedLines))

	return compressedLines
}
