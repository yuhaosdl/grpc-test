package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"google.golang.org/grpc"
	test "grpc-test/proto"
	"io"
	"log"
	"time"
)

type Server struct {
}

func (s *Server) SayHello(ctx context.Context, in *test.HelloReq) (*test.HelloResp, error) {
	resp := fmt.Sprintf("hello:%s", in.Name)
	fmt.Print(resp)
	//parentSpan := opentracing.SpanFromContext(ctx)
	//if parentSpan != nil {
	//	log.Printf("got Parenet ctx in SayHello")
	//} else {
	//	log.Printf("not got Parenet ctx in SayHello")
	//}
	clientTest(ctx)
	return &test.HelloResp{
		Message: resp,
	}, nil
}
func (s *Server) SayBye(ctx context.Context, in *test.ByeReq) (*test.ByeResp, error) {
	//proto时间戳转换为go时间
	timestamp, err := ptypes.Timestamp(in.ByeTime)
	if err != nil {
		fmt.Println(err)
	}
	resp := fmt.Sprintf("%s:bye:%s", timestamp, in.Name)
	fmt.Print(resp)
	return &test.ByeResp{
		Message: resp,
	}, nil
}

func (s *Server) AlwaysSay(stream test.Greeter_AlwaysSayServer) error {
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			log.Println("收到客户端通过context发出的终止信号")
			return ctx.Err()
		default:
			// 接收从客户端发来的消息
			recv, err := stream.Recv()
			if err == io.EOF {
				log.Println("客户端发送的数据流结束")
				return nil
			}
			if err != nil {
				log.Println("接收数据出错:", err)
				return err
			}
			output := fmt.Sprintf("你好，%s", recv.Name)
			// 缺省情况下， 返回 '服务端返回: ' + 输入信息
			fmt.Printf("[收到消息]: %s\n", recv.Name)
			if err := stream.Send(&test.StreamResp{Message: "服务端返回: " + output}); err != nil {
				return err
			}
		}
	}

	return nil
}
func (s *Server) SaySomething(stream test.Greeter_SaySomethingServer) error {
	ctx := stream.Context()
	for {
		select {
		case <-ctx.Done():
			log.Println("收到客户端通过context发出的终止信号")
			return ctx.Err()
		default:
			// 接收从客户端发来的消息
			recv, err := stream.Recv()
			if err == io.EOF {
				log.Println("客户端发送的数据流结束")
				return nil
			}
			if err != nil {
				log.Println("接收数据出错:", err)
				return err
			}
			switch recv.GetSomething().(type) {
			case *test.SaySomethingReq_Name:
				fmt.Println("你好：" + recv.GetName())
				if err := stream.Send(&test.SaySomethingResp{Message: "你好：" + recv.GetName()}); err != nil {
					return err
				}
			case *test.SaySomethingReq_SubMessage:
				fmt.Println("收到音频：" + recv.GetSubMessage().AudioName)
				if err := stream.Send(&test.SaySomethingResp{Message: "接收到音频：" + recv.GetSubMessage().AudioPath}); err != nil {
					return err
				}
			default:
				return errors.New("XXXXXXXXXXX")
			}
		}
	}
	return nil
}

func clientTest(ctx context.Context) {

	// 建立连接到gRPC服务
	//39.106.8.113
	conn, err := grpc.Dial("localhost:8089",
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(&customCredential{}),
		grpc.WithChainUnaryInterceptor(grpc_opentracing.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(grpc_opentracing.StreamClientInterceptor()))
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
	//ctx := context.Background()
	// 调用gRPC接口
	tr, err := t.SayHello(ctx, &test.HelloReq{Name: res})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("服务端响应:%s", tr.Message)
	fmt.Println("-----------------------------------------------")
	//ctx, _ = context.WithCancel(context.Background())
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
	//ctx2, cancel2 := context.WithCancel(context.Background())
	//defer cancel2()
	stream2, err := t.SaySomething(ctx)
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
