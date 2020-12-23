package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/xans-me/grpc-blog-example/protobuff"
	"google.golang.org/grpc"
)

type server struct{}

func main() {
	// set the log level
	// if crash the go code, we get file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("Blog Service Started")

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// tls := false
	// if tls {
	// 	certFile := "ssl/server.cert"
	// 	keyFile := "ssl/server.pem"
	// 	creds, sslErr := credentials.NewServerTLSFromFile(certFile, keyFile)
	// 	if sslErr != nil {
	// 		log.Fatalf("failed loading certificates: %v", sslErr)
	// 		return
	// 	}
	// 	opts = append(opts, grpc.Creds(creds))
	// }

	var opts []grpc.ServerOption
	s := grpc.NewServer(opts...)
	protobuff.RegisterBlogServiceServer(s, &server{})

	go func() {
		fmt.Println("Starting server...")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve : %v", err)
		}
	}()

	// wait for Control C to exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// block until a signal is received
	<-ch
	fmt.Println("Stopping the server...")
	s.Stop()
	fmt.Println("Closing the listener...")
	lis.Close()
	fmt.Println("End of Program...")
}
