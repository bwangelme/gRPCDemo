package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"

	"gRPCDemo/pb"

	"context"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS Cert file")
	keyFile    = flag.String("key_file", "", "The TLS Key file")
	jsonDBFile = flag.String("json_db_file", "", "A json file containing a list of features")
	port       = flag.Int("port", 10000, "The Server port")
)

type routeGuideServer struct {
	pb.UnimplementedRouteGuideServer
	savedFeatures []*pb.Feature
	mu            sync.Mutex
	routeNodes    map[string][]*pb.RouteNode
}

func (s *routeGuideServer) GetFeature(ctx context.Context, point *pb.Point) (*pb.Feature, error) {
	for _, feature := range s.savedFeatures {
		if proto.Equal(feature.Location, point) {
			return feature, nil
		}
	}

	return &pb.Feature{Location: point}, nil
}
func (s *routeGuideServer) ListFeatures(rect *pb.Rectangle, stream pb.RouteGuide_ListFeaturesServer) error {
	return status.Errorf(codes.Unimplemented, "method ListFeatures not implemented")
}
func (s *routeGuideServer) RecordRoute(stream pb.RouteGuide_RecordRouteServer) error {
	return status.Errorf(codes.Unimplemented, "method RecordRoute not implemented")
}
func (s *routeGuideServer) RouteChat(stream pb.RouteGuide_RouteChatServer) error {
	return status.Errorf(codes.Unimplemented, "method RouteChat not implemented")
}

func (s *routeGuideServer) loadFeatures(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Read file %v failed", filename)
	}

	if err := json.Unmarshal(data, &s.savedFeatures); err != nil {
		log.Fatalf("Failed to load default features: %v", err)
	}

	log.Printf("Load %d features from json db\n", len(s.savedFeatures))
}

func newServer() *routeGuideServer {
	s := &routeGuideServer{
		routeNodes: make(map[string][]*pb.RouteNode),
	}
	s.loadFeatures(*jsonDBFile)
	return s
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = "/home/xuyundong/Certs/dev.bwangel.abc.pem"
		}

		if *keyFile == "" {
			*keyFile = "/home/xuyundong/Certs/dev.bwangel.abc.key"
		}

		cerds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalln("Failed to generate crendentials", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(cerds)}
	}

	if *jsonDBFile == "" {
		*jsonDBFile = "./testdata/route_guide_db.json"
	}

	server := grpc.NewServer(opts...)
	log.Printf("Listening on the %v\n", *port)
	pb.RegisterRouteGuideServer(server, newServer())
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
