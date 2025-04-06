package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	pb "commander/pkg/pb/root"

	"github.com/charmbracelet/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "localhost:5000", "Adress of Commander")
	flag.Parse()

	verb := flag.Arg(0)
	if verb != "load" && verb != "run" {
		log.Error("Invalid usage")
		return
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error(err)
		return
	}
	defer conn.Close()

	client := pb.NewCommanderClient(conn)

	switch verb {
	case "load":
		srcs := flag.Args()[1:]

		files := []*pb.File{}
		for _, path := range srcs {
			name := filepath.Base(path)
			data, err := os.ReadFile(path)
			if err != nil {
				log.Error(err)
				return
			}

			files = append(files, &pb.File{Name: name, Data: data})
		}

		resp, err := client.Load(context.Background(), &pb.Source{Files: files})
		if err != nil {
			log.Error(err)
			return
		}

		for {
			status, err := resp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
				return
			}

			log.Infof("Loaded: %v", status.GetName())
		}

		log.Info("Done loading")

	case "run":
		resp, err := client.Run(context.Background(), &pb.Empty{})
		if err != nil {
			log.Error(err)
			return
		}

		for {
			output, err := resp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
				return
			}

			fmt.Print(output.GetData())
		}

		log.Info("Done running")
	}
}
