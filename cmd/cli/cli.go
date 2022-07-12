// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"fmt"
	"gRPCDemo/pb"
	"io"
	"log"
	"math/rand"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	tls                = flag.Bool("tls", false, "Connection use TLS")
	caFile             = flag.String("ca_file", "", "the file containing the ca root cert file")
	serverAddr         = flag.String("server_addr", "localhost:10000", "Server Address")
	serverHostOverride = flag.String("server_host_override", "dev.bwangel.abc", "The server name used to verify the hostname returned by the TLS handshake")
)

func printFeature(client pb.RouteGuideClient, point *pb.Point) {
	log.Printf("Getting feature for point(%d, %d)", point.Latitude, point.Longitude)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	feature, err := client.GetFeature(ctx, point)
	if err != nil {
		log.Fatalf("%v.GetFeatures(_) = _, %v: ", client, err)
	}
	log.Println(feature)
}

func printFeatures(client pb.RouteGuideClient, rect *pb.Rectangle) {
	log.Printf("Looking for features within %v", rect)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := client.ListFeatures(ctx, rect)
	if err != nil {
		log.Fatalf("%v.ListFeatures(_) = _, %v", client, err)
	}

	for {
		feature, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.ListFeatures(_) = _, %v", client, err)
		}
		log.Printf("Feature: name: %q, point: (%v, %v)\n",
			feature.GetName(),
			feature.GetLocation().GetLatitude(),
			feature.GetLocation().GetLongitude(),
		)
	}
}

func randomPoint(r *rand.Rand) *pb.Point {
	lat := (r.Int31n(180) - 90) * 1e7
	long := (r.Int31n(360) - 180) * 1e7
	return &pb.Point{Latitude: lat, Longitude: long}
}

func runRecordRoute(client pb.RouteGuideClient) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pointCount := int(r.Int31n(100)) + 2
	var points []*pb.Point
	for i := 0; i < pointCount; i++ {
		points = append(points, randomPoint(r))
	}
	points = append(points, &pb.Point{
		Latitude:  408122808,
		Longitude: -743999179,
	})
	log.Printf("Traversing %d points", len(points))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.RecordRoute(ctx)
	if err != nil {
		log.Fatalf("%v.RecordRoute(_) = _, %v", client, err)
	}
	for _, point := range points {
		if err := stream.Send(point); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, point, err)
		}
	}
	reply, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
	}
	log.Printf("Route summary: %v", reply)
	log.Printf("Route summary time: %v Milliseconds", reply.ElapsedTime)
}

func runRouteChat(client pb.RouteGuideClient) {
	notes := []*pb.RouteNode{
		{Location: &pb.Point{Latitude: 0, Longitude: 1}, Message: "1st message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 2}, Message: "2nd message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 3}, Message: "3rd message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 1}, Message: "4th message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 2}, Message: "5th message"},
		{Location: &pb.Point{Latitude: 0, Longitude: 3}, Message: "6th message"},
	}
	ctx, cancal := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancal()

	stream, err := client.RouteChat(ctx)
	if err != nil {
		log.Fatalf("%v.RouteChat(_) = _, %v", client, err)
	}
	watic := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				close(watic)
				return
			}
			if err != nil {
				log.Fatalf("failed to receive a note: %v", err)
			}
			log.Printf("Got message %s at point(%d,%d)", in.Message, in.Location.Latitude, in.Location.Longitude)
		}
	}()

	for _, note := range notes {
		if err := stream.Send(note); err != nil {
			log.Fatalf("Failed to send note %v", err)
		}
	}
	stream.CloseSend()
	<-watic
}

func conversations(client pb.EchoClient) {
	stream, err := client.Conversations(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		err := stream.Send(&pb.StreamRequest{
			Question: fmt.Sprintf("Stream client rpc %d", i),
		})
		if err != nil {
			log.Fatalln(err)
		}
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalln(err)
		}

		log.Println(res.Answer)
	}

	err = stream.CloseSend()
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = "/home/xuyundong/Certs/cacert.pem"
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLC credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	opts = append(opts, grpc.WithBlock())
	log.Printf("Start to dial with %v\n", *serverAddr)
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	//client := pb.NewRouteGuideClient(conn)
	//printFeature(client, &pb.Point{Latitude: 407838351, Longitude: -746143763})
	//// Looking for features missing
	//printFeature(client, &pb.Point{Latitude: 1, Longitude: 1})
	//
	//printFeatures(client, &pb.Rectangle{
	//	Lo: &pb.Point{Latitude: 400000000, Longitude: -750000000},
	//	Hi: &pb.Point{Latitude: 420000000, Longitude: -730000000},
	//})

	//runRecordRoute(client)
	//runRouteChat(client)

	echoClient := pb.NewEchoClient(conn)
	conversations(echoClient)
}
