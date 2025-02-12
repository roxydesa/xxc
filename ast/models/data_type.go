package models

import (
	"strings"
	"unicode"

	"github.com/the-xlang/xxc/lex/tokens"
	"github.com/the-xlang/xxc/pkg/xapi"
	"github.com/the-xlang/xxc/pkg/xtype"
)

// Size is the represents data type of sizes (array or etc)
type Size = int

// TypeSize is the represents data type sizes with expression
type TypeSize struct {
	N         Size
	Expr      Expr
	AutoSized bool
}

// DataType is data type identifier.
type DataType struct {
	// Tok used for usually *File comparisons.
	// For this reason, you don't use token as value, identifier or etc.
	Tok             Tok
	Id              uint8
	Original        any
	Kind            string
	MultiTyped      bool
	ComponentType   *DataType
	Size            TypeSize
	Tag             any
	DontUseOriginal bool
}

// KindWithOriginalId returns dt.Kind with OriginalId.
func (dt *DataType) KindWithOriginalId() string {
	if dt.Original == nil {
		return dt.Kind
	}
	_, prefix := dt.KindId()
	original := dt.Original.(DataType)
	id, _ := original.KindId()
	return prefix + id
}

// OriginalKindId returns dt.Kind's identifier of official.
//
// Special case is:
//   OriginalKindId() -> "" if DataType has not original
func (dt *DataType) OriginalKindId() string {
	if dt.Original == nil {
		return ""
	}
	t := dt.Original.(DataType)
	id, _ := t.KindId()
	return id
}

// KindId returns dt.Kind's identifier.
func (dt *DataType) KindId() (id, prefix string) {
	if dt.Id == xtype.Map || dt.Id == xtype.Func {
		return dt.Kind, ""
	}
	id = dt.Kind
	runes := []rune(dt.Kind)
	for i, r := range dt.Kind {
		if r == '_' || unicode.IsLetter(r) {
			id = string(runes[i:])
			prefix = string(runes[:i])
			break
		}
	}
	for _, dt := range xtype.TypeMap {
		if dt == id {
			return
		}
	}
	runes = []rune(id)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == ':' && i+1 < len(runes) && runes[i+1] == ':' { // Namespace?
			i++
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			id = string(runes[:i])
			break
		}
	}
	return
}

func (dt *DataType) SetToOriginal() {
	if dt.DontUseOriginal || dt.Original == nil {
		return
	}
	tag := dt.Tag
	switch tag.(type) {
	case Genericable:
		defer func() { dt.Tag = tag }()
	}
	kind := dt.KindWithOriginalId()
	tok := dt.Tok
	*dt = dt.Original.(DataType)
	dt.Kind = kind
	dt.Tok = tok
}

// Pointers returns pointer marks of data type.
func (dt *DataType) Pointers() string {
	for i, run := range dt.Kind {
		if run != '*' {
			return dt.Kind[:i]
		}
	}
	return ""
}

func (dt DataType) String() (s string) {
	dt.SetToOriginal()
	if dt.MultiTyped {
		return dt.MultiTypeString()
	}
	// Remove namespace
	i := strings.LastIndex(dt.Kind, tokens.DOUBLE_COLON)
	if i != -1 {
		dt.Kind = dt.Kind[i+len(tokens.DOUBLE_COLON):]
	}
	pointers := dt.Pointers()
	// Apply pointers
	defer func() {
		var cpp strings.Builder
		for range pointers {
			cpp.WriteString("ptr<")
		}
		cpp.WriteString(s)
		for range pointers {
			cpp.WriteString(">")
		}
		s = cpp.String()
	}()
	dt.Kind = dt.Kind[len(pointers):]
	switch dt.Id {
	case xtype.Slice:
		return dt.SliceString()
	case xtype.Array:
		return dt.ArrayString()
	case xtype.Map:
		return dt.MapString()
	}
	switch dt.Tag.(type) {
	case CompiledStruct:
		return dt.StructString()
	}
	switch dt.Id {
	case xtype.Id, xtype.Enum:
		return xapi.OutId(dt.Kind, dt.Tok.File)
	case xtype.Trait:
		return dt.TraitString()
	case xtype.Struct:
		return dt.StructString()
	case xtype.Func:
		return dt.FuncString()
	default:
		return xtype.CppId(dt.Id)
	}
}

// SliceString returns cpp value of slice data type.
func (dt *DataType) SliceString() string {
	var cpp strings.Builder
	cpp.WriteString("slice<")
	dt.ComponentType.DontUseOriginal = dt.DontUseOriginal
	cpp.WriteString(dt.ComponentType.String())
	cpp.WriteByte('>')
	return cpp.String()
}

// ArrayString returns cpp value of map data type.
func (dt *DataType) ArrayString() string {
	var cpp strings.Builder
	cpp.WriteString("array<")
	dt.ComponentType.DontUseOriginal = dt.DontUseOriginal
	cpp.WriteString(dt.ComponentType.String())
	cpp.WriteByte(',')
	cpp.WriteString(dt.Size.Expr.String())
	cpp.WriteByte('>')
	return cpp.String()
}

// MapString returns cpp value of map data type.
func (dt *DataType) MapString() string {
	var cpp strings.Builder
	types := dt.Tag.([]DataType)
	cpp.WriteString("map<")
	key := types[0]
	key.DontUseOriginal = dt.DontUseOriginal
	cpp.WriteString(key.String())
	cpp.WriteByte(',')
	value := types[1]
	value.DontUseOriginal = dt.DontUseOriginal
	cpp.WriteString(value.String())
	cpp.WriteByte('>')
	return cpp.String()
}

// TraitString returns cpp value of trait data type.
func (dt *DataType) TraitString() string {
	var cpp strings.Builder
	id, _ := dt.KindId()
	cpp.WriteString("trait<")
	cpp.WriteString(xapi.OutId(id, dt.Tok.File))
	cpp.WriteByte('>')
	return cpp.String()
}

// StructString returns cpp value of struct data type.
func (dt *DataType) StructString() string {
	var cpp strings.Builder
	s := dt.Tag.(CompiledStruct)
	cpp.WriteString(s.OutId())
	types := s.Generics()
	if len(types) == 0 {
		return cpp.String()
	}
	cpp.WriteByte('<')
	for _, t := range types {
		t.DontUseOriginal = dt.DontUseOriginal
		cpp.WriteString(t.String())
		cpp.WriteByte(',')
	}
	return cpp.String()[:cpp.Len()-1] + ">"
}

// FuncString returns cpp value of function DataType.
func (dt *DataType) FuncString() string {
	var cpp strings.Builder
	cpp.WriteString("std::function<")
	f := dt.Tag.(*Func)
	f.RetType.Type.DontUseOriginal = dt.DontUseOriginal
	cpp.WriteString(f.RetType.String())
	cpp.WriteByte('(')
	if len(f.Params) > 0 {
		for _, param := range f.Params {
			param.Type.DontUseOriginal = dt.DontUseOriginal
			cpp.WriteString(param.Prototype())
			cpp.WriteByte(',')
		}
		cppStr := cpp.String()[:cpp.Len()-1]
		cpp.Reset()
		cpp.WriteString(cppStr)
	} else {
		cpp.WriteString("void")
	}
	cpp.WriteString(")>")
	return cpp.String()
}

// MultiTypeString returns cpp value of muli-typed DataType.
func (dt *DataType) MultiTypeString() string {
	types := dt.Tag.([]DataType)
	var cpp strings.Builder
	cpp.WriteString("std::tuple<")
	for _, t := range types {
		t.DontUseOriginal = dt.DontUseOriginal
		cpp.WriteString(t.String())
		cpp.WriteByte(',')
	}
	return cpp.String()[:cpp.Len()-1] + ">" + dt.Pointers()
}

// MapKind returns data type kind string of map data type.
func (dt *DataType) MapKind() string {
	types := dt.Tag.([]DataType)
	var kind strings.Builder
	kind.WriteByte('[')
	kind.WriteString(types[0].Kind)
	kind.WriteByte(':')
	kind.WriteString(types[1].Kind)
	kind.WriteByte(']')
	return kind.String()
}
