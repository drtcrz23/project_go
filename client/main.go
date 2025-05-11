package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/drtcrz23/project_go/proto"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewPubSubClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{Key: "test"})
	if err != nil {
		log.Fatalf("Subscribe failed: %v", err)
	}

	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				log.Println("Stream closed")
				return
			}
			if err != nil {
				log.Printf("Stream error: %v", err)
				return
			}
			log.Printf("Received event: %s", event.Data)
		}
	}()

	time.Sleep(1 * time.Second)
	for i := 1; i <= 3; i++ {
		_, err := client.Publish(ctx, &pb.PublishRequest{
			Key:  "test",
			Data: fmt.Sprintf("Сообщение %d", i),
		})
		if err != nil {
			log.Printf("Publish failed: %v", err)
			continue
		}
		log.Printf("Published: Сообщение %d", i)
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
}
