package models

import "strings"

// Labels is label slice type.
type Labels []*Label

// Gotos is goto slice type.
type Gotos []*Goto

// Label is the AST model of labels.
type Label struct {
	Tok   Tok
	Label string
	Index int
	Used  bool
	Block *Block
}

func (l Label) String() string {
	return l.Label + ":;"
}

// Goto is the AST model of goto statements.
type Goto struct {
	Tok   Tok
	Label string
	Index int
	Block *Block
}

func (gt Goto) String() string {
	var cpp strings.Builder
	cpp.WriteString("goto ")
	cpp.WriteString(gt.Label)
	cpp.WriteByte(';')
	return cpp.String()
}
