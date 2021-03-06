package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/xans-me/grpc-blog-example/protobuff"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive" // for BSON ObjectID
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var collection *mongo.Collection

type server struct{}

func (s server) ListBlog(req *protobuff.ListBlogRequest, stream protobuff.BlogService_ListBlogServer) error {
	fmt.Println("List blog request")

	cur, err := collection.Find(context.Background(), nil)
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("unknown internal error : %v", err),
		)
	}
	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		data := &blogItem{}
		err := cur.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB : %v", err),
			)
		}
		stream.Send(&protobuff.ListBlogResponse{Blog: dataToBlogPb(data)})
	}
	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}

	return nil
}

func (s server) DeleteBlog(ctx context.Context, request *protobuff.DeleteBlogRequest) (*protobuff.DeleteBlogResponse, error) {
	fmt.Println("Delete Blog Request")
	oid, err := primitive.ObjectIDFromHex(request.GetBlogId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("cannot parse ID"))
	}
	filter := bson.M{"_id": oid}

	res, err := collection.DeleteOne(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot delete object in mongoDB : %v", err)
	}

	if res.DeletedCount == 0 {
		return nil, status.Errorf(codes.NotFound, "cannot find blog in mongoDB : %v", err)
	}

	return &protobuff.DeleteBlogResponse{BlogId: request.GetBlogId()}, nil
}

func (s server) UpdateBlog(ctx context.Context, request *protobuff.UpdateBlogRequest) (*protobuff.UpdateBlogResponse, error) {
	fmt.Println("Update Blog Request")
	blog := request.GetBlog()

	oid, err := primitive.ObjectIDFromHex(blog.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("cannot parse ID"))
	}

	// create empty struct
	data := &blogItem{}
	filter := bson.M{"_id": oid}
	err = collection.FindOne(ctx, filter).Decode(&data)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Blog %s not found", oid)
	}

	// update our internal struct
	data.AuthorId = blog.GetAuthorId()
	data.Content = blog.GetContent()
	data.Title = blog.GetTitle()

	_, err = collection.ReplaceOne(ctx, filter, data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot update object in mongoDB : %v", err)
	}

	return &protobuff.UpdateBlogResponse{Blog: dataToBlogPb(data)}, nil
}

func dataToBlogPb(data *blogItem) *protobuff.Blog {
	return &protobuff.Blog{
		Id:       data.ID.Hex(),
		AuthorId: data.AuthorId,
		Title:    data.Title,
		Content:  data.Content,
	}
}

func (s server) ReadBlog(ctx context.Context, request *protobuff.ReadBlogRequest) (*protobuff.ReadBlogResponse, error) {
	fmt.Println("Read blog request")
	blogId := request.GetBlogId()

	data := &blogItem{}
	filter := bson.M{"_id": blogId}
	err := collection.FindOne(ctx, filter).Decode(&data)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Blog %s not found", blogId)
	}

	return &protobuff.ReadBlogResponse{Blog: dataToBlogPb(data)}, nil
}

func (s server) CreateBlog(ctx context.Context, request *protobuff.CreateBlogRequest) (*protobuff.CreateBlogResponse, error) {
	blog := request.GetBlog()

	data := blogItem{
		AuthorId: blog.GetAuthorId(),
		Content:  blog.GetContent(),
		Title:    blog.GetTitle(),
	}

	res, err := collection.InsertOne(ctx, data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("internal error: %v", err))
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("cannot convert to OID : %v", err))
	}

	return &protobuff.CreateBlogResponse{Blog: &protobuff.Blog{
		Id:       oid.Hex(),
		AuthorId: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}}, nil
}

type blogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorId string             `bson:"author_id"`
	Content  string             `bson:"content"`
	Title    string             `bson:"title"`
}

func main() {
	// set the log level
	// if crash the go code, we get file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// connect to mongodb
	fmt.Println("Connecting to MongoDB...")
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database("mydb").Collection("blog")

	fmt.Println("Blog Service Started")
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	s := grpc.NewServer(opts...)
	protobuff.RegisterBlogServiceServer(s, &server{})

	reflection.Register(s)

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
	fmt.Println("Closing the MongoDB connection...")
	client.Disconnect(context.TODO())
	fmt.Println("End of Program...")
}
