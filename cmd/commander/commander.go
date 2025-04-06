package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"

	pb "commander/pkg/pb/root"

	"github.com/charmbracelet/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CommServer struct {
	pb.UnimplementedCommanderServer
}

func (s *CommServer) Load(source *pb.Source, resp grpc.ServerStreamingServer[pb.FileLoaded]) error {
	log.Info("Command: Load")

	for _, file := range source.GetFiles() {
		name := file.GetName()
		data := file.GetData()

		log.Infof("Loading %v", name)

		err := os.WriteFile(name, data, 0644)
		if err != nil {
			log.Error(err)
			return status.Error(codes.Aborted, err.Error())
		}

		err = resp.Send(&pb.FileLoaded{Name: name})
		if err != nil {
			log.Error(err)
			return err
		}
	}

	log.Info("Done Load")

	return nil
}

func (s *CommServer) Run(_ *pb.Empty, resp grpc.ServerStreamingServer[pb.Output]) error {
	log.Info("Command: Run")

	cmd := exec.Command("python", "main.py")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Error(err)
		return status.Error(codes.Aborted, err.Error())
	}
	output := bufio.NewReader(stdout)

	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	err = cmd.Start()
	if err != nil {
		log.Error(err)
		return status.Error(codes.Aborted, err.Error())
	}

	for {
		line, err := output.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return status.Error(codes.Aborted, err.Error())
		}

		resp.Send(&pb.Output{Data: line})
	}

	err = cmd.Wait()
	if err != nil {
		msg := fmt.Sprintf("%v:\n%v", err.Error(), stderr.String())
		log.Error(msg)
		return status.Error(codes.Aborted, msg)
	}

	log.Info("Done Run")
	return nil
}

func main() {
	listener, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Error(err)
		return
	}

	server := grpc.NewServer()
	pb.RegisterCommanderServer(server, &CommServer{})

	log.Info("Server started on port 5000")

	err = server.Serve(listener)
	if err != nil {
		log.Error(err)
		return
	}
}
