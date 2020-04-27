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

/*
Package spannertest contains test helpers for working with Cloud Spanner.

This package is EXPERIMENTAL, and is lacking many features. See the README.md
file in this directory for more details.

In-memory fake

This package has an in-memory fake implementation of spanner. To use it,
create a Server, and then connect to it with no security:
	srv, err := spannertest.NewServer("localhost:0")
	...
	conn, err := grpc.DialContext(ctx, srv.Addr, grpc.WithInsecure())
	...
	client, err := spanner.NewClient(ctx, db, option.WithGRPCConn(conn))
	...

Alternatively, create a Server, then set the SPANNER_EMULATOR_HOST environment
variable and use the regular spanner.NewClient:
	srv, err := spannertest.NewServer("localhost:0")
	...
	os.Setenv("SPANNER_EMULATOR_HOST", srv.Addr)
	client, err := spanner.NewClient(ctx, db)
	...

The same server also supports database admin operations for use with
the cloud.google.com/go/spanner/admin/database/apiv1 package.
*/
package spannertest

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	anypb "github.com/golang/protobuf/ptypes/any"
	emptypb "github.com/golang/protobuf/ptypes/empty"
	structpb "github.com/golang/protobuf/ptypes/struct"
	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
	lropb "google.golang.org/genproto/googleapis/longrunning"
	adminpb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
	spannerpb "google.golang.org/genproto/googleapis/spanner/v1"

	"cloud.google.com/go/spanner/spansql"
)

// Server is an in-memory Cloud Spanner fake.
// It is unauthenticated, non-performant, and only a rough approximation.
type Server struct {
	Addr string

	l   net.Listener
	srv *grpc.Server
	s   *server
}

// server is the real implementation of the fake.
// It is a separate and unexported type so the API won't be cluttered with
// methods that are only relevant to the fake's implementation.
type server struct {
	logf Logger

	db database

	mu       sync.Mutex
	sessions map[string]*session
	lros     map[string]*lro

	// Any unimplemented methods will cause a panic.
	// TODO: Switch to Unimplemented at some point? spannerpb would need regenerating.
	adminpb.DatabaseAdminServer
	spannerpb.SpannerServer
	lropb.OperationsServer
}

type session struct {
	name     string
	creation time.Time

	// This context tracks the lifetime of this session.
	// It is canceled in DeleteSession.
	ctx    context.Context
	cancel func()

	mu           sync.Mutex
	lastUse      time.Time
	transactions map[string]*transaction
}

func (s *session) Proto() *spannerpb.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := &spannerpb.Session{
		Name:                   s.name,
		CreateTime:             timestampProto(s.creation),
		ApproximateLastUseTime: timestampProto(s.lastUse),
	}
	return m
}

// timestampProto returns a valid timestamp.Timestamp,
// or nil if the given time is zero or isn't representable.
func timestampProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	ts, err := ptypes.TimestampProto(t)
	if err != nil {
		return nil
	}
	return ts
}

type transaction struct {
	// TODO: connect this with db.go.
}

func (t *transaction) Commit() error {
	return nil
}

func (t *transaction) Rollback() error {
	return nil
}

func (t *transaction) finish() {
}

// lro represents a Long-Running Operation, generally a schema change.
type lro struct {
	mu    sync.Mutex
	state *lropb.Operation
}

func (l *lro) State() *lropb.Operation {
	l.mu.Lock()
	defer l.mu.Unlock()
	return proto.Clone(l.state).(*lropb.Operation)
}

// Logger is something that can be used for logging.
// It is matched by log.Printf and testing.T.Logf.
type Logger func(format string, args ...interface{})

// NewServer creates a new Server.
// The Server will be listening for gRPC connections, without TLS, on the provided TCP address.
// The resolved address is available in the Addr field.
func NewServer(laddr string) (*Server, error) {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		Addr: l.Addr().String(),
		l:    l,
		srv:  grpc.NewServer(),
		s: &server{
			logf: func(format string, args ...interface{}) {
				log.Printf("spannertest.inmem: "+format, args...)
			},
			sessions: make(map[string]*session),
			lros:     make(map[string]*lro),
		},
	}
	adminpb.RegisterDatabaseAdminServer(s.srv, s.s)
	spannerpb.RegisterSpannerServer(s.srv, s.s)
	lropb.RegisterOperationsServer(s.srv, s.s)

	go s.srv.Serve(s.l)

	return s, nil
}

// SetLogger sets a logger for the server.
// You can use a *testing.T as this argument to collate extra information
// from the execution of the server.
func (s *Server) SetLogger(l Logger) { s.s.logf = l }

// Close shuts down the server.
func (s *Server) Close() {
	s.srv.Stop()
	s.l.Close()
}

func genRandomSession() string {
	var b [4]byte
	rand.Read(b[:])
	return fmt.Sprintf("%x", b)
}

func genRandomTransaction() string {
	var b [6]byte
	rand.Read(b[:])
	return fmt.Sprintf("tx-%x", b)
}

func genRandomOperation() string {
	var b [3]byte
	rand.Read(b[:])
	return fmt.Sprintf("op-%x", b)
}

func (s *server) GetOperation(ctx context.Context, req *lropb.GetOperationRequest) (*lropb.Operation, error) {
	s.mu.Lock()
	lro, ok := s.lros[req.Name]
	s.mu.Unlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "unknown LRO %q", req.Name)
	}
	return lro.State(), nil
}

// UpdateDDL applies the given DDL to the server.
//
// This is a convenience method for tests that may assume an existing schema.
// The more general approach is to dial this server using an admin client, and
// use the UpdateDatabaseDdl RPC method.
func (s *Server) UpdateDDL(ddl spansql.DDL) error {
	ctx := context.Background()
	for _, stmt := range ddl.List {
		if st := s.s.runOneDDL(ctx, stmt); st.Code() != codes.OK {
			return st.Err()
		}
	}
	return nil
}

func (s *server) UpdateDatabaseDdl(ctx context.Context, req *adminpb.UpdateDatabaseDdlRequest) (*lropb.Operation, error) {
	// Parse all the DDL statements first.
	var stmts []spansql.DDLStmt
	for _, s := range req.Statements {
		stmt, err := spansql.ParseDDLStmt(s)
		if err != nil {
			// TODO: check what code the real Spanner returns here.
			return nil, status.Errorf(codes.InvalidArgument, "bad DDL statement %q: %v", s, err)
		}
		stmts = append(stmts, stmt)
	}

	// Nothing should be depending on the exact structure of this,
	// but it is specified in google/spanner/admin/database/v1/spanner_database_admin.proto.
	id := "projects/fake-proj/instances/fake-instance/databases/fake-db/operations/" + genRandomOperation()
	lro := &lro{
		state: &lropb.Operation{
			Name: id,
		},
	}
	s.mu.Lock()
	s.lros[id] = lro
	s.mu.Unlock()

	go lro.Run(s, stmts)
	return lro.State(), nil
}

func (l *lro) Run(s *server, stmts []spansql.DDLStmt) {
	ctx := context.Background()

	for _, stmt := range stmts {
		time.Sleep(100 * time.Millisecond)
		if st := s.runOneDDL(ctx, stmt); st.Code() != codes.OK {
			l.mu.Lock()
			l.state.Done = true
			l.state.Result = &lropb.Operation_Error{st.Proto()}
			l.mu.Unlock()
			return
		}
	}

	l.mu.Lock()
	l.state.Done = true
	l.state.Result = &lropb.Operation_Response{&anypb.Any{}}
	l.mu.Unlock()
}

func (s *server) runOneDDL(ctx context.Context, stmt spansql.DDLStmt) *status.Status {
	return s.db.ApplyDDL(stmt)
}

func (s *server) CreateSession(ctx context.Context, req *spannerpb.CreateSessionRequest) (*spannerpb.Session, error) {
	s.logf("CreateSession(%q)", req.Database)

	id := genRandomSession()
	now := time.Now()
	sess := &session{
		name:         id,
		creation:     now,
		lastUse:      now,
		transactions: make(map[string]*transaction),
	}
	sess.ctx, sess.cancel = context.WithCancel(context.Background())

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	return sess.Proto(), nil
}

func (s *server) GetSession(ctx context.Context, req *spannerpb.GetSessionRequest) (*spannerpb.Session, error) {
	s.mu.Lock()
	sess, ok := s.sessions[req.Name]
	s.mu.Unlock()

	if !ok {
		// TODO: what error does the real Spanner return?
		return nil, status.Errorf(codes.NotFound, "unknown session %q", req.Name)
	}

	return sess.Proto(), nil
}

// TODO: ListSessions

func (s *server) DeleteSession(ctx context.Context, req *spannerpb.DeleteSessionRequest) (*emptypb.Empty, error) {
	s.logf("DeleteSession(%q)", req.Name)

	s.mu.Lock()
	sess, ok := s.sessions[req.Name]
	delete(s.sessions, req.Name)
	s.mu.Unlock()

	if !ok {
		// TODO: what error does the real Spanner return?
		return nil, status.Errorf(codes.NotFound, "unknown session %q", req.Name)
	}

	// Terminate any operations in this session.
	sess.cancel()

	return &emptypb.Empty{}, nil
}

// popTx returns an existing transaction, removing it from the session.
// This is called when a transaction is finishing (Commit, Rollback).
func (s *server) popTx(sessionID, tid string) (tx *transaction, cleanup func(), err error) {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		// TODO: what error does the real Spanner return?
		return nil, nil, status.Errorf(codes.NotFound, "unknown session %q", sessionID)
	}

	sess.mu.Lock()
	sess.lastUse = time.Now()
	tx, ok = sess.transactions[tid]
	if ok {
		delete(sess.transactions, tid)
	}
	sess.mu.Unlock()
	if !ok {
		// TODO: what error does the real Spanner return?
		return nil, nil, status.Errorf(codes.NotFound, "unknown transaction ID %q", tid)
	}
	return tx, tx.finish, nil
}

// readTx returns a transaction for the given session and transaction selector.
// It is used by read/query operations (ExecuteStreamingSql, StreamingRead).
func (s *server) readTx(ctx context.Context, session string, tsel *spannerpb.TransactionSelector) (tx *transaction, cleanup func(), err error) {
	s.mu.Lock()
	sess, ok := s.sessions[session]
	s.mu.Unlock()
	if !ok {
		// TODO: what error does the real Spanner return?
		return nil, nil, status.Errorf(codes.NotFound, "unknown session %q", session)
	}

	sess.mu.Lock()
	sess.lastUse = time.Now()
	sess.mu.Unlock()

	singleUse := func() (*transaction, func(), error) {
		tx := &transaction{}
		return tx, tx.finish, nil
	}
	singleUseReadOnly := func() (*transaction, func(), error) {
		// TODO: figure out a way to make this read-only.
		return singleUse()
	}

	if tsel.GetSelector() == nil {
		return singleUseReadOnly()
	}

	switch sel := tsel.Selector.(type) {
	default:
		return nil, nil, fmt.Errorf("TransactionSelector type %T not supported", sel)
	case *spannerpb.TransactionSelector_SingleUse:
		// Ignore options (e.g. timestamps).
		switch mode := sel.SingleUse.Mode.(type) {
		case *spannerpb.TransactionOptions_ReadOnly_:
			return singleUseReadOnly()
		case *spannerpb.TransactionOptions_ReadWrite_:
			return singleUse()
		default:
			return nil, nil, fmt.Errorf("single use transaction in mode %T not supported", mode)
		}
	case *spannerpb.TransactionSelector_Id:
		id := sel.Id // []byte
		_ = id       // TODO: lookup an existing transaction by ID.
		tx := &transaction{}
		return tx, tx.finish, nil
	}
}

func (s *server) ExecuteSql(ctx context.Context, req *spannerpb.ExecuteSqlRequest) (*spannerpb.ResultSet, error) {
	return nil, status.Errorf(codes.Unimplemented, "ExecuteSql not implemented yet")
}

func (s *server) ExecuteStreamingSql(req *spannerpb.ExecuteSqlRequest, stream spannerpb.Spanner_ExecuteStreamingSqlServer) error {
	tx, cleanup, err := s.readTx(stream.Context(), req.Session, req.Transaction)
	if err != nil {
		return err
	}
	defer cleanup()

	q, err := spansql.ParseQuery(req.Sql)
	if err != nil {
		// TODO: check what code the real Spanner returns here.
		return status.Errorf(codes.InvalidArgument, "bad query: %v", err)
	}

	params := make(queryParams)
	for k, v := range req.GetParams().GetFields() {
		switch v := v.Kind.(type) {
		default:
			return fmt.Errorf("unsupported well-known type value kind %T", v)
		case *structpb.Value_NullValue:
			params[k] = nil
		case *structpb.Value_NumberValue:
			params[k] = v.NumberValue
		case *structpb.Value_StringValue:
			params[k] = v.StringValue
		}

	}

	s.logf("Querying: %s", q.SQL())
	if len(params) > 0 {
		s.logf("        ▹ %v", params)
	}

	ri, err := s.db.Query(q, params)
	if err != nil {
		return err
	}
	return s.readStream(stream.Context(), tx, stream.Send, ri)
}

// TODO: Read

func (s *server) StreamingRead(req *spannerpb.ReadRequest, stream spannerpb.Spanner_StreamingReadServer) error {
	tx, cleanup, err := s.readTx(stream.Context(), req.Session, req.Transaction)
	if err != nil {
		return err
	}
	defer cleanup()

	// Bail out if various advanced features are being used.
	if req.Index != "" {
		return fmt.Errorf("index reads (%q) not supported", req.Index)
	}
	if len(req.ResumeToken) > 0 {
		// This should only happen if we send resume_token ourselves.
		return fmt.Errorf("read resumption not supported")
	}
	if len(req.PartitionToken) > 0 {
		return fmt.Errorf("partition restrictions not supported")
	}

	// TODO: other KeySet types.
	if len(req.KeySet.Ranges) > 0 {
		return fmt.Errorf("reading with ranges not supported")
	}

	var ri *resultIter
	if req.KeySet.All {
		s.logf("Reading all from %s (cols: %v)", req.Table, req.Columns)
		ri, err = s.db.ReadAll(req.Table, req.Columns, req.Limit)
	} else {
		s.logf("Reading %d rows from from %s (cols: %v)", len(req.KeySet.Keys), req.Table, req.Columns)
		ri, err = s.db.Read(req.Table, req.Columns, req.KeySet.Keys, req.Limit)
	}
	if err != nil {
		return err
	}

	// TODO: Figure out the right contexts to use here. There's the session one (sess.ctx),
	// but also this specific RPC one (stream.Context()). Which takes precedence?
	// They appear to be independent.

	return s.readStream(stream.Context(), tx, stream.Send, ri)
}

func (s *server) readStream(ctx context.Context, tx *transaction, send func(*spannerpb.PartialResultSet) error, ri *resultIter) error {
	// Build the result set metadata.
	rsm := &spannerpb.ResultSetMetadata{
		RowType: &spannerpb.StructType{},
		// TODO: transaction info?
	}
	for _, ci := range ri.Cols {
		st, err := spannerTypeFromType(ci.Type)
		if err != nil {
			return err
		}
		rsm.RowType.Fields = append(rsm.RowType.Fields, &spannerpb.StructType_Field{
			Name: ci.Name,
			Type: st,
		})
	}

	for {
		row, ok := ri.Next()
		if !ok {
			break
		}

		values := make([]*structpb.Value, len(row))
		for i, x := range row {
			v, err := spannerValueFromValue(x)
			if err != nil {
				return err
			}
			values[i] = v
		}

		prs := &spannerpb.PartialResultSet{
			Metadata: rsm,
			Values:   values,
		}
		if err := send(prs); err != nil {
			return err
		}

		// ResultSetMetadata is only set for the first PartialResultSet.
		rsm = nil
	}

	return nil
}

func (s *server) BeginTransaction(ctx context.Context, req *spannerpb.BeginTransactionRequest) (*spannerpb.Transaction, error) {
	//s.logf("BeginTransaction(%v)", req)

	s.mu.Lock()
	sess, ok := s.sessions[req.Session]
	s.mu.Unlock()
	if !ok {
		// TODO: what error does the real Spanner return?
		return nil, status.Errorf(codes.NotFound, "unknown session %q", req.Session)
	}

	id := genRandomTransaction()
	tx := &transaction{}

	sess.mu.Lock()
	sess.lastUse = time.Now()
	sess.transactions[id] = tx
	sess.mu.Unlock()

	return &spannerpb.Transaction{Id: []byte(id)}, nil
}

func (s *server) Commit(ctx context.Context, req *spannerpb.CommitRequest) (*spannerpb.CommitResponse, error) {
	//s.logf("Commit(%q, %q)", req.Session, req.Transaction)

	obj, ok := req.Transaction.(*spannerpb.CommitRequest_TransactionId)
	if !ok {
		return nil, fmt.Errorf("unsupported transaction type %T", req.Transaction)
	}
	tid := string(obj.TransactionId)

	tx, cleanup, err := s.popTx(req.Session, tid)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	for _, m := range req.Mutations {
		switch op := m.Operation.(type) {
		default:
			return nil, fmt.Errorf("unsupported mutation operation type %T", op)
		case *spannerpb.Mutation_Insert:
			ins := op.Insert
			err := s.db.Insert(ins.Table, ins.Columns, ins.Values)
			if err != nil {
				return nil, err
			}
		case *spannerpb.Mutation_Update:
			up := op.Update
			err := s.db.Update(up.Table, up.Columns, up.Values)
			if err != nil {
				return nil, err
			}
		case *spannerpb.Mutation_InsertOrUpdate:
			iou := op.InsertOrUpdate
			err := s.db.InsertOrUpdate(iou.Table, iou.Columns, iou.Values)
			if err != nil {
				return nil, err
			}
		case *spannerpb.Mutation_Delete_:
			del := op.Delete
			ks := del.KeySet

			err := s.db.Delete(del.Table, ks.Keys, makeKeyRangeList(ks.Ranges), ks.All)
			if err != nil {
				return nil, err
			}
		}

	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// TODO: return timestamp?
	return &spannerpb.CommitResponse{}, nil
}

func (s *server) Rollback(ctx context.Context, req *spannerpb.RollbackRequest) (*emptypb.Empty, error) {
	s.logf("Rollback(%v)", req)

	tx, cleanup, err := s.popTx(req.Session, string(req.TransactionId))
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := tx.Rollback(); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// TODO: PartitionQuery, PartitionRead

func spannerTypeFromType(typ spansql.Type) (*spannerpb.Type, error) {
	var code spannerpb.TypeCode
	switch typ.Base {
	default:
		return nil, fmt.Errorf("unhandled base type %d", typ.Base)
	case spansql.Bool:
		code = spannerpb.TypeCode_BOOL
	case spansql.Int64:
		code = spannerpb.TypeCode_INT64
	case spansql.Float64:
		code = spannerpb.TypeCode_FLOAT64
	case spansql.String:
		code = spannerpb.TypeCode_STRING
	case spansql.Bytes:
		code = spannerpb.TypeCode_BYTES
	case spansql.Date:
		code = spannerpb.TypeCode_DATE
	}
	st := &spannerpb.Type{Code: code}
	if typ.Array {
		st = &spannerpb.Type{
			Code:             spannerpb.TypeCode_ARRAY,
			ArrayElementType: st,
		}
	}
	return st, nil
}

func spannerValueFromValue(x interface{}) (*structpb.Value, error) {
	switch x := x.(type) {
	default:
		return nil, fmt.Errorf("unhandled database value type %T", x)
	case bool:
		return &structpb.Value{Kind: &structpb.Value_BoolValue{x}}, nil
	case int64:
		// The Spanner int64 is actually a decimal string.
		s := strconv.FormatInt(x, 10)
		return &structpb.Value{Kind: &structpb.Value_StringValue{s}}, nil
	case float64:
		return &structpb.Value{Kind: &structpb.Value_NumberValue{x}}, nil
	case string:
		return &structpb.Value{Kind: &structpb.Value_StringValue{x}}, nil
	case []byte:
		return &structpb.Value{Kind: &structpb.Value_StringValue{base64.StdEncoding.EncodeToString(x)}}, nil
	case nil:
		return &structpb.Value{Kind: &structpb.Value_NullValue{}}, nil
	case []interface{}:
		var vs []*structpb.Value
		for _, elem := range x {
			v, err := spannerValueFromValue(elem)
			if err != nil {
				return nil, err
			}
			vs = append(vs, v)
		}
		return &structpb.Value{Kind: &structpb.Value_ListValue{
			&structpb.ListValue{Values: vs},
		}}, nil
	}
}

func makeKeyRangeList(ranges []*spannerpb.KeyRange) keyRangeList {
	var krl keyRangeList
	for _, r := range ranges {
		krl = append(krl, makeKeyRange(r))
	}
	return krl
}

func makeKeyRange(r *spannerpb.KeyRange) *keyRange {
	var kr keyRange
	switch s := r.StartKeyType.(type) {
	case *spannerpb.KeyRange_StartClosed:
		kr.start = s.StartClosed
		kr.startClosed = true
	case *spannerpb.KeyRange_StartOpen:
		kr.start = s.StartOpen
	}
	switch e := r.EndKeyType.(type) {
	case *spannerpb.KeyRange_EndClosed:
		kr.end = e.EndClosed
		kr.endClosed = true
	case *spannerpb.KeyRange_EndOpen:
		kr.end = e.EndOpen
	}
	return &kr
}
