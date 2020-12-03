// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"gRPCDemo/pb"
	"io"
	"log"
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
	ctx, cancel := context.WithTimeout( context.Background(), 10*time.Second)
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
	client := pb.NewRouteGuideClient(conn)

	printFeature(client, &pb.Point{Latitude: 407838351, Longitude: -746143763})
	// Looing for features missing
	printFeature(client, &pb.Point{Latitude: 1, Longitude: 1})

	printFeatures(client, &pb.Rectangle{
		Lo: &pb.Point{Latitude: 400000000, Longitude: -750000000},
		Hi: &pb.Point{Latitude: 420000000, Longitude: -730000000},
	})
}
