package protocol

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
)

// HeaderLength 包头长度
const HeaderLength = 4 // 包的长度，单位bytes

// 定义响应类型
const (
	OK = iota
	BadProtocol
	ParseRequestFail
	ParseResponseFail
	RequestFail
	NotSupportMethod
	NotFoundInterface
	SvrReturnInvalidData
	SvrInternalFail
)

// ErrInfo 服务之间通用的错误信息结构
type ErrInfo struct {
	ErrCode int    `json:"errcode"`
	ErrInfo string `json:"errmsg"`
}

// Proto 服务层内部接口的协议采用json编码。
// 在TCP上传输。每个请求包之前用4 byte的表示包的长度。协议看上去如下：
// +------------------------+
// | Length |  Proto        |
// +------------------------+
// RestfulMethod restful的请求方法定义
// 由于go的http包用字符串表示方法，为了高效和节省存储占用转换成用数字表示的方法
// type RestfulMethod uint16

// 定义HTTP方法
// const (
// 	GET    RestfulMethod = 0x1
// 	POST                 = 0x2
// 	PUT                  = 0x4
// 	DELETE               = 0x8
// )
// Serialize serialize pack
func serialize(b []byte) ([]byte, error) {
	length := uint32(len(b) + HeaderLength)
	bw := make([]byte, length)
	bw[0] = byte(length >> 24)
	bw[1] = byte(length >> 16)
	bw[2] = byte(length >> 8)
	bw[3] = byte(length)
	copy(bw[HeaderLength:], b)
	return bw, nil
}

// Serialize 将Proto结构的内容写入一段连续的字节块,并在块头用4byte表示字节块的长度。
// 该长度包含这4byte在内
func (pro *Proto) Serialize() ([]byte, error) {
	b, err := proto.Marshal(pro)
	if err != nil {
		return nil, err
	}
	return serialize(b)
}

// Extract extract mediaType and pack
func Extract(stream []byte) (pack []byte, err error) {
	_ = stream[3] // 边界检查
	length := uint32(stream[3]) | uint32(stream[2])<<8 | uint32(stream[1])<<16 | uint32(stream[0])<<24
	if int(length) != len(stream) {
		err = errors.New("pack is crashed")
		return
	}
	pack = stream[4:]
	return
}

// UnSerialize 从一段连续的字节块中解析Proto结构的内容
// 块头的4byte表示字节块的长度(该长度包含这4byte在内)
func (pro *Proto) UnSerialize(srcBuf []byte) error {
	b, err := Extract(srcBuf)
	if err != nil {
		return err
	}
	return proto.Unmarshal(b, pro)
}

// Shadow 生成该结果的一个影子。所谓影子是指生产的影子结构和该结果本身除了
// body， err 两个字段内容为空以外，其他的字段都一样。
// 影子结构的用途是在有限的复制请求，作为响应的基础。这样就能保证请求和响应的
// bizid, requestID是一致的。不容易在写代码的复制，粘贴过程出现失误。
func (pro *Proto) Shadow() Proto {
	shadow := Proto{
		Bizid:       pro.Bizid,
		RequestID:   pro.GetRequestID(),
		ServeURI:    pro.GetServeURI(),
		ServeMethod: pro.GetServeMethod(),
		Method:      pro.GetMethod()}
	return shadow
}

// Context 返回一个包裹Bizid和Header信息的Context
func (pro *Proto) Context() context.Context {
	ctx := context.TODO()
	ctx = context.WithValue(ctx, "bizid", pro.GetBizid())
	for k, v := range pro.Header {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}

// SetHeader set http header
func (pro *Proto) SetHeader(key, val string) {
	if pro.Header == nil {
		pro.Header = make(map[string]string)
	}
	pro.Header[key] = val
}

// ForeachKey 实现opentracing TextMapReader接口，用于opentacing Extract
func (pro *Proto) ForeachKey(handler func(key, val string) error) error {
	for k, v := range pro.GetTraceMap() {
		if err := handler(k, v); err != nil {
			return err
		}
	}
	return nil
}

// AsString 将结构体序列化后的结果，转成可读的字符串
// 其实就是剥离包头表示长度的字节，因为序列化是json操作。所以剥离包头的长度后就是可读的内容了
func (pro *Proto) AsString() string {
	return fmt.Sprintf("Bizid:%s RequestID:%d ServeURI:%s Format:%s ServeMethod:%s Method:%s Header:%v Form:%s Body:%s Err:%s",
		pro.GetBizid(), pro.GetRequestID(), pro.GetServeURI(), pro.GetFormat(), pro.GetServeMethod(), pro.GetMethod(), pro.GetHeader(), pro.GetForm(), pro.GetBody(), pro.GetErr())
}

// FillErrInfo 填充错误信息
func (pro *Proto) FillErrInfo(code int, err error) {
	pro.Body = make([]byte, 0)
	errInfo := ErrInfo{ErrCode: code, ErrInfo: err.Error()}
	pro.Err, _ = json.Marshal(&errInfo)
}

// Raw 处理原始报文
type Raw interface {
	Get() []byte
	Set([]byte) error
}

// Unpack 对请求数据进行反序列化
// 对于请求来的数据，约定GET请求对数据不进行序列化，Body内容为空
func Unpack(fromat RestfulFormat, in []byte, out interface{}) error {
	switch fromat {
	case RestfulFormat_XML:
		if err := xml.Unmarshal(in, out); err != nil {
			return err
		}
	case RestfulFormat_RAWQUERY:
		switch v := out.(type) {
		case Raw:
			v.Set(in)
		case []byte:
			v = in
		case string:
			v = string(in)
		default:
			return fmt.Errorf("Not support out type: %v", v)
		}
	case RestfulFormat_JSON:
		if err := json.Unmarshal(in, out); err != nil {
			return err
		}
	case RestfulFormat_PROTOBUF:
		switch v := out.(type) {
		case proto.Message:
			if err := proto.Unmarshal(in, v); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Not protobuf data: %v", v)
		}
	default:
		return fmt.Errorf("Not support format: %v", fromat)
	}
	return nil
}

// Pack 对请求响应数据进行序列化
// 如果是GET请求，后端响应默认使用JSON编码
func Pack(format RestfulFormat, data interface{}) ([]byte, error) {
	var b []byte
	var err error

	switch format {
	case RestfulFormat_XML:
		b, err = xml.Marshal(data)
		if err != nil {
			return nil, err
		}
	case RestfulFormat_RAWQUERY:
		switch v := data.(type) {
		case Raw:
			b = v.Get()
		case []byte:
			b = v
		case string:
			b = []byte(v)
		default:
			return nil, fmt.Errorf("Not support data type: %v", v)
		}
	case RestfulFormat_PROTOBUF:
		b, err = proto.Marshal(data.(proto.Message))
		if err != nil {
			return nil, err
		}
	case RestfulFormat_JSON, RestfulFormat_FORMATNULL:
		b, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}
