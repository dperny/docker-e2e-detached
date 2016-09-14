package main

import (
	"context"
	"fmt"
	pb "github.com/docker/docker-e2e/control_server/controlserver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"io"
	"time"
)

func runExchangeMessages(client pb.ControllerClient) {
	stream, err := client.ExchangeMessages(context.Background())
	if err != nil {
		grpclog.Fatalf("%v.ExchangeMessages(_) = _, %v", client, err)
	}

	t := time.NewTicker(time.Second)
	defer t.Stop()
	for ; ; <-t.C {
		h := pb.Heartbeat{Id: "1"}
		if err := stream.Send(&h); err != nil {
			grpclog.Fatalf("Failed to send heartbeat: %v", err)
		}
		in, err := stream.Recv()
		if err == io.EOF {
			return
		}
		if err != nil {
			grpclog.Fatalf("Failed to recieve command: %v", err)
		}
		fmt.Println("action: ", in.Action)
	}
}

func main() {
	conn, err := grpc.Dial("127.0.0.1:9001", grpc.WithInsecure())
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewControllerClient(conn)
	runExchangeMessages(client)
}
