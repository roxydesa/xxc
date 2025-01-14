package parser

import (
	"fmt"
	"strings"

	"github.com/the-xlang/xxc/ast/models"
)

type iExpr interface {
	String() string
}

type exprBuildNode struct {
	nodes []iExpr
}

type exprModel struct {
	index int
	nodes []exprBuildNode
}

func newExprModel(processes []Toks) *exprModel {
	m := new(exprModel)
	m.index = 0
	m.nodes = make([]exprBuildNode, len(processes))
	return m
}

func (m *exprModel) appendSubNode(node iExpr) {
	nodes := &m.nodes[m.index].nodes
	*nodes = append(*nodes, node)
}

func (m exprModel) String() string {
	var expr strings.Builder
	for _, node := range m.nodes {
		for _, node := range node.nodes {
			expr.WriteString(node.String())
		}
	}
	return expr.String()
}

func (m *exprModel) Expr() Expr {
	return Expr{Model: m}
}

type exprNode struct {
	value string
}

func (node exprNode) String() string {
	return node.value
}

type anonFuncExpr struct {
	ast     Func
	capture byte
}

func (af anonFuncExpr) String() string {
	var cpp strings.Builder
	cpp.WriteByte('[')
	cpp.WriteByte(af.capture)
	cpp.WriteByte(']')
	cpp.WriteString(paramsToCpp(af.ast.Params))
	cpp.WriteString(" mutable -> ")
	cpp.WriteString(af.ast.RetType.String())
	cpp.WriteByte(' ')
	cpp.WriteString(af.ast.Block.String())
	return cpp.String()
}

type sliceExpr struct {
	dataType DataType
	expr     []iExpr
}

func (a sliceExpr) String() string {
	var cpp strings.Builder
	cpp.WriteString(a.dataType.String())
	cpp.WriteString("({")
	if len(a.expr) == 0 {
		cpp.WriteString("})")
		return cpp.String()
	}
	for _, exp := range a.expr {
		cpp.WriteString(exp.String())
		cpp.WriteByte(',')
	}
	return cpp.String()[:cpp.Len()-1] + "})"
}

type mapExpr struct {
	dataType DataType
	keyExprs []iExpr
	valExprs []iExpr
}

func (m mapExpr) String() string {
	var cpp strings.Builder
	cpp.WriteString(m.dataType.String())
	cpp.WriteByte('{')
	for i, k := range m.keyExprs {
		v := m.valExprs[i]
		cpp.WriteByte('{')
		cpp.WriteString(k.String())
		cpp.WriteByte(',')
		cpp.WriteString(v.String())
		cpp.WriteString("},")
	}
	cpp.WriteByte('}')
	return cpp.String()
}

type genericsExpr struct {
	types []DataType
}

func (ge genericsExpr) String() string {
	if len(ge.types) == 0 {
		return ""
	}
	var cpp strings.Builder
	cpp.WriteByte('<')
	for _, generic := range ge.types {
		cpp.WriteString(generic.String())
		cpp.WriteByte(',')
	}
	return cpp.String()[:cpp.Len()-1] + ">"
}

type argsExpr struct {
	args []models.Arg
}

func (a argsExpr) String() string {
	if len(a.args) == 0 {
		return ""
	}
	var cpp strings.Builder
	for _, arg := range a.args {
		cpp.WriteString(arg.String())
		cpp.WriteByte(',')
	}
	return cpp.String()[:cpp.Len()-1]
}

type callExpr struct {
	generics genericsExpr
	args     argsExpr
}

func (ce callExpr) String() string {
	var cpp strings.Builder
	cpp.WriteString(ce.generics.String())
	cpp.WriteByte('(')
	cpp.WriteString(ce.args.String())
	cpp.WriteByte(')')
	return cpp.String()
}

type multiRetExpr struct {
	models []iExpr
}

func (mre multiRetExpr) String() string {
	var cpp strings.Builder
	cpp.WriteString("std::make_tuple(")
	for _, model := range mre.models {
		cpp.WriteString(model.String())
		cpp.WriteByte(',')
	}
	return cpp.String()[:cpp.Len()-1] + ")"
}

type serieExpr struct {
	exprs []any
}

func (se serieExpr) String() string {
	var exprs strings.Builder
	for _, expr := range se.exprs {
		exprs.WriteString(fmt.Sprint(expr))
	}
	return exprs.String()
}
