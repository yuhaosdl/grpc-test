package main

import (
	"context"
	"fmt"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"github.com/opentracing/opentracing-go"
	zipkinot "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"google.golang.org/grpc"
	test "grpc-test/proto"
	"io"
	"log"
	"time"
)

type customCredential struct{}

func (c customCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"appid":         "101010",
		"authorization": "bearer 12345676",
	}, nil
}

func (c customCredential) RequireTransportSecurity() bool {
	return false
}
func main() {
	// set up a span reporter
	reporter := zipkinhttp.NewReporter("http://39.106.8.113:9411/api/v2/spans")
	defer reporter.Close()

	// create our local service endpoint
	endpoint, err := zipkin.NewEndpoint("grpc-client", "127.0.0.1:9000")
	if err != nil {
		log.Fatalf("unable to create local endpoint: %+v\n", err)
	}
	// set-up our sampling strategy
	//sampler, err := zipkin.NewBoundarySampler(0.01, time.Now().UnixNano())
	//if err != nil {
	//	log.Fatalf("unable to create sampler: %+v\n", err)
	//}
	// initialize our tracer
	nativeTracer, err := zipkin.NewTracer(reporter, zipkin.WithLocalEndpoint(endpoint), zipkin.WithTraceID128Bit(true), zipkin.WithSharedSpans(false))
	if err != nil {
		log.Fatalf("unable to create tracer: %+v\n", err)
	}

	// use zipkin-go-opentracing to wrap our tracer
	tracer := zipkinot.Wrap(nativeTracer)

	// optionally set as Global OpenTracing tracer instance
	opentracing.SetGlobalTracer(tracer)

	// 建立连接到gRPC服务
	//39.106.8.113
	conn, err := grpc.Dial("localhost:8088", grpc.WithInsecure(), grpc.WithPerRPCCredentials(&customCredential{}), grpc.WithPerRPCCredentials(&customCredential{}),
		grpc.WithUnaryInterceptor(grpc_opentracing.UnaryClientInterceptor()), //otgrpc.OpenTracingClientInterceptor(tracer)
		grpc.WithStreamInterceptor(
			grpc_opentracing.StreamClientInterceptor()))

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// 函数结束时关闭连接
	defer conn.Close()

	// 创建Waiter服务的客户端
	t := test.NewGreeterClient(conn)

	// 模拟请求数据
	res := "test123"
	//// os.Args[1] 为用户执行输入的参数 如：go run ***.go 123
	//if len(os.Args) > 1 {
	//	res = os.Args[1]
	//}
	//md := metadata.Pairs("authorization", "bearer 12345676")
	//ctx := metadata.NewOutgoingContext(context.Background(), md)
	ctx := context.Background()
	// 调用gRPC接口
	tr, err := t.SayHello(ctx, &test.HelloReq{Name: res})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("服务端响应:%s", tr.Message)
	fmt.Println("-----------------------------------------------")
	ctx, _ = context.WithCancel(context.Background())
	stream, err := t.AlwaysSay(ctx)
	if err := stream.Send(&test.StreamReq{Name: "小明"}); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := stream.Send(&test.StreamReq{Name: "小张"}); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := stream.Send(&test.StreamReq{Name: "小红"}); err != nil {
		fmt.Println(err.Error())
		return
	}
	//cancel()
	go func() {
		for {
			// 接收从 服务端返回的数据流
			resp, err := stream.Recv()
			if err == io.EOF {
				log.Println("⚠️ 收到服务端的结束信号")
				break //如果收到结束信号，则退出“接收循环”，结束客户端程序
			}

			if err != nil {
				// TODO: 处理接收错误
				log.Println("接收数据出错:", err)
			}

			// 没有错误的情况下，打印来自服务端的消息
			log.Printf("[客户端收到]: %s", resp.Message)
		}
	}()
	stream.CloseSend()
	time.Sleep(time.Millisecond * 20)

	//cancel()

	fmt.Println("-----------------------------------------------")
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	stream2, err := t.SaySomething(ctx2)
	stream2.Send(&test.SaySomethingReq{Something: &test.SaySomethingReq_Name{Name: "小张"}})
	stream2.Send(&test.SaySomethingReq{Something: &test.SaySomethingReq_SubMessage{SubMessage: &test.SubMessage{AudioName: "音频", AudioPath: "路径"}}})
	stream2.Send(&test.SaySomethingReq{Something: &test.SaySomethingReq_Name{Name: "小红"}})
	stream2.Send(&test.SaySomethingReq{Something: &test.SaySomethingReq_SubMessage{SubMessage: &test.SubMessage{AudioName: "音频1", AudioPath: "路径1"}}})
	stream2.Send(&test.SaySomethingReq{Something: &test.SaySomethingReq_Name{Name: "小李"}})
	stream2.Send(&test.SaySomethingReq{Something: &test.SaySomethingReq_SubMessage{SubMessage: &test.SubMessage{AudioName: "音频2", AudioPath: "路径2"}}})
	//stream2.CloseSend()
	//time.Sleep(10 *time.Millisecond)
	stream2.CloseSend()
	go func() {
		for {
			// 接收从 服务端返回的数据流
			resp, err := stream2.Recv()
			if err == io.EOF {
				log.Println("⚠️ 收到服务端的结束信号")
				break //如果收到结束信号，则退出“接收循环”，结束客户端程序
			}

			if err != nil {
				// TODO: 处理接收错误
				log.Println("接收数据出错:", err)
				break
			}

			// 没有错误的情况下，打印来自服务端的消息
			log.Printf("[客户端收到]: %s", resp.Message)
		}
	}()
	time.Sleep(time.Millisecond * 20)
}
