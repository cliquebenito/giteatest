// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package gitgraph

import (
	"bytes"
	"fmt"
)

// Parser represents a git graph parser. It is stateful containing the previous
// glyphs, detected flows and color assignments.
type Parser struct {
	glyphs           []byte
	oldGlyphs        []byte
	flows            []int64
	oldFlows         []int64
	maxFlow          int64
	colors           []int
	oldColors        []int
	AvailableColors  []int
	nextAvailable    int
	FirstInUse       int
	firstAvailable   int
	MaxAllowedColors int
}

// Reset resets the internal parser state.
func (parser *Parser) Reset() {
	parser.glyphs = parser.glyphs[0:0]
	parser.oldGlyphs = parser.oldGlyphs[0:0]
	parser.flows = parser.flows[0:0]
	parser.oldFlows = parser.oldFlows[0:0]
	parser.maxFlow = 0
	parser.colors = parser.colors[0:0]
	parser.oldColors = parser.oldColors[0:0]
	parser.AvailableColors = parser.AvailableColors[0:0]
	parser.AvailableColors = append(parser.AvailableColors, 1, 2)
	parser.nextAvailable = 0
	parser.FirstInUse = -1
	parser.firstAvailable = 0
	parser.MaxAllowedColors = 0
}

// AddLineToGraph adds the line as a row to the graph
func (parser *Parser) AddLineToGraph(graph *Graph, row int, line []byte) error {
	idx := bytes.Index(line, []byte("DATA:"))
	if idx < 0 {
		parser.ParseGlyphs(line)
	} else {
		parser.ParseGlyphs(line[:idx])
	}

	var err error
	commitDone := false

	for column, glyph := range parser.glyphs {
		if glyph == ' ' {
			continue
		}

		flowID := parser.flows[column]

		graph.AddGlyph(row, column, flowID, parser.colors[column], glyph)

		if glyph == '*' {
			if commitDone {
				if err != nil {
					err = fmt.Errorf("double commit on line %d: %s. %w", row, string(line), err)
				} else {
					err = fmt.Errorf("double commit on line %d: %s", row, string(line))
				}
			}
			commitDone = true
			if idx < 0 {
				if err != nil {
					err = fmt.Errorf("missing data section on line %d with commit: %s. %w", row, string(line), err)
				} else {
					err = fmt.Errorf("missing data section on line %d with commit: %s", row, string(line))
				}
				continue
			}
			err2 := graph.AddCommit(row, column, flowID, line[idx+5:])
			if err != nil && err2 != nil {
				err = fmt.Errorf("%v %w", err2, err)
				continue
			} else if err2 != nil {
				err = err2
				continue
			}
		}
	}
	if !commitDone {
		graph.Commits = append(graph.Commits, RelationCommit)
	}
	return err
}

func (parser *Parser) releaseUnusedColors() {
	if parser.FirstInUse > -1 {
		// Here we step through the old colors, searching for them in the
		// "in-use" section of availableColors (that is, the colors between
		// firstInUse and firstAvailable)
		// Ensure that the benchmarks are not worsened with proposed changes
		stepstaken := 0
		position := parser.FirstInUse
		for _, color := range parser.oldColors {
			if color == 0 {
				continue
			}
			found := false
			i := position
			for j := stepstaken; i != parser.firstAvailable && j < len(parser.AvailableColors); j++ {
				colorToCheck := parser.AvailableColors[i]
				if colorToCheck == color {
					found = true
					break
				}
				i = (i + 1) % len(parser.AvailableColors)
			}
			if !found {
				// Duplicate color
				continue
			}
			// Swap them around
			parser.AvailableColors[position], parser.AvailableColors[i] = parser.AvailableColors[i], parser.AvailableColors[position]
			stepstaken++
			position = (parser.FirstInUse + stepstaken) % len(parser.AvailableColors)
			if position == parser.firstAvailable || stepstaken == len(parser.AvailableColors) {
				break
			}
		}
		if stepstaken == len(parser.AvailableColors) {
			parser.firstAvailable = -1
		} else {
			parser.firstAvailable = position
			if parser.nextAvailable == -1 {
				parser.nextAvailable = parser.firstAvailable
			}
		}
	}
}

// ParseGlyphs parses the provided glyphs and sets the internal state
func (parser *Parser) ParseGlyphs(glyphs []byte) {
	// Clean state for parsing this row
	parser.glyphs, parser.oldGlyphs = parser.oldGlyphs, parser.glyphs
	parser.glyphs = parser.glyphs[0:0]
	parser.flows, parser.oldFlows = parser.oldFlows, parser.flows
	parser.flows = parser.flows[0:0]
	parser.colors, parser.oldColors = parser.oldColors, parser.colors

	// Ensure we have enough flows and colors
	parser.colors = parser.colors[0:0]
	for range glyphs {
		parser.flows = append(parser.flows, 0)
		parser.colors = append(parser.colors, 0)
	}

	// Copy the provided glyphs in to state.glyphs for safekeeping
	parser.glyphs = append(parser.glyphs, glyphs...)

	// release unused colors
	parser.releaseUnusedColors()

	for i := len(glyphs) - 1; i >= 0; i-- {
		glyph := glyphs[i]
		switch glyph {
		case '|':
			fallthrough
		case '*':
			parser.setUpFlow(i)
		case '/':
			parser.setOutFlow(i)
		case '\\':
			parser.setInFlow(i)
		case '_':
			parser.setRightFlow(i)
		case '.':
			fallthrough
		case '-':
			parser.setLeftFlow(i)
		case ' ':
			// no-op
		default:
			parser.newFlow(i)
		}
	}
}

func (parser *Parser) takePreviousFlow(i, j int) {
	if j < len(parser.oldFlows) && parser.oldFlows[j] > 0 {
		parser.flows[i] = parser.oldFlows[j]
		parser.oldFlows[j] = 0
		parser.colors[i] = parser.oldColors[j]
		parser.oldColors[j] = 0
	} else {
		parser.newFlow(i)
	}
}

func (parser *Parser) takeCurrentFlow(i, j int) {
	if j < len(parser.flows) && parser.flows[j] > 0 {
		parser.flows[i] = parser.flows[j]
		parser.colors[i] = parser.colors[j]
	} else {
		parser.newFlow(i)
	}
}

func (parser *Parser) newFlow(i int) {
	parser.maxFlow++
	parser.flows[i] = parser.maxFlow

	// Now give this flow a color
	if parser.nextAvailable == -1 {
		next := len(parser.AvailableColors)
		if parser.MaxAllowedColors < 1 || next < parser.MaxAllowedColors {
			parser.nextAvailable = next
			parser.firstAvailable = next
			parser.AvailableColors = append(parser.AvailableColors, next+1)
		}
	}
	parser.colors[i] = parser.AvailableColors[parser.nextAvailable]
	if parser.FirstInUse == -1 {
		parser.FirstInUse = parser.nextAvailable
	}
	parser.AvailableColors[parser.firstAvailable], parser.AvailableColors[parser.nextAvailable] = parser.AvailableColors[parser.nextAvailable], parser.AvailableColors[parser.firstAvailable]

	parser.nextAvailable = (parser.nextAvailable + 1) % len(parser.AvailableColors)
	parser.firstAvailable = (parser.firstAvailable + 1) % len(parser.AvailableColors)

	if parser.nextAvailable == parser.FirstInUse {
		parser.nextAvailable = parser.firstAvailable
	}
	if parser.nextAvailable == parser.FirstInUse {
		parser.nextAvailable = -1
		parser.firstAvailable = -1
	}
}

// setUpFlow handles '|' or '*'
func (parser *Parser) setUpFlow(i int) {
	// In preference order:
	//
	// Previous Row: '\? ' ' |' '  /'
	// Current Row:  ' | ' ' |' ' | '
	if i > 0 && i-1 < len(parser.oldGlyphs) && parser.oldGlyphs[i-1] == '\\' {
		parser.takePreviousFlow(i, i-1)
	} else if i < len(parser.oldGlyphs) && (parser.oldGlyphs[i] == '|' || parser.oldGlyphs[i] == '*') {
		parser.takePreviousFlow(i, i)
	} else if i+1 < len(parser.oldGlyphs) && parser.oldGlyphs[i+1] == '/' {
		parser.takePreviousFlow(i, i+1)
	} else {
		parser.newFlow(i)
	}
}

// setOutFlow handles '/'
func (parser *Parser) setOutFlow(i int) {
	// In preference order:
	//
	// Previous Row: ' |/' ' |_' ' |' ' /' ' _' '\'
	// Current Row:  '/| ' '/| ' '/ ' '/ ' '/ ' '/'
	if i+2 < len(parser.oldGlyphs) &&
		(parser.oldGlyphs[i+1] == '|' || parser.oldGlyphs[i+1] == '*') &&
		(parser.oldGlyphs[i+2] == '/' || parser.oldGlyphs[i+2] == '_') &&
		i+1 < len(parser.glyphs) &&
		(parser.glyphs[i+1] == '|' || parser.glyphs[i+1] == '*') {
		parser.takePreviousFlow(i, i+2)
	} else if i+1 < len(parser.oldGlyphs) &&
		(parser.oldGlyphs[i+1] == '|' || parser.oldGlyphs[i+1] == '*' ||
			parser.oldGlyphs[i+1] == '/' || parser.oldGlyphs[i+1] == '_') {
		parser.takePreviousFlow(i, i+1)
		if parser.oldGlyphs[i+1] == '/' {
			parser.glyphs[i] = '|'
		}
	} else if i < len(parser.oldGlyphs) && parser.oldGlyphs[i] == '\\' {
		parser.takePreviousFlow(i, i)
	} else {
		parser.newFlow(i)
	}
}

// setInFlow handles '\'
func (parser *Parser) setInFlow(i int) {
	// In preference order:
	//
	// Previous Row: '| ' '-. ' '| ' '\ ' '/' '---'
	// Current Row:  '|\' '  \' ' \' ' \' '\' ' \ '
	if i > 0 && i-1 < len(parser.oldGlyphs) &&
		(parser.oldGlyphs[i-1] == '|' || parser.oldGlyphs[i-1] == '*') &&
		(parser.glyphs[i-1] == '|' || parser.glyphs[i-1] == '*') {
		parser.newFlow(i)
	} else if i > 0 && i-1 < len(parser.oldGlyphs) &&
		(parser.oldGlyphs[i-1] == '|' || parser.oldGlyphs[i-1] == '*' ||
			parser.oldGlyphs[i-1] == '.' || parser.oldGlyphs[i-1] == '\\') {
		parser.takePreviousFlow(i, i-1)
		if parser.oldGlyphs[i-1] == '\\' {
			parser.glyphs[i] = '|'
		}
	} else if i < len(parser.oldGlyphs) && parser.oldGlyphs[i] == '/' {
		parser.takePreviousFlow(i, i)
	} else {
		parser.newFlow(i)
	}
}

// setRightFlow handles '_'
func (parser *Parser) setRightFlow(i int) {
	// In preference order:
	//
	// Current Row:  '__' '_/' '_|_' '_|/'
	if i+1 < len(parser.glyphs) &&
		(parser.glyphs[i+1] == '_' || parser.glyphs[i+1] == '/') {
		parser.takeCurrentFlow(i, i+1)
	} else if i+2 < len(parser.glyphs) &&
		(parser.glyphs[i+1] == '|' || parser.glyphs[i+1] == '*') &&
		(parser.glyphs[i+2] == '_' || parser.glyphs[i+2] == '/') {
		parser.takeCurrentFlow(i, i+2)
	} else {
		parser.newFlow(i)
	}
}

// setLeftFlow handles '----.'
func (parser *Parser) setLeftFlow(i int) {
	if parser.glyphs[i] == '.' {
		parser.newFlow(i)
	} else if i+1 < len(parser.glyphs) &&
		(parser.glyphs[i+1] == '-' || parser.glyphs[i+1] == '.') {
		parser.takeCurrentFlow(i, i+1)
	} else {
		parser.newFlow(i)
	}
}
