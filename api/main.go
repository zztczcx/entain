package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"git.neds.sh/matty/entain/api/proto/racing"
	"git.neds.sh/matty/entain/api/proto/sports"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

var (
	apiEndpoint        = flag.String("api-endpoint", "localhost:8000", "API endpoint")
	racingGrpcEndpoint = flag.String("racing-grpc-endpoint", "localhost:9000", "Racing gRPC server endpoint")
	sportsGrpcEndpoint = flag.String("sports-grpc-endpoint", "localhost:9001", "Sports gRPC server endpoint")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Printf("failed running api server: %s\n", err)
	}
}

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()

	// Register racing service
	if err := racing.RegisterRacingHandlerFromEndpoint(
		ctx,
		mux,
		*racingGrpcEndpoint,
		[]grpc.DialOption{grpc.WithInsecure()},
	); err != nil {
		return err
	}

	// Register sports service
	if err := sports.RegisterSportsHandlerFromEndpoint(
		ctx,
		mux,
		*sportsGrpcEndpoint,
		[]grpc.DialOption{grpc.WithInsecure()},
	); err != nil {
		return err
	}

	log.Printf("API server listening on: %s\n", *apiEndpoint)

	return http.ListenAndServe(*apiEndpoint, mux)
}
