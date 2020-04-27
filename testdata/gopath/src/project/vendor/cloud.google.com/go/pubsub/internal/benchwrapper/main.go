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
	"strings"

	"cloud.google.com/go/pubsub"
	pb "cloud.google.com/go/pubsub/internal/benchwrapper/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var port = flag.String("port", "", "specify a port to run on")

func main() {
	flag.Parse()
	if *port == "" {
		log.Fatalf("usage: %s --port=8081", os.Args[0])
	}

	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		log.Fatal("This benchmarking server only works when connected to an emulator. Please set PUBSUB_EMULATOR_HOST.")
	}

	ctx := context.Background()
	c, err := pubsub.NewClient(ctx, "someproject")
	if err != nil {
		log.Fatal(err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterPubsubBenchWrapperServer(s, &server{
		c: c,
	})
	log.Printf("Running on localhost:%s\n", *port)
	log.Fatal(s.Serve(lis))
}

type server struct {
	c *pubsub.Client
}

func (s *server) Recv(ctx context.Context, req *pb.PubsubRecv) (*pb.EmptyResponse, error) {
	sub := s.c.Subscription(req.SubName)
	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
	})

	if err != nil {
		s, _ := status.FromError(err)
		// Return success on server initiated EOF, which is expected.
		if strings.Contains(s.Message(), "EOF") {
			return &pb.EmptyResponse{}, nil
		}
		return nil, err
	}
	return &pb.EmptyResponse{}, nil
}
