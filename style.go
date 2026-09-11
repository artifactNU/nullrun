package main

import (
	"codeberg.org/anaseto/gruid"
	"github.com/gdamore/tcell/v2"
)

// Foreground colors used by the game. ColorDefault (the gruid zero value) is
// reserved to mean "terminal default", so ours start at 1.
const (
	ColorPlayer gruid.Color = iota + 1
	ColorNode
	ColorEdge
	ColorData
	ColorTrace
)

// styleManager implements gtcell.StyleManager, mapping gruid's abstract
// colors to concrete tcell ones.
type styleManager struct{}

func (styleManager) GetStyle(st gruid.Style) tcell.Style {
	ts := tcell.StyleDefault
	switch st.Fg {
	case ColorPlayer:
		ts = ts.Foreground(tcell.ColorGreen)
	case ColorNode:
		ts = ts.Foreground(tcell.ColorAqua)
	case ColorEdge:
		ts = ts.Foreground(tcell.ColorGray)
	case ColorData:
		ts = ts.Foreground(tcell.ColorYellow)
	case ColorTrace:
		ts = ts.Foreground(tcell.ColorRed)
	default:
		ts = ts.Foreground(tcell.ColorReset)
	}
	switch st.Bg {
	default:
		ts = ts.Background(tcell.ColorReset)
	}
	return ts
}
