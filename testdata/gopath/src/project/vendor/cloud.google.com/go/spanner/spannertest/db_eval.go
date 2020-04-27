/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spannertest

// This file contains the part of the Spanner fake that evaluates expressions.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/spanner/spansql"
)

// evalContext represents the context for evaluating an expression.
type evalContext struct {
	table  *table // may be nil
	row    row    // set if table is set, only during expr evaluation
	params queryParams
}

func (d *database) evalSelect(sel spansql.Select, params queryParams, aux []spansql.Expr) (ri *resultIter, evalErr error) {
	// TODO: weave this in below.
	if len(sel.From) == 0 && sel.Where == nil {
		// Simple expressions.
		ri := &resultIter{}
		ec := evalContext{
			params: params,
		}
		var r row
		for _, e := range sel.List {
			ci, err := ec.colInfo(e)
			if err != nil {
				return nil, err
			}
			// TODO: set column names?
			ri.Cols = append(ri.Cols, ci)

			x, err := ec.evalExpr(e)
			if err != nil {
				return nil, err
			}
			r = append(r, x)
		}
		ri.rows = []resultRow{{data: r}}
		return ri, nil
	}

	if len(sel.From) != 1 {
		return nil, fmt.Errorf("selecting from more than one table not supported")
	}
	tableName := sel.From[0].Table
	t, err := d.table(tableName)
	if err != nil {
		return nil, err
	}

	ri = &resultIter{}

	// Handle COUNT(*) specially.
	// TODO: Handle aggregation more generally.
	if len(sel.List) == 1 && isCountStar(sel.List[0]) {
		// Replace the `COUNT(*)` with `1`, then aggregate on the way out.
		sel.List[0] = spansql.IntegerLiteral(1)
		defer func() {
			if evalErr != nil {
				return
			}
			count := int64(len(ri.rows))
			ri.rows = []resultRow{
				{data: []interface{}{count}},
			}
		}()
	}

	// TODO: Support table sampling.

	t.mu.Lock()
	defer t.mu.Unlock()
	ec := evalContext{
		table:  t,
		params: params,
	}

	for _, e := range sel.List {
		ci, err := ec.colInfo(e)
		if err != nil {
			return nil, err
		}
		// TODO: deal with ci.Name == ""?
		ri.Cols = append(ri.Cols, ci)
	}
	for _, r := range t.rows {
		ec.row = r

		// See if we want this row.
		if sel.Where != nil {
			b, err := ec.evalBoolExpr(sel.Where)
			if err != nil {
				return nil, err
			}
			if !b {
				continue
			}
		}

		// Evaluate SELECT expression list on the row.
		out, err := ec.evalExprList(sel.List)
		if err != nil {
			return nil, err
		}
		a, err := ec.evalExprList(aux)
		if err != nil {
			return nil, err
		}

		ri.rows = append(ri.rows, resultRow{data: out, aux: a})
	}

	return ri, nil
}

func (ec evalContext) evalExprList(list []spansql.Expr) ([]interface{}, error) {
	var out []interface{}
	for _, e := range list {
		x, err := ec.evalExpr(e)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}

func (ec evalContext) evalBoolExpr(be spansql.BoolExpr) (bool, error) {
	switch be := be.(type) {
	default:
		return false, fmt.Errorf("unhandled BoolExpr %T", be)
	case spansql.BoolLiteral:
		return bool(be), nil
	case spansql.ID, spansql.Paren:
		e, err := ec.evalExpr(be)
		if err != nil {
			return false, err
		}
		if e == nil { // NULL is a false boolean.
			return false, nil
		}
		b, ok := e.(bool)
		if !ok {
			return false, fmt.Errorf("got %T, want bool", e)
		}
		return b, nil
	case spansql.LogicalOp:
		var lhs, rhs bool
		var err error
		if be.LHS != nil {
			lhs, err = ec.evalBoolExpr(be.LHS)
			if err != nil {
				return false, err
			}
		}
		rhs, err = ec.evalBoolExpr(be.RHS)
		if err != nil {
			return false, err
		}
		switch be.Op {
		case spansql.And:
			return lhs && rhs, nil
		case spansql.Or:
			return lhs || rhs, nil
		case spansql.Not:
			return !rhs, nil
		default:
			return false, fmt.Errorf("unhandled LogicalOp %d", be.Op)
		}
	case spansql.ComparisonOp:
		var lhs, rhs interface{}
		var err error
		lhs, err = ec.evalExpr(be.LHS)
		if err != nil {
			return false, err
		}
		rhs, err = ec.evalExpr(be.RHS)
		if err != nil {
			return false, err
		}
		switch be.Op {
		default:
			return false, fmt.Errorf("TODO: ComparisonOp %d", be.Op)
		case spansql.Lt:
			return compareVals(lhs, rhs) < 0, nil
		case spansql.Le:
			return compareVals(lhs, rhs) <= 0, nil
		case spansql.Gt:
			return compareVals(lhs, rhs) > 0, nil
		case spansql.Ge:
			return compareVals(lhs, rhs) >= 0, nil
		case spansql.Eq:
			return compareVals(lhs, rhs) == 0, nil
		case spansql.Ne:
			return compareVals(lhs, rhs) != 0, nil
		case spansql.Like, spansql.NotLike:
			left, ok := lhs.(string)
			if !ok {
				// TODO: byte works here too?
				return false, fmt.Errorf("LHS of LIKE is %T, not string", lhs)
			}
			right, ok := rhs.(string)
			if !ok {
				// TODO: byte works here too?
				return false, fmt.Errorf("RHS of LIKE is %T, not string", rhs)
			}

			match := evalLike(left, right)
			if be.Op == spansql.NotLike {
				match = !match
			}
			return match, nil
		case spansql.Between, spansql.NotBetween:
			rhs2, err := ec.evalExpr(be.RHS2)
			if err != nil {
				return false, err
			}
			b := compareVals(rhs, lhs) <= 0 && compareVals(lhs, rhs2) <= 0
			if be.Op == spansql.NotBetween {
				b = !b
			}
			return b, nil
		}
	case spansql.IsOp:
		lhs, err := ec.evalExpr(be.LHS)
		if err != nil {
			return false, err
		}
		var b bool
		switch rhs := be.RHS.(type) {
		default:
			return false, fmt.Errorf("unhandled IsOp %T", rhs)
		case spansql.BoolLiteral:
			lhsBool, ok := lhs.(bool)
			if !ok {
				return false, fmt.Errorf("non-bool value %T on LHS for %s", lhs, be.SQL())
			}
			b = (lhsBool == bool(rhs))
		case spansql.NullLiteral:
			b = (lhs == nil)
		}
		if be.Neg {
			b = !b
		}
		return b, nil
	}
}

func (ec evalContext) evalExpr(e spansql.Expr) (interface{}, error) {
	switch e := e.(type) {
	default:
		return nil, fmt.Errorf("TODO: evalExpr(%s %T)", e.SQL(), e)
	case spansql.ID:
		return ec.evalID(e)
	case spansql.Param:
		v, ok := ec.params[string(e)]
		if !ok {
			return 0, fmt.Errorf("unbound param %s", e.SQL())
		}
		return v, nil
	case spansql.IntegerLiteral:
		return int64(e), nil
	case spansql.FloatLiteral:
		return float64(e), nil
	case spansql.StringLiteral:
		return string(e), nil
	case spansql.BytesLiteral:
		return []byte(e), nil
	case spansql.NullLiteral:
		return nil, nil
	case spansql.BoolLiteral:
		return bool(e), nil
	case spansql.Paren:
		return ec.evalExpr(e.Expr)
	case spansql.LogicalOp:
		return ec.evalBoolExpr(e)
	case spansql.IsOp:
		return ec.evalBoolExpr(e)
	}
}

func (ec evalContext) evalID(id spansql.ID) (interface{}, error) {
	// TODO: look beyond column names.
	if ec.table == nil {
		return nil, fmt.Errorf("identifier %s when not SELECTing on a table is not supported", string(id))
	}
	i, ok := ec.table.colIndex[string(id)]
	if !ok {
		return nil, fmt.Errorf("couldn't resolve identifier %s", string(id))
	}
	return ec.row.copyDataElem(i), nil
}

func evalLimit(lim spansql.Limit, params queryParams) (int64, error) {
	switch lim := lim.(type) {
	case spansql.IntegerLiteral:
		return int64(lim), nil
	case spansql.Param:
		return paramAsInteger(lim, params)
	default:
		return 0, fmt.Errorf("LIMIT with %T not supported", lim)
	}
}

func paramAsInteger(p spansql.Param, params queryParams) (int64, error) {
	v, ok := params[string(p)]
	if !ok {
		return 0, fmt.Errorf("unbound param %s", p.SQL())
	}
	switch v := v.(type) {
	default:
		return 0, fmt.Errorf("can't interpret parameter %s value of type %T as integer", p.SQL(), v)
	case int64:
		return v, nil
	case string:
		x, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("bad int64 string %q: %v", v, err)
		}
		return x, nil
	}
}

func compareVals(x, y interface{}) int {
	// NULL is always the minimum possible value.
	if x == nil && y == nil {
		return 0
	} else if x == nil {
		return -1
	} else if y == nil {
		return 1
	}

	// TODO: coerce between comparable types (e.g. int64/float64).

	switch x := x.(type) {
	default:
		panic(fmt.Sprintf("unhandled comparison on %T", x))
	case bool:
		// false < true
		y := y.(bool)
		if !x && y {
			return -1
		} else if x && !y {
			return 1
		}
		return 0
	case int64:
		if s, ok := y.(string); ok {
			var err error
			y, err = strconv.ParseInt(s, 10, 64)
			if err != nil {
				panic(fmt.Sprintf("bad int64 string %q: %v", s, err))
			}
		}
		y := y.(int64)
		if x < y {
			return -1
		} else if x > y {
			return 1
		}
		return 0
	case float64:
		y := y.(float64)
		if x < y {
			return -1
		} else if x > y {
			return 1
		}
		return 0
	case string:
		// This handles DATE too.
		return strings.Compare(x, y.(string))
	}
}

func (ec evalContext) colInfo(e spansql.Expr) (colInfo, error) {
	// TODO: more types
	switch e := e.(type) {
	case spansql.IntegerLiteral:
		return colInfo{Type: spansql.Type{Base: spansql.Int64}}, nil
	case spansql.StringLiteral:
		return colInfo{Type: spansql.Type{Base: spansql.String}}, nil
	case spansql.BytesLiteral:
		return colInfo{Type: spansql.Type{Base: spansql.Bytes}}, nil
	case spansql.LogicalOp, spansql.ComparisonOp, spansql.IsOp:
		return colInfo{Type: spansql.Type{Base: spansql.Bool}}, nil
	case spansql.ID:
		// TODO: support more than only naming a table column.
		name := string(e)
		if ec.table != nil {
			if i, ok := ec.table.colIndex[name]; ok {
				return ec.table.cols[i], nil
			}
		}
	case spansql.Paren:
		return ec.colInfo(e.Expr)
	case spansql.NullLiteral:
		// There isn't necessarily something sensible here.
		// Empirically, though, the real Spanner returns Int64.
		return colInfo{Type: spansql.Type{Base: spansql.Int64}}, nil
	}
	return colInfo{}, fmt.Errorf("can't deduce column type from expression [%s]", e.SQL())
}

func evalLike(str, pat string) bool {
	/*
		% matches any number of chars.
		_ matches a single char.
		TODO: handle escaping
	*/

	// Lean on regexp for simplicity.
	pat = regexp.QuoteMeta(pat)
	pat = strings.Replace(pat, "%", ".*", -1)
	pat = strings.Replace(pat, "_", ".", -1)
	match, err := regexp.MatchString(pat, str)
	if err != nil {
		panic(fmt.Sprintf("internal error: constructed bad regexp /%s/: %v", pat, err))
	}
	return match
}

func isCountStar(e spansql.Expr) bool {
	f, ok := e.(spansql.Func)
	if !ok {
		return false
	}
	if f.Name != "COUNT" || len(f.Args) != 1 {
		return false
	}
	return f.Args[0] == spansql.Star
}
