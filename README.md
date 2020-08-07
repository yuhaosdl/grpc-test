
```
protoc -I proto/ -I e://go/src  --go_out=plugins=grpc:proto test.proto
protoc -I ./proto  -I e://go/src --grpc-gateway_out=logtostderr=true:proto/ test.proto
protoc -I proto --swagger_out=logtostderr=true:proto/ test.proto
```
####zipkin运行
````
docker run -d -p 9411:9411 openzipkin/zipkin
````
