package parser

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/the-xlang/xxc/lex/tokens"
	"github.com/the-xlang/xxc/pkg/xapi"
	"github.com/the-xlang/xxc/pkg/xtype"
)

type valueEvaluator struct {
	tok   Tok
	model *exprModel
	p     *Parser
}

func strModel(v value) iExpr {
	content := v.expr.(string)
	if israwstr(content) {
		return exprNode{xapi.ToRawStr([]byte(content))}
	}
	return exprNode{xapi.ToStr([]byte(content))}
}

func boolModel(v value) iExpr {
	if v.expr.(bool) {
		return exprNode{tokens.TRUE}
	}
	return exprNode{tokens.FALSE}
}

func getModel(v value) iExpr {
	switch v.expr.(type) {
	case string:
		return strModel(v)
	case bool:
		return boolModel(v)
	default:
		return numericModel(v)
	}
}

func numericModel(v value) iExpr {
	cppId := xtype.CppId(v.data.Type.Id)
	switch t := v.expr.(type) {
	case uint64:
		return exprNode{cppId + "{" + strconv.FormatUint(t, 10) + "}"}
	case int64:
		return exprNode{cppId + "{" + strconv.FormatInt(t, 10) + "}"}
	case float64:
		return exprNode{cppId + "{" + fmt.Sprint(t) + "}"}
	}
	return nil
}

func (ve *valueEvaluator) str() value {
	var v value
	v.constExpr = true
	v.data.Value = ve.tok.Kind
	v.data.Type.Id = xtype.Str
	v.data.Type.Kind = xtype.TypeMap[v.data.Type.Id]
	content := ve.tok.Kind[1 : len(ve.tok.Kind)-1]
	v.expr = content
	v.model = strModel(v)
	ve.model.appendSubNode(v.model)
	return v
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
	return kind, isByte
}

func (ve *valueEvaluator) char() value {
	var v value
	v.constExpr = true
	v.data.Value = ve.tok.Kind
	content, isByte := toCharLiteral(ve.tok.Kind)
	if isByte {
		v.data.Type.Id = xtype.U8
		content = xapi.ToChar(content[0])
	} else { // rune
		v.data.Type.Id = xtype.I32
		content = xapi.ToRune([]byte(content))
	}
	v.data.Type.Kind = xtype.TypeMap[v.data.Type.Id]
	v.expr, _ = strconv.ParseInt(content[2:], 16, 64)
	v.model = exprNode{content}
	ve.model.appendSubNode(v.model)
	return v
}

func (ve *valueEvaluator) bool() value {
	var v value
	v.constExpr = true
	v.data.Value = ve.tok.Kind
	v.data.Type.Id = xtype.Bool
	v.data.Type.Kind = xtype.TypeMap[v.data.Type.Id]
	v.expr = ve.tok.Kind == tokens.TRUE
	v.model = boolModel(v)
	ve.model.appendSubNode(v.model)
	return v
}

func (ve *valueEvaluator) nil() value {
	var v value
	v.constExpr = true
	v.data.Value = ve.tok.Kind
	v.data.Type.Id = xtype.Nil
	v.data.Type.Kind = xtype.TypeMap[v.data.Type.Id]
	v.expr = nil
	v.model = exprNode{ve.tok.Kind}
	ve.model.appendSubNode(v.model)
	return v
}

func (ve *valueEvaluator) float() value {
	var v value
	v.data.Value = ve.tok.Kind
	v.data.Type.Id = xtype.F64
	v.data.Type.Kind = xtype.TypeMap[v.data.Type.Id]
	v.expr, _ = strconv.ParseFloat(v.data.Value, 64)
	return v
}

func (ve *valueEvaluator) integer() value {
	var v value
	v.data.Value = ve.tok.Kind
	var bigint big.Int
	switch {
	case strings.HasPrefix(ve.tok.Kind, "0x"):
		_, _ = bigint.SetString(ve.tok.Kind[2:], 16)
	case strings.HasPrefix(ve.tok.Kind, "0b"):
		_, _ = bigint.SetString(ve.tok.Kind[2:], 2)
	case ve.tok.Kind[0] == '0':
		_, _ = bigint.SetString(ve.tok.Kind[1:], 8)
	default:
		_, _ = bigint.SetString(ve.tok.Kind, 10)
	}
	if bigint.IsInt64() {
		v.expr = bigint.Int64()
	} else {
		v.expr = bigint.Uint64()
	}
	bitize(&v)
	return v
}

func (ve *valueEvaluator) numeric() value {
	var v value
	if isfloat(ve.tok.Kind) {
		v = ve.float()
	} else {
		v = ve.integer()
	}
	v.constExpr = true
	v.model = numericModel(v)
	ve.model.appendSubNode(v.model)
	return v
}

func (ve *valueEvaluator) varId(id string, variable *Var) (v value) {
	variable.Used = true
	v.data.Value = id
	v.data.Type = variable.Type
	v.constExpr = variable.Const
	v.data.Tok = variable.Token
	v.lvalue = true
	if id == tokens.SELF && typeIsPtr(variable.Type) {
		ve.model.appendSubNode(exprNode{xapi.CppSelf})
	} else if v.constExpr {
		v.expr = variable.ExprTag
		v.model = variable.Expr.Model
	} else {
		ve.model.appendSubNode(exprNode{variable.OutId()})
		ve.p.eval.hasError = ve.p.eval.hasError || typeIsVoid(v.data.Type)
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
	v.constExpr = true
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

func (ve *valueEvaluator) typeId(id string, t *Type) (_ value, _ bool) {
	dt, ok := ve.p.realType(t.Type, true)
	if !ok {
		return
	}
	if typeIsStruct(dt) {
		return ve.structId(id, dt.Tag.(*xstruct)), true
	}
	return
}

func (ve *valueEvaluator) id() (_ value, ok bool) {
	id := ve.tok.Kind

	v, _, _ := ve.p.varById(id)
	if v != nil {
		return ve.varId(id, v), true
	}

	f, _, _ := ve.p.FuncById(id)
	if f != nil {
		return ve.funcId(id, f), true
	}

	e, _, _ := ve.p.enumById(id)
	if e != nil {
		return ve.enumId(id, e), true
	}

	s, _, _ := ve.p.structById(id)
	if s != nil {
		return ve.structId(id, s), true
	}

	t, _, _ := ve.p.typeById(id)
	if t != nil {
		return ve.typeId(id, t)
	}

	ve.p.eval.pusherrtok(ve.tok, "id_noexist", id)
	return
}
