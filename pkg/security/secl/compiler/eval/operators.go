// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package eval

import (
	"strings"
)

// OpOverrides defines operator override functions
type OpOverrides struct {
	StringEquals         func(a *StringEvaluator, b *StringEvaluator, opts *Opts, state *State) (*BoolEvaluator, error)
	StringValuesContains func(a *StringEvaluator, b *StringValuesEvaluator, opts *Opts, state *State) (*BoolEvaluator, error)
	StringArrayContains  func(a *StringEvaluator, b *StringArrayEvaluator, opts *Opts, state *State) (*BoolEvaluator, error)
	StringArrayMatches   func(a *StringArrayEvaluator, b *StringValuesEvaluator, opts *Opts, state *State) (*BoolEvaluator, error)
}

// return whether a arithmetic operation is deterministic
func isArithmDeterministic(a Evaluator, b Evaluator, state *State) bool {
	isDc := a.IsDeterministicFor(state.field) || b.IsDeterministicFor(state.field)

	if aField := a.GetField(); aField != "" && state.field != "" && aField != state.field {
		isDc = false
	}
	if bField := b.GetField(); bField != "" && state.field != "" && bField != state.field {
		isDc = false
	}

	return isDc
}

// IntNot - ^int operator
func IntNot(a *IntEvaluator, opts *Opts, state *State) *IntEvaluator {
	isDc := a.IsDeterministicFor(state.field)

	if a.EvalFnc != nil {
		ea := a.EvalFnc

		evalFnc := func(ctx *Context) int {
			return ^ea(ctx)
		}

		return &IntEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight,
			isDeterministic: isDc,
		}
	}

	return &IntEvaluator{
		Value:           ^a.Value,
		Weight:          a.Weight,
		isDeterministic: isDc,
	}
}

// StringEquals evaluates string
func StringEquals(a *StringEvaluator, b *StringEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	// default comparison
	op := func(as string, bs string) bool {
		return as == bs
	}

	if a.Field != "" && b.Field != "" {
		if a.StringCmpOpts.ScalarCaseInsensitive || b.StringCmpOpts.ScalarCaseInsensitive {
			op = strings.EqualFold
		}
	} else if a.Field != "" {
		matcher, err := b.ToStringMatcher(a.StringCmpOpts)
		if err != nil {
			return nil, err
		}

		if matcher != nil {
			op = func(as string, bs string) bool {
				return matcher.Matches(as)
			}
		}
	} else if b.Field != "" {
		matcher, err := a.ToStringMatcher(b.StringCmpOpts)
		if err != nil {
			return nil, err
		}

		if matcher != nil {
			op = func(as string, bs string) bool {
				return matcher.Matches(bs)
			}
		}
	}

	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			return op(ea(ctx), eb(ctx))
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.Value, b.Value

		return &BoolEvaluator{
			Value:           op(ea, eb),
			Weight:          a.Weight + InArrayWeight*len(eb),
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.Value

		if a.Field != "" {
			if err := state.UpdateFieldValues(a.Field, FieldValue{Value: eb, Type: b.ValueType}); err != nil {
				return nil, err
			}
		}

		evalFnc := func(ctx *Context) bool {
			return op(ea(ctx), eb)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + InArrayWeight*len(eb),
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.Value, b.EvalFnc

	if b.Field != "" {
		if err := state.UpdateFieldValues(b.Field, FieldValue{Value: ea, Type: a.ValueType}); err != nil {
			return nil, err
		}
	}

	evalFnc := func(ctx *Context) bool {
		return op(ea, eb(ctx))
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// Not - !true operator
func Not(a *BoolEvaluator, opts *Opts, state *State) *BoolEvaluator {
	isDc := a.IsDeterministicFor(state.field)

	if a.EvalFnc != nil {
		ea := func(ctx *Context) bool {
			return !a.EvalFnc(ctx)
		}

		if state.field != "" && !a.IsDeterministicFor(state.field) {
			ea = func(ctx *Context) bool {
				return true
			}
		}

		return &BoolEvaluator{
			EvalFnc:         ea,
			Weight:          a.Weight,
			isDeterministic: isDc,
		}
	}

	return &BoolEvaluator{
		Value:           !a.Value,
		Weight:          a.Weight,
		isDeterministic: isDc,
	}
}

// Minus - -int operator
func Minus(a *IntEvaluator, opts *Opts, state *State) *IntEvaluator {
	isDc := a.IsDeterministicFor(state.field)

	if a.EvalFnc != nil {
		ea := a.EvalFnc

		evalFnc := func(ctx *Context) int {
			return -ea(ctx)
		}

		return &IntEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight,
			isDeterministic: isDc,
		}
	}

	return &IntEvaluator{
		Value:           -a.Value,
		Weight:          a.Weight,
		isDeterministic: isDc,
	}
}

// StringArrayContains evaluates array of strings against a value
func StringArrayContains(a *StringEvaluator, b *StringArrayEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	op := func(a string, b []string, cmp func(a, b string) bool) bool {
		for _, bs := range b {
			if cmp(a, bs) {
				return true
			}
		}
		return false
	}

	cmp := func(a, b string) bool {
		return a == b
	}

	if a.Field != "" && b.Field != "" {
		if a.StringCmpOpts.ScalarCaseInsensitive || b.StringCmpOpts.ScalarCaseInsensitive {
			cmp = strings.EqualFold
		}
	} else if a.Field != "" && a.StringCmpOpts.ScalarCaseInsensitive {
		cmp = strings.EqualFold
	} else if b.Field != "" {
		matcher, err := a.ToStringMatcher(b.StringCmpOpts)
		if err != nil {
			return nil, err
		}

		if matcher != nil {
			cmp = func(a, b string) bool {
				return matcher.Matches(b)
			}
		}
	}

	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			return op(ea(ctx), eb(ctx), cmp)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.Value, b.Values

		return &BoolEvaluator{
			Value:           op(ea, eb, cmp),
			Weight:          a.Weight + InArrayWeight*len(eb),
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.Values

		if a.Field != "" {
			for _, value := range eb {
				if err := state.UpdateFieldValues(a.Field, FieldValue{Value: value, Type: ScalarValueType}); err != nil {
					return nil, err
				}
			}
		}

		evalFnc := func(ctx *Context) bool {
			return op(ea(ctx), eb, cmp)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + InArrayWeight*len(eb),
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.Value, b.EvalFnc

	if b.Field != "" {
		if err := state.UpdateFieldValues(b.Field, FieldValue{Value: ea, Type: a.ValueType}); err != nil {
			return nil, err
		}
	}

	evalFnc := func(ctx *Context) bool {
		return op(a.Value, eb(ctx), cmp)
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// StringValuesContains evaluates a string against values
func StringValuesContains(a *StringEvaluator, b *StringValuesEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	if err := b.Compile(a.StringCmpOpts); err != nil {
		return nil, err
	}

	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			values := eb(ctx)
			return values.Matches(ea(ctx))
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.Value, b.Values

		return &BoolEvaluator{
			Value:           eb.Matches(ea),
			Weight:          a.Weight + InArrayWeight*len(eb.fieldValues),
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.Values

		if a.Field != "" {
			for _, value := range eb.fieldValues {
				if err := state.UpdateFieldValues(a.Field, value); err != nil {
					return nil, err
				}
			}
		}

		evalFnc := func(ctx *Context) bool {
			return eb.Matches(ea(ctx))
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + InArrayWeight*len(eb.fieldValues),
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.Value, b.EvalFnc

	evalFnc := func(ctx *Context) bool {
		values := eb(ctx)
		return values.Matches(ea)
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// StringArrayMatches weak comparison, a least one element of a should be in b. a can't contain regexp
func StringArrayMatches(a *StringArrayEvaluator, b *StringValuesEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	if err := b.Compile(a.StringCmpOpts); err != nil {
		return nil, err
	}

	arrayOp := func(a []string, b *StringValues) bool {
		for _, as := range a {
			if b.Matches(as) {
				return true
			}
		}
		return false
	}

	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx), eb(ctx))
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.Values, b.Values

		return &BoolEvaluator{
			Value:           arrayOp(ea, &eb),
			Weight:          a.Weight + InArrayWeight*len(eb.fieldValues),
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.Values

		if a.Field != "" {
			for _, value := range eb.fieldValues {
				if err := state.UpdateFieldValues(a.Field, value); err != nil {
					return nil, err
				}
			}
		}

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx), &eb)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + InArrayWeight*len(eb.fieldValues),
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.Values, b.EvalFnc

	evalFnc := func(ctx *Context) bool {
		return arrayOp(ea, eb(ctx))
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// IntArrayMatches weak comparison, a least one element of a should be in b
func IntArrayMatches(a *IntArrayEvaluator, b *IntArrayEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	arrayOp := func(a []int, b []int) bool {
		for _, va := range a {
			for _, vb := range b {
				if va == vb {
					return true
				}
			}
		}
		return false
	}

	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx), eb(ctx))
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.Values, b.Values

		return &BoolEvaluator{
			Value:           arrayOp(ea, eb),
			Weight:          a.Weight + InArrayWeight*len(eb),
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.Values

		if a.Field != "" {
			for _, value := range b.Values {
				if err := state.UpdateFieldValues(a.Field, FieldValue{Value: value}); err != nil {
					return nil, err
				}
			}
		}

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx), eb)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + InArrayWeight*len(eb),
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.Values, b.EvalFnc

	evalFnc := func(ctx *Context) bool {
		return arrayOp(ea, eb(ctx))
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// ArrayBoolContains evaluates array of bool against a value
func ArrayBoolContains(a *BoolEvaluator, b *BoolArrayEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	arrayOp := func(a bool, b []bool) bool {
		for _, v := range b {
			if v == a {
				return true
			}
		}
		return false
	}
	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx), eb(ctx))
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.Value, b.Values

		return &BoolEvaluator{
			Value:           arrayOp(ea, eb),
			Weight:          a.Weight + InArrayWeight*len(eb),
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.Values

		if a.Field != "" {
			for _, value := range eb {
				if err := state.UpdateFieldValues(a.Field, FieldValue{Value: value}); err != nil {
					return nil, err
				}
			}
		}

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx), eb)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + InArrayWeight*len(eb),
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.Value, b.EvalFnc

	if b.Field != "" {
		if err := state.UpdateFieldValues(b.Field, FieldValue{Value: ea}); err != nil {
			return nil, err
		}
	}

	evalFnc := func(ctx *Context) bool {
		return arrayOp(ea, eb(ctx))
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// CIDREquals evaluates CIDR ranges
func CIDREquals(a *CIDREvaluator, b *CIDREvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	op := func(as IPMatcher, bs IPMatcher) bool {
		return as.Matches(bs)
	}

	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			return op(ea(ctx).IPMatcher, eb(ctx).IPMatcher)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.cidrMatcher, b.cidrMatcher

		return &BoolEvaluator{
			Value:           op(ea, eb),
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.cidrMatcher

		if a.Field != "" {
			if err := state.UpdateFieldValues(a.Field, FieldValue{Value: eb, Type: b.ValueType, IPMatcher: b.cidrMatcher}); err != nil {
				return nil, err
			}
		}

		evalFnc := func(ctx *Context) bool {
			return op(ea(ctx).IPMatcher, eb)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.cidrMatcher, b.EvalFnc

	if b.Field != "" {
		if err := state.UpdateFieldValues(b.Field, FieldValue{Value: ea, Type: a.ValueType, IPMatcher: a.cidrMatcher}); err != nil {
			return nil, err
		}
	}

	evalFnc := func(ctx *Context) bool {
		return op(ea, eb(ctx).IPMatcher)
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// CIDRValuesContainsWithOpFunction evaluates a CIDR against a list of CIDRs
func CIDRValuesContainsWithOpFunction(a *CIDREvaluator, b *CIDRValuesEvaluator, opts *Opts, state *State, arrayOp func(a IPMatcher, b *CIDRValues) bool) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx).IPMatcher, eb(ctx))
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.cidrMatcher, b.Values

		return &BoolEvaluator{
			Value:           arrayOp(ea, &eb),
			Weight:          a.Weight + InArrayWeight*len(eb.fieldValues),
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.Values

		if a.Field != "" {
			for _, value := range eb.fieldValues {
				if err := state.UpdateFieldValues(a.Field, value); err != nil {
					return nil, err
				}
			}
		}

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx).IPMatcher, &eb)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + InArrayWeight*len(eb.fieldValues),
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.cidrMatcher, b.EvalFnc

	evalFnc := func(ctx *Context) bool {
		return arrayOp(ea, eb(ctx))
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// CIDRValuesContains evaluates a CIDR against a list of CIDRs
func CIDRValuesContains(a *CIDREvaluator, b *CIDRValuesEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	arrayOp := func(a IPMatcher, b *CIDRValues) bool {
		return b.Matches(a)
	}

	return CIDRValuesContainsWithOpFunction(a, b, opts, state, arrayOp)
}

// AllCIDRValuesContains evaluates a CIDR against a list of CIDRs
func AllCIDRValuesContains(a *CIDREvaluator, b *CIDRValuesEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	arrayOp := func(a IPMatcher, b *CIDRValues) bool {
		return b.MatchesAll(a)
	}

	return CIDRValuesContainsWithOpFunction(a, b, opts, state, arrayOp)
}

// CIDRValuesMatchesWithOpFunction compares a and b following the provided arrayOp
func CIDRValuesMatchesWithOpFunction(a *CIDRValuesEvaluator, b *CIDRValuesEvaluator, opts *Opts, state *State, arrayOp func(a *CIDRValues, b *CIDRValues) bool) (*BoolEvaluator, error) {
	isDc := isArithmDeterministic(a, b, state)

	if a.EvalFnc != nil && b.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.EvalFnc

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx), eb(ctx))
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + b.Weight,
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc == nil && b.EvalFnc == nil {
		ea, eb := a.Values, b.Values

		return &BoolEvaluator{
			Value:           arrayOp(&ea, &eb),
			Weight:          a.Weight + InArrayWeight*len(eb.fieldValues),
			isDeterministic: isDc,
		}, nil
	}

	if a.EvalFnc != nil {
		ea, eb := a.EvalFnc, b.Values

		if a.Field != "" {
			for _, value := range eb.fieldValues {
				if err := state.UpdateFieldValues(a.Field, value); err != nil {
					return nil, err
				}
			}
		}

		evalFnc := func(ctx *Context) bool {
			return arrayOp(ea(ctx), &eb)
		}

		return &BoolEvaluator{
			EvalFnc:         evalFnc,
			Weight:          a.Weight + InArrayWeight*len(eb.fieldValues),
			isDeterministic: isDc,
		}, nil
	}

	ea, eb := a.Values, b.EvalFnc

	evalFnc := func(ctx *Context) bool {
		return arrayOp(&ea, eb(ctx))
	}

	return &BoolEvaluator{
		EvalFnc:         evalFnc,
		Weight:          b.Weight,
		isDeterministic: isDc,
	}, nil
}

// CIDRValuesMatches weak comparison, at least one element of a should be in b.
func CIDRValuesMatches(a *CIDRValuesEvaluator, b *CIDRValuesEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	arrayOp := func(a *CIDRValues, b *CIDRValues) bool {
		for _, as := range a.GetFieldValues() {
			if b.Matches(as.IPMatcher) {
				return true
			}
		}
		return false
	}
	return CIDRValuesMatchesWithOpFunction(a, b, opts, state, arrayOp)
}

// AllCIDRValuesMatches ensures that all values from a and b match.
func AllCIDRValuesMatches(a *CIDRValuesEvaluator, b *CIDRValuesEvaluator, opts *Opts, state *State) (*BoolEvaluator, error) {
	arrayOp := func(a *CIDRValues, b *CIDRValues) bool {
		for _, as := range a.GetFieldValues() {
			if !b.MatchesAll(as.IPMatcher) {
				return false
			}
		}
		return true
	}
	return CIDRValuesMatchesWithOpFunction(a, b, opts, state, arrayOp)
}
