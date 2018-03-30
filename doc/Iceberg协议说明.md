# 协议
为了实现 http+tcp 的服务架构，后端服务需要一套通用的协议。

## 协议结构
* 使用protobuf进行序列化和反序列，性能相对较高。
```protobuf
message Proto{
    string Bizid =1; 
	map<string,string> Header = 2;
    map<string,string> TraceMap = 3;
	int64  RequestID = 4; 
	string ServeURI = 5; 
    RestfulFormat Format = 6; 
    string ServeMethod = 7; 
	RestfulMethod Method = 8;
	bytes Body = 9; 
	bytes Err  = 10;
}
```

* Bizid: 全局唯一ID，用于跨服务多实例对请求进行追踪，一个请求夸多服务，会使用同一个Bizid。

* Header: HTTP Header

* TraceMap：记录Trace信息，集成zipkin trace功能

* RequestID：递增请求ID，服务内唯一，异步请求标识ID。

* ServeURI：路由 Path，客户端请求Path，同时也是存储在ETCD中标识一个服务KEY。

* RestfulFormat：数据编码格式，目前支持 json，xml，protobuf，原始数据。其他类型，默认为原始数据类型，如果为原始数据类型，服务端请求和响应数据结构，必须实现 **RAW** 接口。

* ServeMethod：服务方法 如：CreateOrderWithPay，对应服务内一个Handler

* Body：请求/正确响应 数据

* Err：内部错误时，响应信息，Body 和 Err 互斥