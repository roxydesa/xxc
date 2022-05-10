package ast

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/the-xlang/xxc/lex/tokens"
	"github.com/the-xlang/xxc/pkg/xapi"
	"github.com/the-xlang/xxc/pkg/xtype"
)

// Obj is an element of AST.
type Obj struct {
	Tok   Tok
	Value any
}

// Statement is statement.
type Statement struct {
	Tok            Tok
	Val            any
	WithTerminator bool
}

func (s Statement) String() string { return fmt.Sprint(s.Val) }

type Labels []*Label
type Gotos []*Goto

// Block is code block.
type Block struct {
	Parent   *Block
	SubIndex int // Anonymous block sub count
	Tree     []Statement
	Gotos    *Gotos
	Labels   *Labels
	Func     *Func
}

// IndentSpace of blocks.
const IndentSpace = 2

// Indent is indention count.
// This should be manuplate atomic.
var Indent uint32 = 0

// IndentString returns indent space of current block.
func IndentString() string { return strings.Repeat(" ", int(Indent)*IndentSpace) }

// AddIndent adds new indent to IndentString.
func AddIndent() { atomic.AddUint32(&Indent, 1) }

// DoneIndent removes last indent from IndentString.
func DoneIndent() { atomic.SwapUint32(&Indent, Indent-1) }

func (b Block) String() string {
	AddIndent()
	defer func() { DoneIndent() }()
	return ParseBlock(b)
}

// ParseBlock to cxx.
func ParseBlock(b Block) string {
	// Space count per indent.
	var cxx strings.Builder
	cxx.WriteByte('{')
	for _, s := range b.Tree {
		if s.Val == nil {
			continue
		}
		cxx.WriteByte('\n')
		cxx.WriteString(IndentString())
		cxx.WriteString(s.String())
	}
	cxx.WriteByte('\n')
	cxx.WriteString(strings.Repeat(" ", int(Indent-1)*IndentSpace))
	cxx.WriteByte('}')
	return cxx.String()
}

// DataType is data type identifier.
type DataType struct {
	Tok        Tok
	Id         uint8
	Val        string
	MultiTyped bool
	Tag        any
}

func (dt DataType) String() string {
	var cxx strings.Builder
	for i, run := range dt.Val {
		if run == '*' {
			cxx.WriteRune(run)
			continue
		}
		dt.Val = dt.Val[i:]
		break
	}
	if dt.MultiTyped {
		return dt.MultiTypeString() + cxx.String()
	}
	if dt.Val != "" {
		switch {
		case strings.HasPrefix(dt.Val, "[]"):
			pointers := cxx.String()
			cxx.Reset()
			cxx.WriteString("array<")
			dt.Val = dt.Val[2:]
			cxx.WriteString(dt.String())
			cxx.WriteByte('>')
			cxx.WriteString(pointers)
			return cxx.String()
		case dt.Id == xtype.Map && dt.Val[0] == '[':
			pointers := cxx.String()
			types := dt.Tag.([]DataType)
			cxx.Reset()
			cxx.WriteString("map<")
			cxx.WriteString(types[0].String())
			cxx.WriteByte(',')
			cxx.WriteString(types[1].String())
			cxx.WriteByte('>')
			cxx.WriteString(pointers)
			return cxx.String()
		}
	}
	switch dt.Id {
	case xtype.Id, xtype.Enum, xtype.Struct:
		return xapi.OutId(dt.Tok.Kind, dt.Tok.File) + cxx.String()
	case xtype.Func:
		return dt.FuncString() + cxx.String()
	default:
		return xtype.CxxTypeIdFromType(dt.Id) + cxx.String()
	}
}

func (dt *DataType) FuncString() string {
	var cxx strings.Builder
	cxx.WriteString("func<")
	fun := dt.Tag.(Func)
	cxx.WriteString(fun.RetType.String())
	cxx.WriteByte('(')
	if len(fun.Params) > 0 {
		for _, param := range fun.Params {
			cxx.WriteString(param.Prototype())
			cxx.WriteByte(',')
		}
		cxxStr := cxx.String()[:cxx.Len()-1]
		cxx.Reset()
		cxx.WriteString(cxxStr)
	} else {
		cxx.WriteString("void")
	}
	cxx.WriteString(")>")
	return cxx.String()
}

func (dt *DataType) MultiTypeString() string {
	types := dt.Tag.([]DataType)
	var cxx strings.Builder
	cxx.WriteString("std::tuple<")
	for _, t := range types {
		cxx.WriteString(t.String())
		cxx.WriteByte(',')
	}
	return cxx.String()[:cxx.Len()-1] + ">"
}

// Type is type declaration.
type Type struct {
	Pub  bool
	Tok  Tok
	Id   string
	Type DataType
	Desc string
	Used bool
}

func (t Type) String() string {
	var cxx strings.Builder
	cxx.WriteString("typedef ")
	cxx.WriteString(t.Type.String())
	cxx.WriteByte(' ')
	cxx.WriteString(xapi.OutId(t.Id, t.Tok.File))
	cxx.WriteByte(';')
	return cxx.String()
}

// Func is function declaration AST model.
type Func struct {
	Pub     bool
	Tok     Tok
	Id      string
	Params  []Param
	RetType DataType
	Block   Block
}

// DataTypeString returns data type string of function.
func (fc Func) DataTypeString() string {
	var cxx strings.Builder
	cxx.WriteByte('(')
	if len(fc.Params) > 0 {
		for _, p := range fc.Params {
			if p.Variadic {
				cxx.WriteString("...")
			}
			cxx.WriteString(p.Type.String())
			cxx.WriteString(", ")
		}
		cxxStr := cxx.String()[:cxx.Len()-2]
		cxx.Reset()
		cxx.WriteString(cxxStr)
	}
	cxx.WriteByte(')')
	if fc.RetType.Id != xtype.Void {
		cxx.WriteString(fc.RetType.String())
	}
	return cxx.String()
}

// Param is function parameter AST model.
type Param struct {
	Tok      Tok
	Id       string
	Const    bool
	Volatile bool
	Variadic bool
	Type     DataType
	Default  Expr
}

func (p Param) String() string {
	var cxx strings.Builder
	cxx.WriteString(p.Prototype())
	if p.Id != "" {
		cxx.WriteByte(' ')
		cxx.WriteString(xapi.OutId(p.Id, p.Tok.File))
	}
	return cxx.String()
}

// Prototype returns prototype cxx of parameter.
func (p Param) Prototype() string {
	var cxx strings.Builder
	if p.Volatile {
		cxx.WriteString("volatile ")
	}
	if p.Const {
		cxx.WriteString("const ")
	}
	if p.Variadic {
		cxx.WriteString("array<")
		cxx.WriteString(p.Type.String())
		cxx.WriteByte('>')
	} else {
		cxx.WriteString(p.Type.String())
	}
	return cxx.String()
}

// Arg is AST model of argument.
type Arg struct {
	Tok      Tok
	TargetId string
	Expr     Expr
}

// Argument base.
type Args struct {
	Src       []Arg
	Targetted bool
}

func (a Arg) String() string { return a.Expr.String() }

// Expr is AST model of expression.
type Expr struct {
	Toks      []Tok
	Processes [][]Tok
	Model     IExprModel
}

// IExprModel for special expression model to Cxx string.
type IExprModel interface{ String() string }

func (e Expr) String() string {
	if e.Model != nil {
		return e.Model.String()
	}
	var expr strings.Builder
	for _, process := range e.Processes {
		for _, tok := range process {
			switch tok.Id {
			case tokens.Id:
				expr.WriteString(xapi.OutId(tok.Kind, tok.File))
			default:
				expr.WriteString(tok.Kind)
			}
		}
	}
	return expr.String()
}

// ExprStatement is AST model of expression statement in block.
type ExprStatement struct{ Expr Expr }

func (be ExprStatement) String() string {
	var cxx strings.Builder
	cxx.WriteString(be.Expr.String())
	cxx.WriteByte(';')
	return cxx.String()
}

// Value is AST model of constant value.
type Value struct {
	Tok  Tok
	Data string
	Type DataType
}

func (v Value) String() string { return v.Data }

// Ret is return statement AST model.
type Ret struct {
	Tok  Tok
	Expr Expr
}

func (r Ret) String() string {
	var cxx strings.Builder
	cxx.WriteString("return ")
	cxx.WriteString(r.Expr.String())
	cxx.WriteByte(';')
	return cxx.String()
}

// Attribute is attribtue AST model.
type Attribute struct {
	Tok Tok
	Tag Tok
}

func (a Attribute) String() string { return a.Tag.Kind }

// Var is variable declaration AST model.
type Var struct {
	Pub       bool
	DefTok    Tok
	IdTok     Tok
	SetterTok Tok
	Id        string
	Type      DataType
	Val       Expr
	Const     bool
	Volatile  bool
	New       bool
	Tag       any
	Desc      string
	Used      bool
}

func (v Var) String() string {
	var cxx strings.Builder
	if v.Volatile {
		cxx.WriteString("volatile ")
	}
	if v.Const {
		cxx.WriteString("const ")
	}
	cxx.WriteString(v.Type.String())
	cxx.WriteByte(' ')
	cxx.WriteString(xapi.OutId(v.Id, v.IdTok.File))
	cxx.WriteByte('{')
	if v.Val.Processes != nil {
		cxx.WriteString(v.Val.String())
	}
	cxx.WriteByte('}')
	cxx.WriteByte(';')
	return cxx.String()
}

// FieldString returns variable as cxx struct field.
func (v *Var) FieldString() string {
	var cxx strings.Builder
	if v.Volatile {
		cxx.WriteString("volatile ")
	}
	if v.Const {
		cxx.WriteString("const ")
	}
	cxx.WriteString(v.Type.String())
	cxx.WriteByte(' ')
	cxx.WriteString(xapi.OutId(v.Id, v.IdTok.File))
	cxx.WriteByte(';')
	return cxx.String()
}

// AssignSelector is selector for assignment operation.
type AssignSelector struct {
	Var    Var
	Expr   Expr
	Ignore bool
}

func (as AssignSelector) String() string {
	switch {
	case as.Var.New:
		// Returns variable identifier.
		tok := as.Expr.Toks[0]
		return xapi.OutId(tok.Kind, tok.File)
	case as.Ignore:
		return xapi.CxxIgnore
	}
	return as.Expr.String()
}

// Assign is assignment AST model.
type Assign struct {
	Setter      Tok
	SelectExprs []AssignSelector
	ValueExprs  []Expr
	IsExpr      bool
	MultipleRet bool
}

func (a *Assign) cxxSingleAssign() string {
	expr := a.SelectExprs[0]
	if expr.Var.New {
		expr.Var.Val = a.ValueExprs[0]
		s := expr.Var.String()
		return s[:len(s)-1] // Remove statement terminator
	}
	var cxx strings.Builder
	if len(expr.Expr.Toks) != 1 ||
		!xapi.IsIgnoreId(expr.Expr.Toks[0].Kind) {
		cxx.WriteString(expr.String())
		cxx.WriteString(a.Setter.Kind)
	}
	cxx.WriteString(a.ValueExprs[0].String())
	return cxx.String()
}

func (a *Assign) hasSelector() bool {
	for _, s := range a.SelectExprs {
		if !s.Ignore {
			return true
		}
	}
	return false
}

func (a *Assign) cxxMultipleAssign() string {
	var cxx strings.Builder
	if !a.hasSelector() {
		for _, expr := range a.ValueExprs {
			cxx.WriteString(expr.String())
			cxx.WriteByte(';')
		}
		return cxx.String()[:cxx.Len()-1] // Remove last semicolon
	}
	cxx.WriteString(a.cxxNewDefines())
	cxx.WriteString("std::tie(")
	var expCxx strings.Builder
	expCxx.WriteString("std::make_tuple(")
	for i, selector := range a.SelectExprs {
		cxx.WriteString(selector.String())
		cxx.WriteByte(',')
		expCxx.WriteString(a.ValueExprs[i].String())
		expCxx.WriteByte(',')
	}
	str := cxx.String()[:cxx.Len()-1] + ")"
	cxx.Reset()
	cxx.WriteString(str)
	cxx.WriteString(a.Setter.Kind)
	cxx.WriteString(expCxx.String()[:expCxx.Len()-1] + ")")
	return cxx.String()
}

func (a *Assign) cxxMultipleReturn() string {
	var cxx strings.Builder
	cxx.WriteString(a.cxxNewDefines())
	cxx.WriteString("std::tie(")
	for _, selector := range a.SelectExprs {
		if selector.Ignore {
			cxx.WriteString(xapi.CxxIgnore)
			cxx.WriteByte(',')
			continue
		}
		cxx.WriteString(selector.String())
		cxx.WriteByte(',')
	}
	str := cxx.String()[:cxx.Len()-1]
	cxx.Reset()
	cxx.WriteString(str)
	cxx.WriteByte(')')
	cxx.WriteString(a.Setter.Kind)
	cxx.WriteString(a.ValueExprs[0].String())
	return cxx.String()
}

func (a *Assign) cxxNewDefines() string {
	var cxx strings.Builder
	for _, selector := range a.SelectExprs {
		if selector.Ignore || !selector.Var.New {
			continue
		}
		cxx.WriteString(selector.Var.String() + " ")
	}
	return cxx.String()
}

func (a Assign) String() string {
	var cxx strings.Builder
	switch {
	case a.MultipleRet:
		cxx.WriteString(a.cxxMultipleReturn())
	case len(a.SelectExprs) == 1:
		cxx.WriteString(a.cxxSingleAssign())
	default:
		cxx.WriteString(a.cxxMultipleAssign())
	}
	if !a.IsExpr {
		cxx.WriteByte(';')
	}
	return cxx.String()
}

type Free struct {
	Tok  Tok
	Expr Expr
}

func (f Free) String() string {
	var cxx strings.Builder
	cxx.WriteString("delete ")
	cxx.WriteString(f.Expr.String())
	cxx.WriteByte(';')
	return cxx.String()
}

// IterProfile interface for iteration profiles.
type IterProfile interface {
	String(iter Iter) string
}

// WhileProfile is while iteration profile.
type WhileProfile struct{ Expr Expr }

func (wp WhileProfile) String(iter Iter) string {
	var cxx strings.Builder
	cxx.WriteString("while (")
	cxx.WriteString(wp.Expr.String())
	cxx.WriteString(") ")
	cxx.WriteString(iter.Block.String())
	return cxx.String()
}

// ForeachProfile is foreach iteration profile.
type ForeachProfile struct {
	KeyA     Var
	KeyB     Var
	InTok    Tok
	Expr     Expr
	ExprType DataType
}

func (fp ForeachProfile) String(iter Iter) string {
	if !xapi.IsIgnoreId(fp.KeyA.Id) {
		return fp.ForeachString(iter)
	}
	return fp.IterationString(iter)
}

func (fp *ForeachProfile) ClassicString(iter Iter) string {
	var cxx strings.Builder
	cxx.WriteString("foreach<")
	cxx.WriteString(fp.ExprType.String())
	cxx.WriteByte(',')
	cxx.WriteString(fp.KeyA.Type.String())
	if !xapi.IsIgnoreId(fp.KeyB.Id) {
		cxx.WriteByte(',')
		cxx.WriteString(fp.KeyB.Type.String())
	}
	cxx.WriteString(">(")
	cxx.WriteString(fp.Expr.String())
	cxx.WriteString(", [&](")
	cxx.WriteString(fp.KeyA.Type.String())
	cxx.WriteByte(' ')
	cxx.WriteString(xapi.OutId(fp.KeyA.Id, fp.KeyA.IdTok.File))
	if !xapi.IsIgnoreId(fp.KeyB.Id) {
		cxx.WriteByte(',')
		cxx.WriteString(fp.KeyB.Type.String())
		cxx.WriteByte(' ')
		cxx.WriteString(xapi.OutId(fp.KeyB.Id, fp.KeyB.IdTok.File))
	}
	cxx.WriteString(") -> void ")
	cxx.WriteString(iter.Block.String())
	cxx.WriteString(");")
	return cxx.String()
}

func (fp *ForeachProfile) MapString(iter Iter) string {
	var cxx strings.Builder
	cxx.WriteString("foreach<")
	types := fp.ExprType.Tag.([]DataType)
	cxx.WriteString(types[0].String())
	cxx.WriteByte(',')
	cxx.WriteString(types[1].String())
	cxx.WriteString(">(")
	cxx.WriteString(fp.Expr.String())
	cxx.WriteString(", [&](")
	cxx.WriteString(fp.KeyA.Type.String())
	cxx.WriteByte(' ')
	cxx.WriteString(xapi.OutId(fp.KeyA.Id, fp.KeyA.IdTok.File))
	if !xapi.IsIgnoreId(fp.KeyB.Id) {
		cxx.WriteByte(',')
		cxx.WriteString(fp.KeyB.Type.String())
		cxx.WriteByte(' ')
		cxx.WriteString(xapi.OutId(fp.KeyB.Id, fp.KeyB.IdTok.File))
	}
	cxx.WriteString(") -> void ")
	cxx.WriteString(iter.Block.String())
	cxx.WriteString(");")
	return cxx.String()
}

func (fp *ForeachProfile) ForeachString(iter Iter) string {
	switch {
	case fp.ExprType.Val == tokens.STR,
		strings.HasPrefix(fp.ExprType.Val, "[]"):
		return fp.ClassicString(iter)
	case fp.ExprType.Val[0] == '[':
		return fp.MapString(iter)
	}
	return ""
}

func (fp ForeachProfile) IterationString(iter Iter) string {
	var cxx strings.Builder
	cxx.WriteString("for (auto ")
	cxx.WriteString(xapi.OutId(fp.KeyB.Id, fp.KeyB.IdTok.File))
	cxx.WriteString(" : ")
	cxx.WriteString(fp.Expr.String())
	cxx.WriteString(") ")
	cxx.WriteString(iter.Block.String())
	return cxx.String()
}

// Iter is the AST model of iterations.
type Iter struct {
	Tok     Tok
	Block   Block
	Profile IterProfile
}

func (iter Iter) String() string {
	if iter.Profile == nil {
		var cxx strings.Builder
		cxx.WriteString("while (true) ")
		cxx.WriteString(iter.Block.String())
		return cxx.String()
	}
	return iter.Profile.String(iter)
}

// Break is the AST model of break statement.
type Break struct{ Tok Tok }

func (b Break) String() string { return "break;" }

// Continue is the AST model of break statement.
type Continue struct{ Tok Tok }

func (c Continue) String() string { return "continue;" }

// If is the AST model of if expression.
type If struct {
	Tok   Tok
	Expr  Expr
	Block Block
}

func (ifast If) String() string {
	var cxx strings.Builder
	cxx.WriteString("if (")
	cxx.WriteString(ifast.Expr.String())
	cxx.WriteString(") ")
	cxx.WriteString(ifast.Block.String())
	return cxx.String()
}

// ElseIf is the AST model of else if expression.
type ElseIf struct {
	Tok   Tok
	Expr  Expr
	Block Block
}

func (elif ElseIf) String() string {
	var cxx strings.Builder
	cxx.WriteString("else if (")
	cxx.WriteString(elif.Expr.String())
	cxx.WriteString(") ")
	cxx.WriteString(elif.Block.String())
	return cxx.String()
}

// Else is the AST model of else blocks.
type Else struct {
	Tok   Tok
	Block Block
}

func (elseast Else) String() string {
	var cxx strings.Builder
	cxx.WriteString("else ")
	cxx.WriteString(elseast.Block.String())
	return cxx.String()
}

// Comment is the AST model of just comment lines.
type Comment struct{ Content string }

func (c Comment) String() string {
	var cxx strings.Builder
	cxx.WriteString("// ")
	cxx.WriteString(c.Content)
	return cxx.String()
}

// Use is the AST model of use declaration.
type Use struct {
	Tok  Tok
	Path string
}

// CxxEmbed is the AST model of cxx code embed.
type CxxEmbed struct{ Content string }

func (ce CxxEmbed) String() string { return ce.Content }

// Preprocessor is the AST model of preprocessor directives.
type Preprocessor struct {
	Tok     Tok
	Command any
}

func (pp Preprocessor) String() string { return fmt.Sprint(pp.Command) }

// Directive is the AST model of directives.
type Directive struct{ Command any }

func (d Directive) String() string { return fmt.Sprint(d.Command) }

// EnofiDirective is the AST model of enofi directive.
type EnofiDirective struct{}

func (EnofiDirective) String() string { return "" }

// Defer is the AST model of deferred calls.
type Defer struct {
	Tok  Tok
	Expr Expr
}

func (d Defer) String() string { return xapi.ToDeferredCall(d.Expr.String()) }

// Label is the AST model of labels.
type Label struct {
	Tok   Tok
	Label string
	Index int
	Used  bool
	Block *Block
}

func (l Label) String() string { return l.Label + ":;" }

// Goto is the AST model of goto statements.
type Goto struct {
	Tok   Tok
	Label string
	Index int
	Block *Block
}

func (gt Goto) String() string {
	var cxx strings.Builder
	cxx.WriteString("goto ")
	cxx.WriteString(gt.Label)
	cxx.WriteByte(';')
	return cxx.String()
}

// Namespace is the AST model of namespace statements.
type Namespace struct {
	Tok  Tok
	Ids  []string
	Tree []Obj
}

// EnumItem is the AST model of enumerator items.
type EnumItem struct {
	Tok  Tok
	Id   string
	Expr Expr
}

func (ei EnumItem) String() string {
	var cxx strings.Builder
	cxx.WriteString(xapi.OutId(ei.Id, ei.Tok.File))
	cxx.WriteString(" = ")
	cxx.WriteString(ei.Expr.String())
	return cxx.String()
}

// Enum is the AST model of enumerator statements.
type Enum struct {
	Pub   bool
	Tok   Tok
	Id    string
	Type  DataType
	Items []*EnumItem
	Used  bool
	Desc  string
}

// ItemById returns item by id if exist, nil if not.
func (e *Enum) ItemById(id string) *EnumItem {
	for _, item := range e.Items {
		if item.Id == id {
			return item
		}
	}
	return nil
}

func (e Enum) String() string {
	var cxx strings.Builder
	cxx.WriteString("enum ")
	cxx.WriteString(xapi.OutId(e.Id, e.Tok.File))
	cxx.WriteByte(':')
	cxx.WriteString(e.Type.String())
	cxx.WriteString(" {\n")
	AddIndent()
	for _, item := range e.Items {
		cxx.WriteString(IndentString())
		cxx.WriteString(item.String())
		cxx.WriteString(",\n")
	}
	DoneIndent()
	cxx.WriteString("};")
	return cxx.String()
}

// Struct is the AST model of structures.
type Struct struct {
	Tok    Tok
	Id     string
	Pub    bool
	Fields []*Var
}

// ConcurrentCall is the AST model of concurrent calls.
type ConcurrentCall struct {
	Tok  Tok
	Expr Expr
}

func (cc ConcurrentCall) String() string {
	return xapi.ToConcurrentCall(cc.Expr.String())
}

// Try is the AST model of try blocks.
type Try struct {
	Tok   Tok
	Block Block
	Catch Catch
}

func (t Try) String() string {
	var cxx strings.Builder
	cxx.WriteString("try ")
	cxx.WriteString(t.Block.String())
	if t.Catch.Tok.Id == tokens.NA {
		cxx.WriteString(" catch(...) {}")
	} else {
		cxx.WriteByte(' ')
		cxx.WriteString(t.Catch.String())
	}
	return cxx.String()
}

// Catch is the AST model of catch blocks.
type Catch struct {
	Tok   Tok
	Var   Var
	Block Block
}

func (c Catch) String() string {
	var cxx strings.Builder
	cxx.WriteString("catch (")
	if c.Var.Id == "" {
		cxx.WriteString("...")
	} else {
		cxx.WriteString(c.Var.Type.String())
		cxx.WriteByte(' ')
		cxx.WriteString(xapi.OutId(c.Var.Id, c.Tok.File))
	}
	cxx.WriteString(") ")
	cxx.WriteString(c.Block.String())
	return cxx.String()
}
