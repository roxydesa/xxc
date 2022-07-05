package parser

import (
	"github.com/the-xlang/xxc/lex/tokens"
	"github.com/the-xlang/xxc/pkg/xapi"
	"github.com/the-xlang/xxc/pkg/xbits"
	"github.com/the-xlang/xxc/pkg/xtype"
)

func toRawStrLiteral(literal string) string {
	literal = literal[1 : len(literal)-1] // Remove bounds
	literal = `"(` + literal + `)"`
	literal = xapi.ToRawStr(literal)
	return literal
}

func toCharLiteral(kind string) (string, bool) {
	kind = kind[1 : len(kind)-1]
	isByte := false
	switch {
	case len(kind) == 1 && kind[0] <= 255:
		isByte = true
	case kind[0] == '\\' && kind[1] == 'x':
		isByte = true
	case kind[0] == '\\' && kind[1] >= '0' && kind[1] <= '7':
		isByte = true
	}
	kind = "'" + kind + "'"
	return xapi.ToChar(kind), isByte
}

type valueEvaluator struct {
	tok   Tok
	model *exprModel
	p     *Parser
}

func (p *valueEvaluator) str() value {
	var v value
	v.data.Value = p.tok.Kind
	v.data.Type.Id = xtype.Str
	v.data.Type.Kind = tokens.STR
	if israwstr(p.tok.Kind) {
		p.model.appendSubNode(exprNode{toRawStrLiteral(p.tok.Kind)})
	} else {
		p.model.appendSubNode(exprNode{xapi.ToStr(p.tok.Kind)})
	}
	return v
}

func (ve *valueEvaluator) char() value {
	var v value
	v.data.Value = ve.tok.Kind
	literal, _ := toCharLiteral(ve.tok.Kind)
	v.data.Type.Id = xtype.U8
	v.data.Type.Kind = tokens.U8
	ve.model.appendSubNode(exprNode{literal})
	return v
}

func (ve *valueEvaluator) bool() value {
	var v value
	v.data.Value = ve.tok.Kind
	v.data.Type.Id = xtype.Bool
	v.data.Type.Kind = tokens.BOOL
	ve.model.appendSubNode(exprNode{ve.tok.Kind})
	return v
}

func (ve *valueEvaluator) nil() value {
	var v value
	v.data.Value = ve.tok.Kind
	v.data.Type.Id = xtype.Nil
	v.data.Type.Kind = xtype.NilTypeStr
	ve.model.appendSubNode(exprNode{ve.tok.Kind})
	return v
}

func (ve *valueEvaluator) float() value {
	var v value
	v.data.Value = ve.tok.Kind
	v.data.Type.Id = xtype.F64
	v.data.Type.Kind = tokens.F64
	return v
}

func (ve *valueEvaluator) integer() value {
	var v value
	v.data.Value = ve.tok.Kind
	intbit := xbits.BitsizeType(xtype.Int)
	switch {
	case xbits.CheckBitInt(ve.tok.Kind, intbit):
		v.data.Type.Id = xtype.Int
		v.data.Type.Kind = tokens.INT
	case intbit < xbits.MaxInt && xbits.CheckBitInt(ve.tok.Kind, xbits.MaxInt):
		v.data.Type.Id = xtype.I64
		v.data.Type.Kind = tokens.I64
	default:
		v.data.Type.Id = xtype.U64
		v.data.Type.Kind = tokens.U64
	}
	return v
}

func (ve *valueEvaluator) numeric() value {
	var v value
	if isfloat(ve.tok.Kind) {
		v = ve.float()
	} else {
		v = ve.integer()
	}
	cxxId := xtype.CxxTypeIdFromType(v.data.Type.Id)
	node := exprNode{cxxId + "{" + ve.tok.Kind + "}"}
	ve.model.appendSubNode(node)
	return v
}

func (ve *valueEvaluator) varId(id string, variable *Var) (v value) {
	variable.Used = true
	v.data.Value = id
	v.data.Type = variable.Type
	v.constant = variable.Const
	v.volatile = variable.Volatile
	v.data.Tok = variable.IdTok
	v.lvalue = true
	// If built-in.
	if variable.IdTok.Id == tokens.NA {
		ve.model.appendSubNode(exprNode{xapi.OutId(id, nil)})
	} else {
		ve.model.appendSubNode(exprNode{xapi.OutId(id, variable.IdTok.File)})
	}
	return
}

func (ve *valueEvaluator) funcId(id string, f *function) (v value) {
	f.used = true
	v.data.Value = id
	v.data.Type.Id = xtype.Func
	v.data.Type.Tag = f.Ast
	v.data.Type.Kind = f.Ast.DataTypeString()
	v.data.Tok = f.Ast.Tok
	ve.model.appendSubNode(exprNode{f.outId()})
	return
}

func (ve *valueEvaluator) enumId(id string, e *Enum) (v value) {
	e.Used = true
	v.data.Value = id
	v.data.Type.Id = xtype.Enum
	v.data.Type.Tag = e
	v.data.Type.Kind = e.Id
	v.data.Tok = e.Tok
	v.constant = true
	v.isType = true
	// If built-in.
	if e.Tok.Id == tokens.NA {
		ve.model.appendSubNode(exprNode{xapi.OutId(id, nil)})
	} else {
		ve.model.appendSubNode(exprNode{xapi.OutId(id, e.Tok.File)})
	}
	return
}

func (ve *valueEvaluator) structId(id string, s *xstruct) (v value) {
	s.Used = true
	v.data.Value = id
	v.data.Type.Id = xtype.Struct
	v.data.Type.Tag = s
	v.data.Type.Kind = s.Ast.Id
	v.data.Type.Tok = s.Ast.Tok
	v.data.Tok = s.Ast.Tok
	v.isType = true
	// If built-in.
	if s.Ast.Tok.Id == tokens.NA {
		ve.model.appendSubNode(exprNode{xapi.OutId(id, nil)})
	} else {
		ve.model.appendSubNode(exprNode{xapi.OutId(id, s.Ast.Tok.File)})
	}
	return
}

func (ve *valueEvaluator) id() (_ value, ok bool) {
	id := ve.tok.Kind
	if v, _ := ve.p.varById(id); v != nil {
		return ve.varId(id, v), true
	} else if f, _, _ := ve.p.FuncById(id); f != nil {
		return ve.funcId(id, f), true
	} else if e, _, _ := ve.p.enumById(id); e != nil {
		return ve.enumId(id, e), true
	} else if s, _, _ := ve.p.structById(id); s != nil {
		return ve.structId(id, s), true
	} else {
		ve.p.pusherrtok(ve.tok, "id_noexist", id)
	}
	return
}
