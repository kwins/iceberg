# builds the gatesvr and s1 & s2 projects
all:clean
	@go get github.com/opentracing/opentracing-go
	@go get github.com/openzipkin/zipkin-go-opentracing
	@go get github.com/coreos/etcd/clientv3
	@go get github.com/nobugtodebug/go-objectid
	@go get github.com/golang/protobuf/proto
	@go build -v -o build/gatesvr ../gatesvr
	@go build -v -o build/s1 ./s1
	@go build -v -o build/s2 ./s2
	@cp ../gatesvr/gatesvr_conf.json build/
clean:
	@rm -rf build
