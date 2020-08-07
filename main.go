package main

import (
	"context"
	"flag"
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/opentracing/opentracing-go"
	zipkinot "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/tmc/grpc-websocket-proxy/wsproxy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	test "grpc-test/proto"
	"grpc-test/services"
	//"log"
	"grpc-test/log"
	"net"
	"net/http"
)

var (
	grpcServerEndpoint = flag.String("grpc-server-endpoint", "localhost:8088", "gRPC server endpoint")
)

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := test.RegisterGreeterHandlerFromEndpoint(ctx, mux, *grpcServerEndpoint, opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(":8081", wsproxy.WebsocketProxy(mux))
}
func main() {
	port := flag.String("p", "8088", "port")
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port)) //监听所有网卡8088端口的TCP连接
	if err != nil {
		log.Fatal("监听失败", err)
		return
	}

	// set up a span reporter
	reporter := zipkinhttp.NewReporter("http://xxxxxxxxxxx:9411/api/v2/spans")
	defer reporter.Close()

	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("grpc-server", "127.0.0.1:8088")
	if err != nil {
		log.Fatal("unable to create local endpoint", err)
		return
	}
	// set-up our sampling strategy
	//sampler, err := zipkin.NewBoundarySampler(0.01, time.Now().UnixNano())
	//if err != nil {
	//	log.Fatalf("unable to create sampler: %+v\n", err)
	//}
	// initialize our tracer
	nativeTracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint), zipkin.WithTraceID128Bit(true))
	if err != nil {
		log.Fatal("unable to create tracer", err)
		return
	}

	// use zipkin-go-opentracing to wrap our tracer
	tracer := zipkinot.Wrap(nativeTracer)

	// optionally set as Global OpenTracing tracer instance
	opentracing.SetGlobalTracer(tracer)

	s := grpc.NewServer(grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
		grpc_ctxtags.StreamServerInterceptor(),
		grpc_opentracing.StreamServerInterceptor(),
		grpc_zap.StreamServerInterceptor(log.ZapInterceptor()),
		//otgrpc.OpenTracingStreamServerInterceptor(tracer, otgrpc.LogPayloads()),
		grpc_auth.StreamServerInterceptor(myAuthFunction),
		grpc_recovery.StreamServerInterceptor(),
	)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_opentracing.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(log.ZapInterceptor()),
			//otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads()),
			grpc_auth.UnaryServerInterceptor(myAuthFunction),
			grpc_recovery.UnaryServerInterceptor(),
		))) //创建gRPC服务

	defer log.FlushLogger()
	/**注册接口服务
	 * 以定义proto时的service为单位注册，服务中可以有多个方法
	 * (proto编译时会为每个service生成Register***Server方法)
	 * 包.注册服务方法(gRpc服务实例，包含接口方法的结构体[指针])
	 */
	test.RegisterGreeterServer(s, &services.Server{})
	/**如果有可以注册多个接口服务,结构体要实现对应的接口方法
	 * user.RegisterLoginServer(s, &server{})
	 * minMovie.RegisterFbiServer(s, &server{})
	 */
	// 在gRPC服务器上注册反射服务
	reflection.Register(s)

	go run()
	// 将监听交给gRPC服务处理
	err = s.Serve(lis)
	if err != nil {
		log.Fatal("failed to serve", err)
		return
	}
}

func myAuthFunction(ctx context.Context) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "bearer")

	if err != nil {
		return nil, err
	}
	fmt.Println(token)
	//tokenInfo, err := parseToken(token)
	//
	//if err != nil {
	//
	//	return nil, grpc.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	//
	//}
	//
	//grpc_ctxtags.Extract(ctx).Set("auth.sub", userClaimFromToken(tokenInfo))
	//
	//newCtx := context.WithValue(ctx, "tokenInfo", tokenInfo)

	return ctx, nil
}
