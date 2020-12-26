package main

import (
	"context"
	"fmt"
	"log"

	"github.com/xans-me/grpc-blog-example/protobuff"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("Blog Client")

	// init a dial connection of grpc to service
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("could not dial: %v", err)
	}
	defer conn.Close()

	// initiate of blog service as client
	c := protobuff.NewBlogServiceClient(conn)

	// Create a Blog section
	blog := &protobuff.Blog{
		AuthorId: "xans",
		Title:    "My First Blog",
		Content:  "Content of the first blog",
	}
	resp, err := c.CreateBlog(context.Background(), &protobuff.CreateBlogRequest{Blog: blog})
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	fmt.Printf("Blog has been created successfully : %v", resp)

	// Read Blog section
	respRead, err := c.ReadBlog(context.Background(), &protobuff.ReadBlogRequest{BlogId: "_iud"})
	if err != nil {
		log.Fatalf("error happened while reading : %v", err)
	}
	fmt.Printf("Blog was read : %v", respRead)
}
