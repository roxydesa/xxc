package parser

import (
	"github.com/the-xlang/xxc/ast/models"
	"github.com/the-xlang/xxc/pkg/x"
)

func getParamMap(params []Param) *paramMap {
	pmap := new(paramMap)
	*pmap = make(paramMap, len(params))
	for i := range params {
		param := &params[i]
		(*pmap)[param.Id] = &paramMapPair{param, nil}
	}
	return pmap
}

type pureArgParser struct {
	p       *Parser
	pmap    *paramMap
	f       *Func
	args    *models.Args
	i       int
	arg     Arg
	errTok  Tok
	m       *exprModel
	paramId string
}

func (pap *pureArgParser) buildArgs() {
	pap.args.Src = make([]Arg, len(*pap.pmap))
	for i, p := range pap.f.Params {
		pair := (*pap.pmap)[p.Id]
		switch {
		case pair.arg != nil:
			pap.args.Src[i] = *pair.arg
		case pair.param.Variadic:
			model := sliceExpr{pair.param.Type, nil}
			model.dataType.Kind = x.Prefix_Slice + model.dataType.Kind // For slice.
			arg := Arg{Expr: Expr{Model: model}}
			pap.args.Src[i] = arg
		}
	}
}

func (pap *pureArgParser) pushVariadicArgs(pair *paramMapPair) {
	model := sliceExpr{pair.param.Type, nil}
	model.dataType.Kind = x.Prefix_Slice + model.dataType.Kind // For slice.
	variadiced := false
	pap.p.parseArg(*pair.param, pair.arg, &variadiced)
	model.expr = append(model.expr, pair.arg.Expr.Model.(iExpr))
	once := false
	for pap.i++; pap.i < len(pap.args.Src); pap.i++ {
		arg := pap.args.Src[pap.i]
		if arg.TargetId != "" {
			pap.i--
			break
		}
		once = true
		pap.p.parseArg(*pair.param, &arg, &variadiced)
		model.expr = append(model.expr, arg.Expr.Model.(iExpr))
	}
	if !once {
		return
	}
	// Variadic argument have one more variadiced expressions.
	if variadiced {
		pap.p.pusherrtok(pap.errTok, "more_args_with_variadiced")
	}
	pair.arg.Expr.Model = model
}

func (pap *pureArgParser) checkPasses() {
	for _, pair := range *pap.pmap {
		if pair.arg == nil && !pair.param.Variadic {
			pap.p.pusherrtok(pap.errTok, "missing_expr_for", pair.param.Id)
		}
	}
}

func (pap *pureArgParser) pushArg() {
	defer func() { pap.i++ }()
	pair := (*pap.pmap)[pap.paramId]
	arg := pap.arg
	pair.arg = &arg
	if pair.param.Variadic {
		pap.pushVariadicArgs(pair)
	} else {
		pap.p.parseArg(*pair.param, pair.arg, nil)
	}
}

func (pap *pureArgParser) parse() {
	if len(pap.args.Src) < len(pap.f.Params) {
		if len(pap.args.Src) == 1 {
			if pap.tryFuncMultiRetAsArgs() {
				return
			}
		}
	}
	pap.pmap = getParamMap(pap.f.Params)
	argCount := 0
	for pap.i < len(pap.args.Src) {
		if argCount >= len(pap.f.Params) {
			pap.p.pusherrtok(pap.errTok, "argument_overflow")
			return
		}
		argCount++
		pap.arg = pap.args.Src[pap.i]
		pap.paramId = pap.f.Params[pap.i].Id
		pap.pushArg()
	}
	pap.checkPasses()
	pap.buildArgs()
}

func (pap *pureArgParser) tryFuncMultiRetAsArgs() bool {
	arg := pap.args.Src[0]
	val, model := pap.p.evalExpr(arg.Expr)
	arg.Expr.Model = model
	if !val.data.Type.MultiTyped {
		return false
	}
	types := val.data.Type.Tag.([]DataType)
	if len(types) < len(pap.f.Params) {
		return false
	} else if len(types) > len(pap.f.Params) {
		return false
	}
	if pap.m != nil {
		fname := pap.m.nodes[pap.m.index].nodes[0]
		pap.m.nodes[pap.m.index].nodes[0] = exprNode{"tuple_as_args"}
		pap.args.Src = make([]Arg, 2)
		pap.args.Src[0] = Arg{Expr: Expr{Model: fname}}
		pap.args.Src[1] = arg
	}
	for i, param := range pap.f.Params {
		rt := types[i]
		pap.p.wg.Add(1)
		val := value{data: models.Data{Type: rt}}
		go pap.p.checkArgType(param, val, arg.Tok)
	}
	return true
}
