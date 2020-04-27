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

// This file contains the implementation of the Spanner fake itself,
// namely the part behind the RPC interface.

// TODO: missing transactionality in a serious way!

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	structpb "github.com/golang/protobuf/ptypes/struct"

	"cloud.google.com/go/spanner/spansql"
)

type database struct {
	mu      sync.Mutex
	tables  map[string]*table
	indexes map[string]struct{} // only record their existence
}

type table struct {
	mu sync.Mutex

	// Information about the table columns.
	// They are reordered on table creation so the primary key columns come first.
	cols     []colInfo
	colIndex map[string]int // col name to index
	pkCols   int            // number of primary key columns (may be 0)

	// Rows are stored in primary key order.
	rows []row
}

// colInfo represents information about a column in a table or result set.
type colInfo struct {
	Name string
	Type spansql.Type
}

/*
row represents a list of data elements.

The mapping between Spanner types and Go types internal to this package are:
	BOOL		bool
	INT64		int64
	FLOAT64		float64
	STRING		string
	BYTES		[]byte
	DATE		string (RFC 3339 date; "YYYY-MM-DD")
	TIMESTAMP	TODO
	ARRAY<T>	[]T
	STRUCT		TODO
*/
type row []interface{}

func (r row) copyDataElem(index int) interface{} {
	v := r[index]
	if is, ok := v.([]interface{}); ok {
		// Deep-copy array values.
		v = append([]interface{}(nil), is...)
	}
	return v
}

// copyData returns a copy of a subset of a row.
func (r row) copyData(indexes []int) row {
	if len(indexes) == 0 {
		return nil
	}
	dst := make(row, 0, len(indexes))
	for _, i := range indexes {
		dst = append(dst, r.copyDataElem(i))
	}
	return dst
}

func (d *database) ApplyDDL(stmt spansql.DDLStmt) *status.Status {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Lazy init.
	if d.tables == nil {
		d.tables = make(map[string]*table)
	}
	if d.indexes == nil {
		d.indexes = make(map[string]struct{})
	}

	switch stmt := stmt.(type) {
	default:
		return status.Newf(codes.Unimplemented, "unhandled DDL statement type %T", stmt)
	case spansql.CreateTable:
		if _, ok := d.tables[stmt.Name]; ok {
			return status.Newf(codes.AlreadyExists, "table %s already exists", stmt.Name)
		}

		// TODO: check stmt.Interleave details.

		// Move primary keys first, preserving their order.
		pk := make(map[string]int)
		for i, kp := range stmt.PrimaryKey {
			pk[kp.Column] = -1000 + i
		}
		sort.SliceStable(stmt.Columns, func(i, j int) bool {
			a, b := pk[stmt.Columns[i].Name], pk[stmt.Columns[j].Name]
			return a < b
		})

		t := &table{
			colIndex: make(map[string]int),
			pkCols:   len(pk),
		}
		for _, cd := range stmt.Columns {
			if st := t.addColumn(cd); st.Code() != codes.OK {
				return st
			}
		}
		for col := range pk {
			if _, ok := t.colIndex[col]; !ok {
				return status.Newf(codes.InvalidArgument, "primary key column %q not in table", col)
			}
		}
		d.tables[stmt.Name] = t
		return nil
	case spansql.CreateIndex:
		if _, ok := d.indexes[stmt.Name]; ok {
			return status.Newf(codes.AlreadyExists, "index %s already exists", stmt.Name)
		}
		d.indexes[stmt.Name] = struct{}{}
		return nil
	case spansql.DropTable:
		if _, ok := d.tables[stmt.Name]; !ok {
			return status.Newf(codes.NotFound, "no table named %s", stmt.Name)
		}
		// TODO: check for indexes on this table.
		delete(d.tables, stmt.Name)
		return nil
	case spansql.DropIndex:
		if _, ok := d.indexes[stmt.Name]; !ok {
			return status.Newf(codes.NotFound, "no index named %s", stmt.Name)
		}
		delete(d.indexes, stmt.Name)
		return nil
	case spansql.AlterTable:
		t, ok := d.tables[stmt.Name]
		if !ok {
			return status.Newf(codes.NotFound, "no table named %s", stmt.Name)
		}
		switch alt := stmt.Alteration.(type) {
		default:
			return status.Newf(codes.Unimplemented, "unhandled DDL table alteration type %T", alt)
		case spansql.AddColumn:
			if alt.Def.NotNull {
				return status.Newf(codes.InvalidArgument, "new non-key columns cannot be NOT NULL")
			}
			if st := t.addColumn(alt.Def); st.Code() != codes.OK {
				return st
			}
			return nil
		}
	}

}

func (d *database) table(tbl string) (*table, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	t, ok := d.tables[tbl]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no table named %s", tbl)
	}
	return t, nil
}

// writeValues executes a write option (Insert, Update, etc.).
func (d *database) writeValues(tbl string, cols []string, values []*structpb.ListValue, f func(t *table, colIndexes []int, r row) error) error {
	t, err := d.table(tbl)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	colIndexes, err := t.colIndexes(cols)
	if err != nil {
		return err
	}
	revIndex := make(map[int]int) // table index to col index
	for j, i := range colIndexes {
		revIndex[i] = j
	}

	for pki := 0; pki < t.pkCols; pki++ {
		_, ok := revIndex[pki]
		if !ok {
			return status.Errorf(codes.InvalidArgument, "primary key column %s not included in write", t.cols[pki].Name)
		}
	}

	for _, vs := range values {
		if len(vs.Values) != len(colIndexes) {
			return status.Errorf(codes.InvalidArgument, "row of %d values can't be written to %d columns", len(vs.Values), len(colIndexes))
		}

		r := make(row, len(t.cols))
		for j, v := range vs.Values {
			i := colIndexes[j]

			x, err := valForType(v, t.cols[i].Type)
			if err != nil {
				return err
			}

			r[i] = x
		}
		// TODO: enforce NOT NULL?

		if err := f(t, colIndexes, r); err != nil {
			return err
		}
	}

	return nil
}

func (d *database) Insert(tbl string, cols []string, values []*structpb.ListValue) error {
	return d.writeValues(tbl, cols, values, func(t *table, colIndexes []int, r row) error {
		pk := r[:t.pkCols]
		rowNum, found := t.rowForPK(pk)
		if found {
			// TODO: how do we return `ALREADY_EXISTS`?
			return status.Errorf(codes.Unknown, "row already in table")
		}
		t.insertRow(rowNum, r)
		return nil
	})
}

func (d *database) Update(tbl string, cols []string, values []*structpb.ListValue) error {
	return d.writeValues(tbl, cols, values, func(t *table, colIndexes []int, r row) error {
		if t.pkCols == 0 {
			return status.Errorf(codes.InvalidArgument, "cannot update table %s with no columns in primary key", tbl)
		}
		pk := r[:t.pkCols]
		rowNum, found := t.rowForPK(pk)
		if !found {
			// TODO: is this the right way to return `NOT_FOUND`?
			return status.Errorf(codes.NotFound, "row not in table")
		}

		for _, i := range colIndexes {
			t.rows[rowNum][i] = r[i]
		}
		return nil
	})
}

func (d *database) InsertOrUpdate(tbl string, cols []string, values []*structpb.ListValue) error {
	return d.writeValues(tbl, cols, values, func(t *table, colIndexes []int, r row) error {
		pk := r[:t.pkCols]
		rowNum, found := t.rowForPK(pk)
		if !found {
			// New row; do an insert.
			t.insertRow(rowNum, r)
		} else {
			// Existing row; do an update.
			for _, i := range colIndexes {
				t.rows[rowNum][i] = r[i]
			}
		}
		return nil
	})
}

// TODO: Replace

func (d *database) Delete(table string, keys []*structpb.ListValue, keyRanges keyRangeList, all bool) error {
	t, err := d.table(table)
	if err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if all {
		t.rows = nil
		return nil
	}

	for _, key := range keys {
		pk, err := t.primaryKey(key.Values)
		if err != nil {
			return err
		}
		// Not an error if the key does not exist.
		rowNum, found := t.rowForPK(pk)
		if found {
			copy(t.rows[rowNum:], t.rows[rowNum+1:])
			t.rows = t.rows[:len(t.rows)-1]
		}
	}

	for _, r := range keyRanges {
		r.startKey, err = t.primaryKeyPrefix(r.start.Values)
		if err != nil {
			return err
		}
		r.endKey, err = t.primaryKeyPrefix(r.end.Values)
		if err != nil {
			return err
		}
		startRow, endRow := t.findRange(r)
		if n := endRow - startRow; n > 0 {
			copy(t.rows[startRow:], t.rows[endRow:])
			t.rows = t.rows[:len(t.rows)-n]
		}
	}

	return nil
}

// resultIter is returned by reads and queries.
// Use its Next method to iterate over the result rows.
type resultIter struct {
	// Cols is the metadata about the returned data.
	Cols []colInfo

	// rows holds the result data itself.
	rows []resultRow
}

type resultRow struct {
	data []interface{}

	// aux is any auxiliary values evaluated for the row.
	// When a query has an ORDER BY clause, this will contain the values for those expressions.
	aux []interface{}
}

func (ri *resultIter) Next() ([]interface{}, bool) {
	if len(ri.rows) == 0 {
		return nil, false
	}
	res := ri.rows[0]
	ri.rows = ri.rows[1:]
	return res.data, true
}

func (ri *resultIter) add(src row, colIndexes []int) {
	ri.rows = append(ri.rows, resultRow{
		data: src.copyData(colIndexes),
	})
}

// readTable executes a read option (Read, ReadAll).
func (d *database) readTable(table string, cols []string, f func(*table, *resultIter, []int) error) (*resultIter, error) {
	t, err := d.table(table)
	if err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	colIndexes, err := t.colIndexes(cols)
	if err != nil {
		return nil, err
	}

	ri := &resultIter{}
	for _, i := range colIndexes {
		ri.Cols = append(ri.Cols, t.cols[i])
	}
	return ri, f(t, ri, colIndexes)
}

func (d *database) Read(tbl string, cols []string, keys []*structpb.ListValue, limit int64) (*resultIter, error) {
	return d.readTable(tbl, cols, func(t *table, ri *resultIter, colIndexes []int) error {
		for _, key := range keys {
			pk, err := t.primaryKey(key.Values)
			if err != nil {
				return err
			}
			// Not an error if the key does not exist.
			rowNum, found := t.rowForPK(pk)
			if !found {
				continue
			}
			ri.add(t.rows[rowNum], colIndexes)
			if limit > 0 && len(ri.rows) >= int(limit) {
				break
			}
		}
		return nil
	})
}

func (d *database) ReadAll(tbl string, cols []string, limit int64) (*resultIter, error) {
	return d.readTable(tbl, cols, func(t *table, ri *resultIter, colIndexes []int) error {
		for _, r := range t.rows {
			ri.add(r, colIndexes)
			if limit > 0 && len(ri.rows) >= int(limit) {
				break
			}
		}
		return nil
	})
}

type queryParams map[string]interface{}

func (d *database) Query(q spansql.Query, params queryParams) (*resultIter, error) {
	// If there's an ORDER BY clause, prepare the list of auxiliary data we need.
	// This is provided to evalSelect to evaluate with each row.
	var aux []spansql.Expr
	var desc []bool
	if len(q.Order) > 0 {
		if len(q.Select.From) == 0 {
			return nil, fmt.Errorf("ORDER BY doesn't work without a table")
		}

		for _, o := range q.Order {
			aux = append(aux, o.Expr)
			desc = append(desc, o.Desc)
		}
	}

	ri, err := d.evalSelect(q.Select, params, aux)
	if err != nil {
		return nil, err
	}
	if len(q.Order) > 0 {
		sort.Slice(ri.rows, func(one, two int) bool {
			r1, r2 := ri.rows[one], ri.rows[two]
			for i := range r1.aux {
				cmp := compareVals(r1.aux[i], r2.aux[i])
				if desc[i] {
					cmp = -cmp
				}
				if cmp == 0 {
					continue
				}
				return cmp < 0
			}
			return false
		})
	}
	if q.Limit != nil {
		lim, err := evalLimit(q.Limit, params)
		if err != nil {
			return nil, err
		}
		if n := int(lim); n < len(ri.rows) {
			ri.rows = ri.rows[:n]
		}
	}
	return ri, nil
}

func (t *table) addColumn(cd spansql.ColumnDef) *status.Status {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.rows) > 0 {
		if cd.NotNull {
			// TODO: what happens in this case?
			return status.Newf(codes.Unimplemented, "can't add NOT NULL columns to non-empty tables yet")
		}
		for i := range t.rows {
			t.rows[i] = append(t.rows[i], nil)
		}
	}

	t.cols = append(t.cols, colInfo{
		Name: cd.Name,
		Type: cd.Type,
	})
	t.colIndex[cd.Name] = len(t.cols) - 1

	return nil
}

func (t *table) insertRow(rowNum int, r row) {
	t.rows = append(t.rows, nil)
	copy(t.rows[rowNum+1:], t.rows[rowNum:])
	t.rows[rowNum] = r
}

// findRange finds the rows included in the key range,
// reporting it as a half-open interval.
// r.startKey and r.endKey should be populated.
func (t *table) findRange(r *keyRange) (int, int) {
	// TODO: This is incorrect for primary keys with descending order.
	// It might be sufficient for the caller to switch start/end in that case.

	// startRow is the first row matching the range.
	startRow := sort.Search(len(t.rows), func(i int) bool {
		return rowCmp(r.startKey, t.rows[i][:t.pkCols]) <= 0
	})
	if startRow == len(t.rows) {
		return startRow, startRow
	}
	if !r.startClosed && rowCmp(r.startKey, t.rows[startRow][:t.pkCols]) == 0 {
		startRow++
	}

	// endRow is one more than the last row matching the range.
	endRow := sort.Search(len(t.rows), func(i int) bool {
		return rowCmp(r.endKey, t.rows[i][:t.pkCols]) < 0
	})
	if !r.endClosed && rowCmp(r.endKey, t.rows[endRow-1][:t.pkCols]) == 0 {
		endRow--
	}

	return startRow, endRow
}

// colIndexes returns the indexes for the named columns.
func (t *table) colIndexes(cols []string) ([]int, error) {
	var is []int
	for _, col := range cols {
		i, ok := t.colIndex[col]
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "column %s not in table", col)
		}
		is = append(is, i)
	}
	return is, nil
}

// primaryKey constructs the internal representation of a primary key.
// The list of given values must be in 1:1 correspondence with the primary key of the table.
func (t *table) primaryKey(values []*structpb.Value) ([]interface{}, error) {
	if len(values) != t.pkCols {
		return nil, status.Errorf(codes.InvalidArgument, "primary key length mismatch: got %d values, table has %d", len(values), t.pkCols)
	}
	return t.primaryKeyPrefix(values)
}

// primaryKeyPrefix constructs the internal representation of a primary key prefix.
func (t *table) primaryKeyPrefix(values []*structpb.Value) ([]interface{}, error) {
	if len(values) > t.pkCols {
		return nil, status.Errorf(codes.InvalidArgument, "primary key length too long: got %d values, table has %d", len(values), t.pkCols)
	}

	var pk []interface{}
	for i, value := range values {
		v, err := valForType(value, t.cols[i].Type)
		if err != nil {
			return nil, err
		}
		pk = append(pk, v)
	}
	return pk, nil
}

// rowForPK returns the index of t.rows that holds the row for the given primary key, and true.
// If the given primary key isn't found, it returns the row that should hold it, and false.
func (t *table) rowForPK(pk []interface{}) (row int, found bool) {
	if len(pk) != t.pkCols {
		panic(fmt.Sprintf("primary key length mismatch: got %d values, table has %d", len(pk), t.pkCols))
	}

	i := sort.Search(len(t.rows), func(i int) bool {
		return rowCmp(pk, t.rows[i][:t.pkCols]) <= 0
	})
	if i == len(t.rows) {
		return i, false
	}
	return i, rowCmp(pk, t.rows[i][:t.pkCols]) == 0
}

// rowCmp compares two rows, returning -1/0/+1.
// This is used for primary key matching and so doesn't support array/struct types.
// a is permitted to be shorter than b.
func rowCmp(a, b []interface{}) int {
	for i := 0; i < len(a); i++ {
		if cmp := compareVals(a[i], b[i]); cmp != 0 {
			return cmp
		}
	}
	return 0
}

func valForType(v *structpb.Value, t spansql.Type) (interface{}, error) {
	if _, ok := v.Kind.(*structpb.Value_NullValue); ok {
		// TODO: enforce NOT NULL constraints?
		return nil, nil
	}

	if lv, ok := v.Kind.(*structpb.Value_ListValue); ok && t.Array {
		et := t // element type
		et.Array = false

		// Construct the non-nil slice for the list.
		arr := make([]interface{}, 0, len(lv.ListValue.Values))
		for _, v := range lv.ListValue.Values {
			x, err := valForType(v, et)
			if err != nil {
				return nil, err
			}
			arr = append(arr, x)
		}
		return arr, nil
	}

	switch t.Base {
	case spansql.Bool:
		bv, ok := v.Kind.(*structpb.Value_BoolValue)
		if ok {
			return bv.BoolValue, nil
		}
	case spansql.Int64:
		// The Spanner protocol encodes int64 as a decimal string.
		sv, ok := v.Kind.(*structpb.Value_StringValue)
		if ok {
			x, err := strconv.ParseInt(sv.StringValue, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("bad int64 string %q: %v", sv.StringValue, err)
			}
			return x, nil
		}
	case spansql.Float64:
		nv, ok := v.Kind.(*structpb.Value_NumberValue)
		if ok {
			return nv.NumberValue, nil
		}
	case spansql.String:
		sv, ok := v.Kind.(*structpb.Value_StringValue)
		if ok {
			return sv.StringValue, nil
		}
	case spansql.Date:
		// The Spanner protocol encodes DATE in RFC 3339 date format.
		sv, ok := v.Kind.(*structpb.Value_StringValue)
		if ok {
			// Store it internally as a string, but validate its value.
			s := sv.StringValue
			if _, err := time.Parse("2006-01-02", s); err != nil {
				return nil, fmt.Errorf("bad DATE string %q: %v", s, err)
			}
			return s, nil
		}
	}
	return nil, fmt.Errorf("unsupported inserting value kind %T into column of type %s", v.Kind, t.SQL())
}

type keyRange struct {
	start, end             *structpb.ListValue
	startClosed, endClosed bool

	// These are populated during an operation
	// when we know what table this keyRange applies to.
	startKey, endKey []interface{}
}

func (r *keyRange) String() string {
	var sb bytes.Buffer // TODO: Switch to strings.Builder when we drop support for Go 1.9.
	if r.startClosed {
		sb.WriteString("[")
	} else {
		sb.WriteString("(")
	}
	fmt.Fprintf(&sb, "%v,%v", r.startKey, r.endKey)
	if r.endClosed {
		sb.WriteString("]")
	} else {
		sb.WriteString(")")
	}
	return sb.String()
}

type keyRangeList []*keyRange
