package main

import (
	"fmt"
	"io"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	pb "github.com/docker/docker-e2e/control_server/controlserver"
)

type controllerServer struct { // nothing. it's just empty. who cares

}

func (s *controllerServer) ExchangeMessages(stream pb.Controller_ExchangeMessagesServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Println("heartbeat: ", in.Id)
		cmd := pb.Command{Action: "ack"}
		if err := stream.Send(&cmd); err != nil {
			return err
		}
	}
}

func newServer() *controllerServer {
	s := new(controllerServer)
	return s
}

func main() {
	lis, err := net.Listen("tcp", ":9001")
	if err != nil {
		grpclog.Fatalf("failed to listed: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterControllerServer(grpcServer, newServer())
	grpcServer.Serve(lis)
}
