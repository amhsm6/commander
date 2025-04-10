package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"

	pb "commander/pkg/pb/root"

	"github.com/charmbracelet/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := flag.String("addr", "candlebot:5000", "Adress of Commander")
	flag.Parse()

	verb := flag.Arg(0)
	if verb != "load" && verb != "run" {
		log.Error("Invalid usage")
		os.Exit(1)
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error(err)
		os.Exit(1)
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
                os.Exit(1)
			}

			files = append(files, &pb.File{Name: name, Data: data})
		}

		resp, err := client.Load(context.Background(), &pb.Source{Files: files})
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}

		for {
			status, err := resp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Error(err)
                os.Exit(1)
			}

			log.Infof("Loaded: %v", status.GetName())
		}

		log.Info("Done loading")

	case "run":
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()

		resp, err := client.Run(ctx)
		if err != nil {
            cancel()

			log.Error(err)
			os.Exit(1)
		}

        sigint := make(chan os.Signal, 15)
        signal.Notify(sigint, os.Interrupt)

        go func() {
            select {
            case <-sigint:
                err := resp.Send(&pb.Interrupt{})
                if err != nil {
                    cancel()

                    log.Error(err)
                    os.Exit(1)
                }
            
            case <-ctx.Done():
            }
        }()

		for {
			output, err := resp.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
                cancel()

				log.Error(err)
				os.Exit(1)
			}

			fmt.Print(output.GetData())
		}

		log.Info("Done running")
	}
}
