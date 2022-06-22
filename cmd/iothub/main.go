package main

import (
	"context"
	"flag"
	Iothub_v1 "github.com/tkeel-io/iothub/api/iothub/v1"
	"os"
	"os/signal"
	"syscall"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/tkeel-io/iothub/pkg/server"
	"github.com/tkeel-io/iothub/pkg/service"
	pb "github.com/tkeel-io/iothub/protobuf"
	"github.com/tkeel-io/kit/app"
	"github.com/tkeel-io/kit/log"
	"github.com/tkeel-io/kit/transport"

	openapi "github.com/tkeel-io/iothub/api/openapi/v1"
)

var (
	// Name app.
	Name string
	// HTTPAddr string.
	HTTPAddr string
	// GRPCAddr string.
	GRPCAddr string
	//
)

func init() {
	flag.StringVar(&Name, "name", "iothub", "app name.")
	flag.StringVar(&HTTPAddr, "http_addr", ":8080", "http listen address.")
	flag.StringVar(&GRPCAddr, "grpc_addr", ":9000", "grpc listen address.")
}

func main() {
	flag.Parse()
	httpSrv := server.NewHTTPServer(HTTPAddr)
	grpcSrv := server.NewGRPCServer(GRPCAddr)
	serverList := []transport.Server{httpSrv, grpcSrv}

	app := app.New(Name,
		&log.Conf{
			App:   Name,
			Level: "debug",
			Dev:   true,
		},
		serverList...,
	)
	{ // User service
		OpenapiSrv := service.NewOpenapiService()
		openapi.RegisterOpenapiHTTPServer(httpSrv.Container, OpenapiSrv)
		openapi.RegisterOpenapiServer(grpcSrv.GetServe(), OpenapiSrv)

		client, err := dapr.NewClient()
		if nil != err {
			panic(err)
		}

		// topic service
		HookServiceSrv := service.NewHookService(client)
		pb.RegisterHookProviderServer(grpcSrv.GetServe(), HookServiceSrv)

		TopicSrv, err := service.NewTopicService(context.Background(), HookServiceSrv)
		if nil != err {
			log.Fatal(err)
		}
		Iothub_v1.RegisterTopicHTTPServer(httpSrv.Container, TopicSrv)
		Iothub_v1.RegisterTopicServer(grpcSrv.GetServe(), TopicSrv)

		//
        // metrics service.
        metricsSrv := service.NewMetricsService()
        Iothub_v1.RegisterMetricsHTTPServer(httpSrv.Container, metricsSrv)
	}

	if err := app.Run(context.TODO()); err != nil {
		panic(err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)
	<-stop

	if err := app.Stop(context.TODO()); err != nil {
		panic(err)
	}
}
