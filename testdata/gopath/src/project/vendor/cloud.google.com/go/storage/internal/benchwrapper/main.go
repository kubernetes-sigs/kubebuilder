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

// Package main wraps the client library in a gRPC interface that a benchmarker can communicate through.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"

	"cloud.google.com/go/storage"
	pb "cloud.google.com/go/storage/internal/benchwrapper/proto"
	"google.golang.org/grpc"
)

var port = flag.String("port", "", "specify a port to run on")

// minRead respresents the number of bytes to read at a time.
const minRead = 1024 * 1024

func main() {
	flag.Parse()
	if *port == "" {
		log.Fatalf("usage: %s --port=8081", os.Args[0])
	}

	if os.Getenv("STORAGE_EMULATOR_HOST") == "" {
		log.Fatal("This benchmarking server only works when connected to an emulator. Please set STORAGE_EMULATOR_HOST.")
	}

	ctx := context.Background()
	c, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterStorageBenchWrapperServer(s, &server{
		c: c,
	})
	log.Printf("Running on localhost:%s\n", *port)
	log.Fatal(s.Serve(lis))
}

type server struct {
	c *storage.Client
}

func (s *server) Read(ctx context.Context, in *pb.ObjectRead) (*pb.EmptyResponse, error) {
	b := s.c.Bucket(in.GetBucketName())
	o := b.Object(in.GetObjectName())
	r, err := o.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	for int(r.Remain()) > 0 {
		ba := make([]byte, minRead)
		_, err := r.Read(ba)
		if err == io.EOF {
			return &pb.EmptyResponse{}, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return &pb.EmptyResponse{}, nil
}

func (s *server) Write(ctx context.Context, in *pb.ObjectWrite) (*pb.EmptyResponse, error) {
	b := s.c.Bucket(in.GetBucketName())
	o := b.Object(in.GetObjectName())
	w := o.NewWriter(ctx)
	content, err := ioutil.ReadFile(in.GetDestination())
	if err != nil {
		return nil, err
	}
	w.ContentType = "text/plain"
	if _, err := w.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		if err == io.EOF {
			return &pb.EmptyResponse{}, nil
		}
		return nil, err
	}
	return &pb.EmptyResponse{}, nil
}
