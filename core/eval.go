package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Env map[string][]Value

// Eval normalizes Term to a Value.
func Eval(t Term) Value {
	return evalWith(t, Env{}, false)
}

// AlphaBetaEval alpha-beta-normalizes Term to a Value.
func AlphaBetaEval(t Term) Value {
	return evalWith(t, Env{}, true)
}

func evalWith(t Term, e Env, shouldAlphaNormalize bool) Value {
	switch t := t.(type) {
	case Universe:
		return t
	case Builtin:
		switch t {
		case NaturalBuild:
			return NaturalBuildVal
		case NaturalEven:
			return NaturalEvenVal
		case NaturalFold:
			return NaturalFoldVal
		case NaturalIsZero:
			return NaturalIsZeroVal
		case NaturalOdd:
			return NaturalOddVal
		case NaturalShow:
			return NaturalShowVal
		case NaturalSubtract:
			return NaturalSubtractVal
		case NaturalToInteger:
			return NaturalToIntegerVal
		case IntegerShow:
			return IntegerShowVal
		case IntegerToDouble:
			return IntegerToDoubleVal
		case DoubleShow:
			return DoubleShowVal
		case OptionalBuild:
			return OptionalBuildVal
		case OptionalFold:
			return OptionalFoldVal
		case TextShow:
			return TextShowVal
		case ListBuild:
			return ListBuildVal
		case ListFold:
			return ListFoldVal
		case ListHead:
			return ListHeadVal
		case ListIndexed:
			return ListIndexedVal
		case ListLength:
			return ListLengthVal
		case ListLast:
			return ListLastVal
		case ListReverse:
			return ListReverseVal
		default:
			return t
		}
	case Var:
		if t.Index >= len(e[t.Name]) {
			return t
		}
		return e[t.Name][t.Index]
	case LocalVar:
		return t
	case LambdaTerm:
		v := LambdaValue{
			Label:  t.Label,
			Domain: evalWith(t.Type, e, shouldAlphaNormalize),
			Fn: func(x Value) Value {
				newEnv := Env{}
				for k, v := range e {
					newEnv[k] = v
				}
				newEnv[t.Label] = append([]Value{x}, newEnv[t.Label]...)
				return evalWith(t.Body, newEnv, shouldAlphaNormalize)
			},
		}
		if shouldAlphaNormalize {
			v.Label = "_"
		}
		return v
	case PiTerm:
		v := PiValue{
			Label:  t.Label,
			Domain: evalWith(t.Type, e, shouldAlphaNormalize),
			Range: func(x Value) Value {
				newEnv := Env{}
				for k, v := range e {
					newEnv[k] = v
				}
				newEnv[t.Label] = append([]Value{x}, newEnv[t.Label]...)
				return evalWith(t.Body, newEnv, shouldAlphaNormalize)
			}}
		if shouldAlphaNormalize {
			v.Label = "_"
		}
		return v
	case AppTerm:
		fn := evalWith(t.Fn, e, shouldAlphaNormalize)
		arg := evalWith(t.Arg, e, shouldAlphaNormalize)
		return applyVal(fn, arg)
	case Let:
		newEnv := Env{}
		for k, v := range e {
			newEnv[k] = v
		}

		for _, b := range t.Bindings {
			val := evalWith(b.Value, newEnv, shouldAlphaNormalize)
			newEnv[b.Variable] = append([]Value{val}, newEnv[b.Variable]...)
		}
		return evalWith(t.Body, newEnv, shouldAlphaNormalize)
	case Annot:
		return evalWith(t.Expr, e, shouldAlphaNormalize)
	case DoubleLit:
		return t
	case TextLitTerm:
		var str strings.Builder
		var newChunks ChunkVals
		for _, chunk := range t.Chunks {
			str.WriteString(chunk.Prefix)
			normExpr := evalWith(chunk.Expr, e, shouldAlphaNormalize)
			if text, ok := normExpr.(TextLitVal); ok {
				if len(text.Chunks) != 0 {
					// first chunk gets the rest of str
					str.WriteString(text.Chunks[0].Prefix)
					newChunks = append(newChunks,
						ChunkVal{Prefix: str.String(), Expr: text.Chunks[0].Expr})
					newChunks = append(newChunks,
						text.Chunks[1:]...)
					str.Reset()
				}
				str.WriteString(text.Suffix)

			} else {
				newChunks = append(newChunks, ChunkVal{Prefix: str.String(), Expr: normExpr})
				str.Reset()
			}
		}
		str.WriteString(t.Suffix)
		newSuffix := str.String()

		// Special case: "${<expr>}" → <expr>
		if len(newChunks) == 1 && newChunks[0].Prefix == "" && newSuffix == "" {
			return newChunks[0].Expr
		}

		return TextLitVal{Chunks: newChunks, Suffix: newSuffix}
	case BoolLit:
		return t
	case IfTerm:
		condVal := evalWith(t.Cond, e, shouldAlphaNormalize)
		if condVal == True {
			return evalWith(t.T, e, shouldAlphaNormalize)
		}
		if condVal == False {
			return evalWith(t.F, e, shouldAlphaNormalize)
		}
		tVal := evalWith(t.T, e, shouldAlphaNormalize)
		fVal := evalWith(t.F, e, shouldAlphaNormalize)
		if tVal == True && fVal == False {
			return condVal
		}
		if judgmentallyEqualVals(tVal, fVal) {
			return tVal
		}
		return IfVal{
			Cond: condVal,
			T:    evalWith(t.T, e, shouldAlphaNormalize),
			F:    evalWith(t.F, e, shouldAlphaNormalize),
		}
	case NaturalLit:
		return t
	case IntegerLit:
		return t
	case OpTerm:
		l := evalWith(t.L, e, shouldAlphaNormalize)
		r := evalWith(t.R, e, shouldAlphaNormalize)
		switch t.OpCode {
		case OrOp, AndOp, EqOp, NeOp:
			lb, lok := l.(BoolLit)
			rb, rok := r.(BoolLit)
			switch t.OpCode {
			case OrOp:
				if lok {
					if lb {
						return True
					}
					return r
				}
				if rok {
					if rb {
						return True
					}
					return l
				}
				if judgmentallyEqualVals(l, r) {
					return l
				}
			case AndOp:
				if lok {
					if lb {
						return r
					}
					return False
				}
				if rok {
					if rb {
						return l
					}
					return False
				}
				if judgmentallyEqualVals(l, r) {
					return l
				}
			case EqOp:
				if lok && bool(lb) {
					return r
				}
				if rok && bool(rb) {
					return l
				}
				if judgmentallyEqualVals(l, r) {
					return True
				}
			case NeOp:
				if lok && !bool(lb) {
					return r
				}
				if rok && !bool(rb) {
					return l
				}
				if judgmentallyEqualVals(l, r) {
					return False
				}
			}
		case TextAppendOp:
			return evalWith(
				TextLitTerm{Chunks: Chunks{{Expr: t.L}, {Expr: t.R}}},
				e, shouldAlphaNormalize)
		case ListAppendOp:
			if _, ok := l.(EmptyListVal); ok {
				return r
			}
			if _, ok := r.(EmptyListVal); ok {
				return l
			}
			ll, lok := l.(NonEmptyListVal)
			rl, rok := r.(NonEmptyListVal)
			if lok && rok {
				return append(ll, rl...)
			}
		case PlusOp:
			ln, lok := l.(NaturalLit)
			rn, rok := r.(NaturalLit)
			if lok && rok {
				return NaturalLit(ln + rn)
			}
			if l == NaturalLit(0) {
				return r
			}
			if r == NaturalLit(0) {
				return l
			}
		case TimesOp:
			ln, lok := l.(NaturalLit)
			rn, rok := r.(NaturalLit)
			if lok && rok {
				return NaturalLit(ln * rn)
			}
			if l == NaturalLit(0) {
				return NaturalLit(0)
			}
			if r == NaturalLit(0) {
				return NaturalLit(0)
			}
			if l == NaturalLit(1) {
				return r
			}
			if r == NaturalLit(1) {
				return l
			}
		case RecordMergeOp:
			lR, lOk := l.(RecordLitVal)
			rR, rOk := r.(RecordLitVal)

			if lOk && len(lR) == 0 {
				return r
			}
			if rOk && len(rR) == 0 {
				return l
			}
			if lOk && rOk {
				return mustMergeRecordLitVals(lR, rR)
			}
		case RecordTypeMergeOp:
			lRT, lOk := l.(RecordTypeVal)
			rRT, rOk := r.(RecordTypeVal)

			if lOk && len(lRT) == 0 {
				return r
			}
			if rOk && len(rRT) == 0 {
				return l
			}
			if lOk && rOk {
				result, err := mergeRecordTypes(lRT, rRT)
				if err != nil {
					panic(err) // shouldn't happen for well-typed terms
				}
				return result
			}
		case RightBiasedRecordMergeOp:
			lLit, lOk := l.(RecordLitVal)
			rLit, rOk := r.(RecordLitVal)
			if lOk && len(lLit) == 0 {
				return r
			}
			if rOk && len(rLit) == 0 {
				return l
			}
			if lOk && rOk {
				result := RecordLitVal{}
				for k, v := range lLit {
					result[k] = v
				}
				for k, v := range rLit {
					result[k] = v
				}
				return result
			}
			if judgmentallyEqualVals(l, r) {
				return l
			}
		case ImportAltOp:
			// nothing special
		case EquivOp:
			// nothing special
		case CompleteOp:
			return evalWith(
				Annot{
					Expr: OpTerm{
						OpCode: RightBiasedRecordMergeOp,
						L:      Field{t.L, "default"},
						R:      t.R,
					},
					Annotation: Field{t.L, "Type"},
				},
				e, shouldAlphaNormalize)
		}
		return OpValue{OpCode: t.OpCode, L: l, R: r}
	case EmptyList:
		return EmptyListVal{Type: evalWith(t.Type, e, shouldAlphaNormalize)}
	case NonEmptyList:
		result := make([]Value, len(t))
		for i, t := range t {
			result[i] = evalWith(t, e, shouldAlphaNormalize)
		}
		return NonEmptyListVal(result)
	case Some:
		return SomeVal{evalWith(t.Val, e, shouldAlphaNormalize)}
	case RecordType:
		newRT := RecordTypeVal{}
		for k, v := range t {
			newRT[k] = evalWith(v, e, shouldAlphaNormalize)
		}
		return newRT
	case RecordLit:
		newRT := RecordLitVal{}
		for k, v := range t {
			newRT[k] = evalWith(v, e, shouldAlphaNormalize)
		}
		return newRT
	case ToMap:
		recordVal := evalWith(t.Record, e, shouldAlphaNormalize)
		record, ok := recordVal.(RecordLitVal)
		if ok {
			if len(record) == 0 {
				return EmptyListVal{Type: evalWith(t.Type, e, shouldAlphaNormalize)}
			}
			fieldnames := []string{}
			for k := range record {
				fieldnames = append(fieldnames, k)
			}
			sort.Strings(fieldnames)
			result := make(NonEmptyListVal, len(fieldnames))
			for i, k := range fieldnames {
				result[i] = RecordLitVal{"mapKey": TextLitVal{Suffix: k}, "mapValue": record[k]}
			}
			return result
		}
		return ToMapVal{
			Record: record,
			Type:   evalWith(t.Type, e, shouldAlphaNormalize),
		}
	case Field:
		record := evalWith(t.Record, e, shouldAlphaNormalize)
		for { // simplifications
			if proj, ok := record.(ProjectVal); ok {
				record = proj.Record
				continue
			}
			op, ok := record.(OpValue)
			if ok && op.OpCode == RecordMergeOp {
				if l, ok := op.L.(RecordLitVal); ok {
					if lField, ok := l[t.FieldName]; ok {
						return FieldVal{
							Record: OpValue{
								L:      RecordLitVal{t.FieldName: lField},
								R:      op.R,
								OpCode: RecordMergeOp,
							},
							FieldName: t.FieldName,
						}
					}
					record = op.R
					continue
				}
				if r, ok := op.R.(RecordLitVal); ok {
					if rField, ok := r[t.FieldName]; ok {
						return FieldVal{
							Record: OpValue{
								L:      op.L,
								R:      RecordLitVal{t.FieldName: rField},
								OpCode: RecordMergeOp,
							},
							FieldName: t.FieldName,
						}
					}
					record = op.L
					continue
				}
			}
			if ok && op.OpCode == RightBiasedRecordMergeOp {
				if l, ok := op.L.(RecordLitVal); ok {
					if lField, ok := l[t.FieldName]; ok {
						return FieldVal{
							Record: OpValue{
								L:      RecordLitVal{t.FieldName: lField},
								R:      op.R,
								OpCode: RightBiasedRecordMergeOp,
							},
							FieldName: t.FieldName,
						}
					}
					record = op.R
					continue
				}
				if r, ok := op.R.(RecordLitVal); ok {
					if rField, ok := r[t.FieldName]; ok {
						return rField
					}
					record = op.L
					continue
				}
			}
			break
		}
		if lit, ok := record.(RecordLitVal); ok {
			return lit[t.FieldName]
		}
		return FieldVal{
			Record:    record,
			FieldName: t.FieldName,
		}
	case Project:
		record := evalWith(t.Record, e, shouldAlphaNormalize)
		fieldNames := t.FieldNames
		sort.Strings(fieldNames)
		// simplifications
		for {
			if proj, ok := record.(ProjectVal); ok {
				record = proj.Record
				continue
			}
			op, ok := record.(OpValue)
			if ok && op.OpCode == RightBiasedRecordMergeOp {
				if r, ok := op.R.(RecordLitVal); ok {
					notOverridden := []string{}
					overrides := RecordLitVal{}
					for _, fieldName := range fieldNames {
						if override, ok := r[fieldName]; ok {
							overrides[fieldName] = override
						} else {
							notOverridden = append(notOverridden, fieldName)
						}
					}
					if len(notOverridden) == 0 {
						return overrides
					}
					return OpValue{
						OpCode: RightBiasedRecordMergeOp,
						L: ProjectVal{
							Record:     op.L,
							FieldNames: notOverridden,
						},
						R: overrides,
					}
				}
			}

			break
		}
		if lit, ok := record.(RecordLitVal); ok {
			result := make(RecordLitVal)
			for _, k := range fieldNames {
				result[k] = lit[k]
			}
			return result
		}
		if len(fieldNames) == 0 {
			return RecordLitVal{}
		}
		return ProjectVal{
			Record:     record,
			FieldNames: fieldNames,
		}
	case ProjectType:
		// if `t` typechecks, `t.Selector` has to eval to a
		// RecordTypeVal, so this is safe
		s := evalWith(t.Selector, e, shouldAlphaNormalize).(RecordTypeVal)
		fieldNames := make([]string, 0, len(s))
		for fieldName := range s {
			fieldNames = append(fieldNames, fieldName)
		}
		return evalWith(
			Project{
				Record:     t.Record,
				FieldNames: fieldNames,
			},
			e, shouldAlphaNormalize)
	case UnionType:
		result := make(UnionTypeVal, len(t))
		for k, v := range t {
			if v == nil {
				result[k] = nil
				continue
			}
			result[k] = evalWith(v, e, shouldAlphaNormalize)
		}
		return result
	case Merge:
		handlerVal := evalWith(t.Handler, e, shouldAlphaNormalize)
		unionVal := evalWith(t.Union, e, shouldAlphaNormalize)
		if handlers, ok := handlerVal.(RecordLitVal); ok {
			// TODO: test tricky Field inputs
			if union, ok := unionVal.(AppValue); ok {
				if field, ok := union.Fn.(FieldVal); ok {
					return applyVal(
						handlers[field.FieldName],
						union.Arg,
					)
				}
			}
			if union, ok := unionVal.(FieldVal); ok {
				// empty union alternative
				return handlers[union.FieldName]
			}
		}
		output := MergeVal{
			Handler: handlerVal,
			Union:   unionVal,
		}
		if t.Annotation != nil {
			output.Annotation = evalWith(t.Annotation, e, shouldAlphaNormalize)
		}
		return output
	case Assert:
		return AssertVal{Annotation: evalWith(t.Annotation, e, shouldAlphaNormalize)}
	default:
		panic(fmt.Sprint("unknown term type", t))
	}
}

func applyVal(fn Value, args ...Value) Value {
	out := fn
	for _, arg := range args {
		if f, ok := out.(Callable); ok {
			if result := f.Call(arg); result != nil {
				out = result
				continue
			}
		}
		out = AppValue{Fn: out, Arg: arg}
	}
	return out
}

func mergeRecordTypes(l RecordTypeVal, r RecordTypeVal) (RecordTypeVal, error) {
	var err error
	result := make(RecordTypeVal)
	for k, v := range l {
		result[k] = v
	}
	for k, v := range r {
		if lField, ok := result[k]; ok {
			lSubrecord, Lok := lField.(RecordTypeVal)
			rSubrecord, Rok := v.(RecordTypeVal)
			if !(Lok && Rok) {
				return nil, errors.New("Record mismatch")
			}
			result[k], err = mergeRecordTypes(lSubrecord, rSubrecord)
			if err != nil {
				return nil, err
			}
		} else {
			result[k] = v
		}
	}
	return result, nil
}

func mustMergeRecordLitVals(l RecordLitVal, r RecordLitVal) RecordLitVal {
	output := make(RecordLitVal)
	for k, v := range l {
		output[k] = v
	}
	for k, v := range r {
		if lField, ok := output[k]; ok {
			lSubrecord, Lok := lField.(RecordLitVal)
			rSubrecord, Rok := v.(RecordLitVal)
			if !(Lok && Rok) {
				// typecheck ought to have caught this
				panic("Record mismatch")
			}
			output[k] = mustMergeRecordLitVals(lSubrecord, rSubrecord)
		} else {
			output[k] = v
		}
	}
	return output
}
