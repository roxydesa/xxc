package parser

import (
	"strings"

	"github.com/the-xlang/xxc/ast/models"
	"github.com/the-xlang/xxc/lex/tokens"
	"github.com/the-xlang/xxc/pkg/xbits"
	"github.com/the-xlang/xxc/pkg/xtype"
)

func arrayExprIsAutoSized(expr models.Expr) bool {
	return len(expr.Processes) == 0 || len(expr.Toks) == 0
}

func isstr(s string) bool {
	return s != "" && (s[0] == '"' || israwstr(s))
}

func israwstr(s string) bool {
	return s != "" && s[0] == '`'
}

func ischar(s string) bool {
	return s != "" && s[0] == '\''
}

func isnil(s string) bool {
	return s == tokens.NIL
}

func isbool(s string) bool {
	return s == tokens.TRUE || s == tokens.FALSE
}

func valIsStructType(v value) bool {
	return v.isType && typeIsStruct(v.data.Type)
}

func valIsEnumType(v value) bool {
	return v.isType && typeIsEnum(v.data.Type)
}

func isBoolExpr(val value) bool {
	switch {
	case typeIsNilCompatible(val.data.Type):
		return true
	case val.data.Type.Id == xtype.Bool && typeIsPure(val.data.Type):
		return true
	}
	return false
}

func isfloat(s string) bool {
	if strings.HasPrefix(s, "0x") {
		return false
	}
	return strings.Contains(s, tokens.DOT) ||
		strings.ContainsAny(s, "eE")
}

func canGetPtr(v value) bool {
	if !v.lvalue {
		return false
	}
	switch v.data.Type.Id {
	case xtype.Func, xtype.Enum:
		return false
	default:
		return v.data.Tok.Id == tokens.Id
	}
}

func valIsStructIns(val value) bool {
	return !val.isType && val.data.Type.Id == xtype.Struct
}

func isForeachIterExpr(val value) bool {
	switch {
	case typeIsSlice(val.data.Type),
		typeIsArray(val.data.Type),
		typeIsMap(val.data.Type):
		return true
	case !typeIsPure(val.data.Type):
		return false
	}
	code := val.data.Type.Id
	return code == xtype.Str
}

func isConstNumeric(v string) bool {
	if v == "" {
		return false
	}
	return v[0] == '-' || (v[0] >= '0' && v[0] <= '9')
}

func isConstExpression(v string) bool {
	return isConstNumeric(v) || isstr(v) || ischar(v) || isnil(v) || isbool(v)
}

func checkIntBit(v models.Data, bit int) bool {
	if bit == 0 {
		return false
	}
	if xtype.IsSignedNumericType(v.Type.Id) {
		return xbits.CheckBitInt(v.Value, bit)
	}
	return xbits.CheckBitUInt(v.Value, bit)
}

func checkFloatBit(v models.Data, bit int) bool {
	if bit == 0 {
		return false
	}
	return xbits.CheckBitFloat(v.Value, bit)
}

/*
func defaultValueOfType(t DataType) string {
	if typeIsNilCompatible(t) {
		return tokens.NIL
	}
	return xtype.DefaultValOfType(t.Id)
}
*/
