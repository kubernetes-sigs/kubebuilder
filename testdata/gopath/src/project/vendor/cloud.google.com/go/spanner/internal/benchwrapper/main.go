// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main wraps the client library in a gRPC interface that a benchmarker
// can communicate through.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"cloud.google.com/go/spanner"
	pb "cloud.google.com/go/spanner/internal/benchwrapper/proto"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc"
)

var port = flag.String("port", "", "specify a port to run on")

func main() {
	flag.Parse()
	if *port == "" {
		log.Fatalf("usage: %s --port=8081", os.Args[0])
	}

	if os.Getenv("SPANNER_EMULATOR_HOST") == "" {
		log.Fatal("This benchmarking server only works when connected to an emulator. Please set SPANNER_EMULATOR_HOST.")
	}

	ctx := context.Background()
	c, err := spanner.NewClient(ctx, "projects/someproject/instances/someinstance/databases/somedatabase")
	if err != nil {
		log.Fatal(err)
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterSpannerBenchWrapperServer(s, &server{
		c: c,
	})
	log.Printf("Running on localhost:%s\n", *port)
	log.Fatal(s.Serve(lis))
}

type server struct {
	c *spanner.Client
}

func (s *server) Read(ctx context.Context, req *pb.ReadQuery) (*pb.EmptyResponse, error) {
	it := s.c.ReadOnlyTransaction().Query(context.Background(), spanner.Statement{SQL: req.Query})
	for {
		_, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		// Do nothing with the data.
	}
	return &pb.EmptyResponse{}, nil
}

func (s *server) Insert(ctx context.Context, req *pb.InsertQuery) (*pb.EmptyResponse, error) {
	var muts []*spanner.Mutation
	for _, i := range req.Users {
		muts = append(muts, spanner.Insert("sometable", []string{"name", "age"}, []interface{}{i.Name, i.Age}))
	}
	if _, err := s.c.Apply(context.Background(), muts); err != nil {
		log.Fatal(err)
	}
	// Do nothing with the data.
	return &pb.EmptyResponse{}, nil
}

func (s *server) Update(ctx context.Context, req *pb.UpdateQuery) (*pb.EmptyResponse, error) {
	var stmts []spanner.Statement
	for _, q := range req.Queries {
		stmts = append(stmts, spanner.Statement{SQL: q})
	}
	if _, err := s.c.ReadWriteTransaction(context.Background(), func(ctx2 context.Context, tx *spanner.ReadWriteTransaction) error {
		_, err := tx.BatchUpdate(ctx2, stmts)
		return err
	}); err != nil {
		log.Fatal(err)
	}
	// Do nothing with the data.
	return &pb.EmptyResponse{}, nil
}
