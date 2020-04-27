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

// This file holds the type definitions for the SQL dialect.

import (
	"math"
)

// CreateTable represents a CREATE TABLE statement.
// https://cloud.google.com/spanner/docs/data-definition-language#create_table
type CreateTable struct {
	Name       string
	Columns    []ColumnDef
	PrimaryKey []KeyPart
	Interleave *Interleave
}

// Interleave represents an interleave clause of a CREATE TABLE statement.
type Interleave struct {
	Parent   string
	OnDelete OnDelete
}

// CreateIndex represents a CREATE INDEX statement.
// https://cloud.google.com/spanner/docs/data-definition-language#create-index
type CreateIndex struct {
	Name    string
	Table   string
	Columns []KeyPart

	Unique       bool
	NullFiltered bool

	Storing    []string
	Interleave string
}

// DropTable represents a DROP TABLE statement.
// https://cloud.google.com/spanner/docs/data-definition-language#drop_table
type DropTable struct{ Name string }

// DropIndex represents a DROP INDEX statement.
// https://cloud.google.com/spanner/docs/data-definition-language#drop-index
type DropIndex struct{ Name string }

// AlterTable represents an ALTER TABLE statement.
// https://cloud.google.com/spanner/docs/data-definition-language#alter_table
type AlterTable struct {
	Name       string
	Alteration TableAlteration
}

// TableAlteration is satisfied by AddColumn, DropColumn and SetOnDelete.
type TableAlteration interface {
	isTableAlteration()
	SQL() string
}

func (AddColumn) isTableAlteration()   {}
func (DropColumn) isTableAlteration()  {}
func (SetOnDelete) isTableAlteration() {}

//func (AlterColumn) isTableAlteration() {}

type AddColumn struct{ Def ColumnDef }
type DropColumn struct{ Name string }
type SetOnDelete struct{ Action OnDelete }

type OnDelete int

const (
	NoActionOnDelete OnDelete = iota
	CascadeOnDelete
)

/* TODO
type AlterColumn struct {
}
*/

// ColumnDef represents a column definition as part of a CREATE TABLE
// or ALTER TABLE statement.
type ColumnDef struct {
	Name    string
	Type    Type
	NotNull bool
}

// Type represents a column type.
type Type struct {
	Array bool
	Base  TypeBase // Bool, Int64, Float64, String, Bytes, Date, Timestamp
	Len   int64    // if Base is String or Bytes; may be MaxLen
}

// MaxLen is a sentinel for Type's Len field, representing the MAX value.
const MaxLen = math.MaxInt64

type TypeBase int

const (
	Bool TypeBase = iota
	Int64
	Float64
	String
	Bytes
	Date
	Timestamp
)

// KeyPart represents a column specification as part of a primary key or index definition.
type KeyPart struct {
	Column string
	Desc   bool
}

// Query represents a query statement.
// https://cloud.google.com/spanner/docs/query-syntax#sql-syntax
type Query struct {
	Select Select
	Order  []Order
	Limit  Limit
}

// Select represents a SELECT statement.
// https://cloud.google.com/spanner/docs/query-syntax#select-list
type Select struct {
	List  []Expr
	From  []SelectFrom
	Where BoolExpr
	// TODO: GroupBy, Having
}

type SelectFrom struct {
	// This only supports a FROM clause directly from a table.
	Table       string
	TableSample *TableSample
}

type Order struct {
	Expr Expr
	Desc bool
}

type TableSample struct {
	Method   TableSampleMethod
	Size     Expr
	SizeType TableSampleSizeType
}

type TableSampleMethod int

const (
	Bernoulli TableSampleMethod = iota
	Reservoir
)

type TableSampleSizeType int

const (
	PercentTableSample TableSampleSizeType = iota
	RowsTableSample
)

type BoolExpr interface {
	isBoolExpr()
	Expr
}

type Expr interface {
	isExpr()
	SQL() string
}

type Limit interface {
	isLimit()
	SQL() string
}

type LogicalOp struct {
	Op       LogicalOperator
	LHS, RHS BoolExpr // only RHS is set for Not
}

func (LogicalOp) isBoolExpr() {}
func (LogicalOp) isExpr()     {}

type LogicalOperator int

const (
	And LogicalOperator = iota
	Or
	Not
)

type ComparisonOp struct {
	LHS, RHS Expr
	Op       ComparisonOperator

	// RHS2 is the third operand for BETWEEN.
	// "<LHS> BETWEEN <RHS> AND <RHS2>".
	RHS2 Expr
}

func (ComparisonOp) isBoolExpr() {}
func (ComparisonOp) isExpr()     {}

type ComparisonOperator int

const (
	Lt ComparisonOperator = iota
	Le
	Gt
	Ge
	Eq
	Ne // both "!=" and "<>"
	Like
	NotLike
	Between
	NotBetween
)

type IsOp struct {
	LHS Expr
	Neg bool
	RHS IsExpr
}

func (IsOp) isBoolExpr() {}
func (IsOp) isExpr()     {}

type IsExpr interface {
	isIsExpr()
	isExpr()
	SQL() string
}

// Func represents a function call.
type Func struct {
	Name string
	Args []Expr

	// TODO: various functions permit as-expressions, which might warrant different types in here.
}

func (Func) isBoolExpr() {} // possibly bool
func (Func) isExpr()     {}

// Paren represents a parenthesised expression.
type Paren struct {
	Expr Expr
}

func (Paren) isBoolExpr() {} // possibly bool
func (Paren) isExpr()     {}

// ID represents an identifier.
type ID string

func (ID) isBoolExpr() {} // possibly bool
func (ID) isExpr()     {}

// Param represents a query parameter.
type Param string

func (Param) isBoolExpr() {} // possibly bool
func (Param) isExpr()     {}
func (Param) isLimit()    {}

type BoolLiteral bool

const (
	True  = BoolLiteral(true)
	False = BoolLiteral(false)
)

func (BoolLiteral) isBoolExpr() {}
func (BoolLiteral) isIsExpr()   {}
func (BoolLiteral) isExpr()     {}

type NullLiteral int

const Null = NullLiteral(0)

func (NullLiteral) isIsExpr() {}
func (NullLiteral) isExpr()   {}

// IntegerLiteral represents an integer literal.
// https://cloud.google.com/spanner/docs/lexical#integer-literals
type IntegerLiteral int64

func (IntegerLiteral) isLimit() {}
func (IntegerLiteral) isExpr()  {}

// FloatLiteral represents a floating point literal.
// https://cloud.google.com/spanner/docs/lexical#floating-point-literals
type FloatLiteral float64

func (FloatLiteral) isExpr() {}

// StringLiteral represents a string literal.
// https://cloud.google.com/spanner/docs/lexical#string-and-bytes-literals
type StringLiteral string

func (StringLiteral) isExpr() {}

// BytesLiteral represents a bytes literal.
// https://cloud.google.com/spanner/docs/lexical#string-and-bytes-literals
type BytesLiteral string

func (BytesLiteral) isExpr() {}

type StarExpr int

// Star represents a "*" in an expression.
const Star = StarExpr(0)

func (StarExpr) isExpr() {}

// DDL
// https://cloud.google.com/spanner/docs/data-definition-language#ddl_syntax

// DDL represents a Data Definition Language (DDL) file.
type DDL struct {
	List []DDLStmt
}

// DDLStmt is satisfied by a type that can appear in a DDL.
type DDLStmt interface {
	isDDLStmt()
	SQL() string
}

func (CreateTable) isDDLStmt() {}
func (CreateIndex) isDDLStmt() {}
func (AlterTable) isDDLStmt()  {}
func (DropTable) isDDLStmt()   {}
func (DropIndex) isDDLStmt()   {}
