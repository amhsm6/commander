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

	log.Info("Success: Load")
	return nil
}

func (s *CommServer) Run(stream grpc.BidiStreamingServer[pb.Interrupt, pb.Output]) error {
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

	go func() {
		_, err := stream.Recv()
		if err == io.EOF {
			return
		}

		if err != nil {
			// TODO: if context cancelled by interrupt or success of command process - do not error
			log.Error(err)
		}

		if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
			log.Warn("Interrupting command...")

			err = cmd.Process.Signal(os.Interrupt)
			if err != nil {
				log.Error(err)
			}

			log.Info("Interrupted successfully")
		} else {
			log.Warn("Command process has already finished")
		}
	}()

	for {
		line, err := output.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return status.Error(codes.Aborted, err.Error())
		}

		log.Info(line[:len(line)-1])

		err = stream.Send(&pb.Output{Data: line})
		if err != nil {
			log.Error(err)
			return status.Error(codes.Aborted, err.Error())
		}
	}

	err = cmd.Wait()
	if err != nil {
		msg := fmt.Sprintf("%v:\n%v", err.Error(), stderr.String())

		log.Error(msg)
		return status.Error(codes.Aborted, msg)
	}

	log.Info("Success: Run")
	return nil
}

func main() {
	listener, err := net.Listen("tcp4", "0.0.0.0:5000")
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	server := grpc.NewServer()
	pb.RegisterCommanderServer(server, &CommServer{})

	log.Info("Server started on port 5000")

	err = server.Serve(listener)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
}
