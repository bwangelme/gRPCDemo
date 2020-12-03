// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"gRPCDemo/pb"
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
}
