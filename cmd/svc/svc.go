package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"sync"
	"time"

	"gRPCDemo/pb"

	"context"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS Cert file")
	keyFile    = flag.String("key_file", "", "The TLS Key file")
	jsonDBFile = flag.String("json_db_file", "", "A json file containing a list of features")
	port       = flag.Int("port", 10000, "The Server port")
)

type echoServer struct {
	pb.UnimplementedEchoServer
}

func (s *echoServer) Conversations(stream pb.Echo_ConversationsServer) error {
	n := 1
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = stream.Send(&pb.StreamResponse{
			Answer: fmt.Sprintf("Answer: %d, Question: %s", n, req.Question),
		})
		if err != nil {
			return err
		}
		n++
		log.Printf("from stream client question: %s", req.Question)
	}

}

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
	for _, feature := range s.savedFeatures {
		if inRange(feature.Location, rect) {
			if err := stream.Send(feature); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *routeGuideServer) RecordRoute(stream pb.RouteGuide_RecordRouteServer) error {
	var pointCount, featureCount, distance int32
	var lastPoint *pb.Point

	startTime := time.Now()
	for {
		point, err := stream.Recv()
		if err == io.EOF {
			endTime := time.Now()
			return stream.SendAndClose(&pb.RouteSummary{
				PointCount:   pointCount,
				FeatureCount: featureCount,
				Distance:     distance,
				ElapsedTime:  int32(endTime.Sub(startTime).Milliseconds()),
			})
		}
		if err != nil {
			return err
		}
		pointCount++
		for _, feature := range s.savedFeatures {
			if proto.Equal(feature.Location, point) {
				featureCount++
			}
		}
		if lastPoint != nil {
			distance += calcDistance(lastPoint, point)
		}
		time.Sleep(time.Millisecond * 10)
		lastPoint = point
	}
}

// RouteChat
// echo 服务，将客户端输入的节点返回回去
func (s *routeGuideServer) RouteChat(stream pb.RouteGuide_RouteChatServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		key := serialize(in.Location)

		s.mu.Lock()
		s.routeNodes[key] = append(s.routeNodes[key], in)
		// 这里复制一遍是为了防止当服务端写一个客户端的流时，另一个客户端修改了 routeNodes
		// 这里不需要执行深拷贝的原因是，routeNodes 只会增加，不会修改
		rn := make([]*pb.RouteNode, len(s.routeNodes[key]))
		copy(rn, s.routeNodes[key])
		s.mu.Unlock()

		for _, note := range rn {
			if err := stream.Send(note); err != nil {
				return err
			}
		}
	}
}

func toRadians(num float64) float64 {
	return num * math.Pi / float64(180)
}

// calcDistance 计算两个节点之间的距离
func calcDistance(p1 *pb.Point, p2 *pb.Point) int32 {
	const CordFactor float64 = 1e7
	const R = float64(6371000) // 地球半径

	lat1 := toRadians(float64(p1.Latitude) / CordFactor)
	lat2 := toRadians(float64(p2.Latitude) / CordFactor)
	lng1 := toRadians(float64(p1.Longitude) / CordFactor)
	lng2 := toRadians(float64(p2.Longitude) / CordFactor)
	dlat := lat2 - lat1
	dlng := lng2 - lng1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c
	return int32(distance)
}

// inRange 判断 point 是否在 rect 所划定的范围内
func inRange(point *pb.Point, rect *pb.Rectangle) bool {
	left := math.Min(float64(rect.Lo.Longitude), float64(rect.Hi.Longitude))
	right := math.Max(float64(rect.Lo.Longitude), float64(rect.Hi.Longitude))
	top := math.Max(float64(rect.Lo.Latitude), float64(rect.Hi.Latitude))
	bottom := math.Min(float64(rect.Lo.Latitude), float64(rect.Hi.Latitude))

	if float64(point.Longitude) >= left &&
		float64(point.Longitude) <= right &&
		float64(point.Latitude) <= top &&
		float64(point.Latitude) >= bottom {
		return true
	}
	return true
}

func serialize(point *pb.Point) string {
	return fmt.Sprintf("%d %d", point.Latitude, point.Longitude)
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
	pb.RegisterEchoServer(server, &echoServer{})
	if err := server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
