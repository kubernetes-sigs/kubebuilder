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

package spansql

// This file holds SQL methods for rendering the types in types.go
// as the SQL dialect that this package parses.

import (
	"strconv"
	"strings"
)

func (ct CreateTable) SQL() string {
	str := "CREATE TABLE " + ct.Name + " (\n"
	for _, c := range ct.Columns {
		str += "  " + c.SQL() + ",\n"
	}
	str += ") PRIMARY KEY("
	for i, c := range ct.PrimaryKey {
		if i > 0 {
			str += ", "
		}
		str += c.SQL()
	}
	str += ")"
	if il := ct.Interleave; il != nil {
		str += ",\n  INTERLEAVE IN PARENT " + il.Parent + " ON DELETE " + il.OnDelete.SQL()
	}
	return str
}

func (ci CreateIndex) SQL() string {
	str := "CREATE"
	if ci.Unique {
		str += " UNIQUE"
	}
	if ci.NullFiltered {
		str += " NULL_FILTERED"
	}
	str += " INDEX " + ci.Name + " ON " + ci.Table + "("
	for i, c := range ci.Columns {
		if i > 0 {
			str += ", "
		}
		str += c.SQL()
	}
	str += ")"
	if len(ci.Storing) > 0 {
		str += " STORING ("
		str += strings.Join(ci.Storing, ", ")
		str += ")"
	}
	if ci.Interleave != "" {
		str += ", INTERLEAVE IN " + ci.Interleave
	}
	return str
}

func (dt DropTable) SQL() string {
	return "DROP TABLE " + dt.Name
}

func (di DropIndex) SQL() string {
	return "DROP INDEX " + di.Name
}

func (at AlterTable) SQL() string {
	return "ALTER TABLE " + at.Name + " " + at.Alteration.SQL()
}

func (ac AddColumn) SQL() string {
	return "ADD COLUMN " + ac.Def.SQL()
}

func (dc DropColumn) SQL() string {
	return "DROP COLUMN " + dc.Name
}

func (sod SetOnDelete) SQL() string {
	return "SET ON DELETE " + sod.Action.SQL()
}

func (od OnDelete) SQL() string {
	switch od {
	case NoActionOnDelete:
		return "NO ACTION"
	case CascadeOnDelete:
		return "CASCADE"
	}
	panic("unknown OnDelete")
}

// TODO func (ac AlterColumn) SQL() string { }

func (cd ColumnDef) SQL() string {
	str := cd.Name + " " + cd.Type.SQL()
	if cd.NotNull {
		str += " NOT NULL"
	}
	return str
}

func (t Type) SQL() string {
	str := t.Base.SQL()
	if t.Base == String || t.Base == Bytes {
		str += "("
		if t.Len == MaxLen {
			str += "MAX"
		} else {
			str += strconv.FormatInt(t.Len, 10)
		}
		str += ")"
	}
	if t.Array {
		str = "ARRAY<" + str + ">"
	}
	return str
}

func (tb TypeBase) SQL() string {
	switch tb {
	case Bool:
		return "BOOL"
	case Int64:
		return "INT64"
	case Float64:
		return "FLOAT64"
	case String:
		return "STRING"
	case Bytes:
		return "BYTES"
	case Date:
		return "DATE"
	case Timestamp:
		return "TIMESTAMP"
	}
	panic("unknown TypeBase")
}

func (kp KeyPart) SQL() string {
	str := kp.Column
	if kp.Desc {
		str += " DESC"
	}
	return str
}

func (q Query) SQL() string {
	str := q.Select.SQL()
	if len(q.Order) > 0 {
		str += " ORDER BY "
		for i, o := range q.Order {
			if i > 0 {
				str += ", "
			}
			str += o.SQL()
		}
	}
	if q.Limit != nil {
		str += " LIMIT " + q.Limit.SQL()
	}
	return str
}

func (sel Select) SQL() string {
	str := "SELECT "
	for i, e := range sel.List {
		if i > 0 {
			str += ", "
		}
		str += e.SQL()
	}
	if len(sel.From) > 0 {
		str += " FROM "
		for i, f := range sel.From {
			if i > 0 {
				str += ", "
			}
			str += f.Table
		}
	}
	if sel.Where != nil {
		str += " WHERE " + sel.Where.SQL()
	}
	return str
}

func (o Order) SQL() string {
	str := o.Expr.SQL()
	if o.Desc {
		str += " DESC"
	}
	return str
}

func (lo LogicalOp) SQL() string {
	switch lo.Op {
	default:
		panic("unknown LogicalOp")
	case And:
		return lo.LHS.SQL() + " AND " + lo.RHS.SQL()
	case Or:
		return lo.LHS.SQL() + " OR " + lo.RHS.SQL()
	case Not:
		return "NOT " + lo.RHS.SQL()
	}
}

var compOps = map[ComparisonOperator]string{
	Lt:         "<",
	Le:         "<=",
	Gt:         ">",
	Ge:         ">=",
	Eq:         "=",
	Ne:         "!=",
	Like:       "LIKE",
	NotLike:    "NOT LIKE",
	Between:    "BETWEEN",
	NotBetween: "NOT BETWEEN",
}

func (co ComparisonOp) SQL() string {
	op, ok := compOps[co.Op]
	if !ok {
		panic("unknown ComparisonOp")
	}
	s := co.LHS.SQL() + " " + op + " " + co.RHS.SQL()
	if co.Op == Between || co.Op == NotBetween {
		s += " AND " + co.RHS2.SQL()
	}
	return s
}

func (io IsOp) SQL() string {
	str := io.LHS.SQL() + " IS "
	if io.Neg {
		str += "NOT "
	}
	str += io.RHS.SQL()
	return str
}

func (f Func) SQL() string {
	str := f.Name + "("
	for i, e := range f.Args {
		if i > 0 {
			str += ", "
		}
		str += e.SQL()
	}
	str += ")"
	return str
}

func (p Paren) SQL() string { return "(" + p.Expr.SQL() + ")" }

func (id ID) SQL() string   { return string(id) }
func (p Param) SQL() string { return "@" + string(p) }

func (b BoolLiteral) SQL() string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}

func (n NullLiteral) SQL() string { return "NULL" }
func (StarExpr) SQL() string      { return "*" }

func (il IntegerLiteral) SQL() string { return strconv.Itoa(int(il)) }
func (fl FloatLiteral) SQL() string   { return strconv.FormatFloat(float64(fl), 'g', -1, 64) }

// TODO: provide correct string quote method and use it.

func (sl StringLiteral) SQL() string { return strconv.Quote(string(sl)) }
func (bl BytesLiteral) SQL() string  { return "B" + strconv.Quote(string(bl)) }
