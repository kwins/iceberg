package frame

import (
	goctx "context"
	"encoding/json"
	"encoding/xml"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	log "github.com/kwins/iceberg/frame/icelog"
	"github.com/kwins/iceberg/frame/protocol"

	"github.com/golang/protobuf/proto"
)

// Context 请求上下文
// Context包含了请求的所有信息，并封装了一系列所需的操作
type Context interface {
	// Context
	Ctx() goctx.Context

	// Bizid 服务全局ID
	Bizid() string

	// Reset Reset
	Reset(r *protocol.Proto, w *protocol.Proto)

	// 原始请求信息
	Request() *protocol.Proto

	// Response 响应信息
	Response() *protocol.Proto

	// Bind Parse Request data
	Bind(i interface{}) error

	// HTTP Header
	Header() http.Header

	// 请求数据序列化格式
	ReqFormat() protocol.RestfulFormat

	// 响应数据序列化格式
	RespFormat() protocol.RestfulFormat

	// Client Request RealIP
	RealIP() string

	// http 表单数据和raw query都使用此结构获取看k,v对
	FormValue(name string) string

	// FormValues FormValues
	FormValues() url.Values

	// GetString 获取FormValue的值
	GetString(name string, defaultValue string) string

	// GetInt 获取FormValue的值
	GetInt(name string, defaultValue int) int

	// GetUint 获取FormValue值并转化为 uint64
	GetUint(name string, defaultValue uint64) uint64

	// GetFloat 获取FormValue值并转化为 float64
	GetFloat(name string, defaultValue float64) float64

	// Ctx Get
	Get(key string) interface{}

	// Ctx Set
	Set(key string, val interface{})

	// JSON 响应JSON数据
	JSON(i interface{}) error

	// XML 响应XML数据
	XML(i interface{}) error

	// PROTOBUF 响应PROTOBUF数据
	Protobuf(i proto.Message) error

	// Bytes 响应数据
	Bytes(i []byte) error

	// String 响应数据
	String(s string) error

	// JSON2 返回code和msg的json数据格式
	JSON2(code int, msg string, data interface{}) error

	// XML2 返回code和msg的xml数据格式
	XML2(code int, msg string, data interface{}) error

	Debug(args ...interface{})

	Warn(args ...interface{})

	Info(args ...interface{})

	Error(args ...interface{})

	Fatal(args ...interface{})

	Debugf(fmt string, args ...interface{})

	Warnf(fmt string, args ...interface{})

	Infof(fmt string, args ...interface{})

	Errorf(fmt string, args ...interface{})

	Fatalf(fmt string, args ...interface{})
}

// NewContext new context
func NewContext() Context {
	return &icecontext{
		req:    nil,
		resp:   nil,
		header: nil,
		form:   nil}
}

var defaultContext = NewContext()

type icecontext struct {
	req       *protocol.Proto
	resp      *protocol.Proto
	header    http.Header
	srcFormat protocol.RestfulFormat
	dstFormat protocol.RestfulFormat
	form      url.Values
	clientip  string
	ctx       goctx.Context
}

// Ctx 将一些需要的参数传递给下一个请求的Context
// 多层级调用链，使用go context将bizid传递下去
func (c *icecontext) Ctx() goctx.Context {
	if c.ctx == nil {
		c.ctx = goctx.TODO()
	}
	return goctx.WithValue(c.ctx, "bizid", c.Bizid())
}

// Bizid 用户追踪ID
func (c *icecontext) Bizid() string {
	if c.req != nil {
		return c.req.GetBizid()
	}
	return ""
}

// Reset 调用其他方法前已在框架中Reset，故其他方法获取参数是安全的
func (c *icecontext) Reset(r *protocol.Proto, w *protocol.Proto) {
	c.req = r
	c.resp = w
	c.srcFormat = r.GetFormat()
	c.dstFormat = protocol.RestfulFormat_FORMATNULL

	c.header = make(http.Header)
	for k, v := range r.GetHeader() {
		c.header.Set(k, v)
	}

	c.form = make(url.Values)
	for k, v := range r.GetForm() {
		c.form.Set(k, v)
	}

	c.clientip = ""
	c.ctx = goctx.TODO()
}

// Header HTTP header
func (c *icecontext) Header() http.Header {
	if c.header == nil {
		c.header = make(http.Header)
		for k, v := range c.req.GetHeader() {
			c.header.Set(k, v)
		}
	}
	return c.header
}

// Request 原始请求信息
func (c *icecontext) Request() *protocol.Proto {
	return c.req
}

// Response 响应信息
func (c *icecontext) Response() *protocol.Proto {
	return c.resp
}

// Bind Parse Request data
func (c *icecontext) Bind(i interface{}) error {
	return protocol.Unpack(c.Request().GetFormat(),
		c.Request().GetBody(), i)
}

// ReqFormat 请求数据序列化格式
func (c *icecontext) ReqFormat() protocol.RestfulFormat {
	if c.req != nil {
		return c.req.GetFormat()
	}
	return protocol.RestfulFormat_FORMATNULL
}

// RespFormat 响应数据序列化格式
func (c *icecontext) RespFormat() protocol.RestfulFormat {
	if c.resp != nil {
		return c.resp.GetFormat()
	}
	return protocol.RestfulFormat_FORMATNULL
}

// URL RAW Query data
func (c *icecontext) FormValue(name string) string {
	return c.form.Get(name)
}

// GetString 获取FormValue中的参数值
func (c *icecontext) GetString(name string, defaultValue string) string {
	if v := c.form.Get(name); v != "" {
		return v
	}
	return defaultValue
}

// GetInt 获取FormValue中的参数值
func (c *icecontext) GetInt(name string, defaultValue int) int {
	if v := c.form.Get(name); v != "" {
		number, err := strconv.Atoi(v)
		if err != nil {
			c.Error(err.Error())
			return defaultValue
		}
		return number
	}
	return defaultValue
}

// GetUint 获取FormValue中的参数值
func (c *icecontext) GetUint(name string, defaultValue uint64) uint64 {
	if v := c.form.Get(name); v != "" {
		number, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			c.Error(err.Error())
			return defaultValue
		}
		return number
	}
	return defaultValue
}

// GetFloat 获取FormValue中的参数值
func (c *icecontext) GetFloat(name string, defaultValue float64) float64 {
	if v := c.form.Get(name); v != "" {
		number, err := strconv.ParseFloat(v, 64)
		if err != nil {
			c.Error(err.Error())
			return defaultValue
		}
		return number
	}
	return defaultValue
}

// FormValues FormValues
func (c *icecontext) FormValues() url.Values {
	return c.form
}

// RealIP Client Request RealIP
func (c *icecontext) RealIP() string {
	if c.clientip == "" {
		ra := c.Request().GetRemoteAddr()
		if ip := c.Header().Get(protocol.HeaderXForwardedFor); ip != "" {
			ra = strings.Split(ip, ", ")[0]
		} else if ip := c.Header().Get(protocol.HeaderXRealIP); ip != "" {
			ra = ip
		} else {
			ra, _, _ = net.SplitHostPort(ra)
		}
		c.clientip = ra
		return ra
	}
	return c.clientip
}

// Get Ctx Get
func (c *icecontext) Get(key string) interface{} {
	if c.ctx == nil {
		return nil
	}
	return c.ctx.Value(key)
}

// Set Ctx Set
func (c *icecontext) Set(key string, val interface{}) {
	if c.ctx == nil {
		c.ctx = goctx.TODO()
	}
	goctx.WithValue(c.ctx, key, val)
}

// JSON 响应JSON数据
func (c *icecontext) JSON(i interface{}) error {
	c.dstFormat = protocol.RestfulFormat_JSON
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	if c.resp == nil {
		s := c.Request().Shadow()
		c.resp = &s
	}
	c.resp.Format = protocol.RestfulFormat_JSON
	c.resp.Body = b
	return nil
}

// XML 响应XML数据
func (c *icecontext) XML(i interface{}) error {
	c.dstFormat = protocol.RestfulFormat_XML
	b, err := xml.Marshal(i)
	if err != nil {
		return err
	}
	if c.resp == nil {
		s := c.Request().Shadow()
		c.resp = &s
	}
	c.resp.Format = protocol.RestfulFormat_XML
	c.resp.Body = b
	return nil
}

// PROTOBUF 响应PROTOBUF数据
func (c *icecontext) Protobuf(i proto.Message) error {
	c.dstFormat = protocol.RestfulFormat_PROTOBUF
	b, err := proto.Marshal(i)
	if err != nil {
		return err
	}
	if c.resp == nil {
		s := c.Request().Shadow()
		c.resp = &s
	}
	c.resp.Format = protocol.RestfulFormat_PROTOBUF
	c.resp.Body = b
	return nil
}

// Bytes 响应Bytes数据
func (c *icecontext) Bytes(i []byte) error {
	c.dstFormat = protocol.RestfulFormat_RAWQUERY
	if c.resp == nil {
		s := c.Request().Shadow()
		c.resp = &s
	}
	c.resp.Format = protocol.RestfulFormat_RAWQUERY
	c.resp.Body = i
	return nil
}

// String 响应数据
func (c *icecontext) String(s string) error {
	c.dstFormat = protocol.RestfulFormat_RAWQUERY
	if c.resp == nil {
		s := c.Request().Shadow()
		c.resp = &s
	}
	c.resp.Format = protocol.RestfulFormat_RAWQUERY
	c.resp.Body = []byte(s)
	return nil
}

// JSON2 公共JSON响应
func (c *icecontext) JSON2(code int, msg string, data interface{}) error {
	return c.JSON(&protocol.Message{
		Errcode: code,
		Errmsg:  msg,
		Data:    data,
	})
}

// XML2 公共XML响应
func (c *icecontext) XML2(code int, msg string, data interface{}) error {
	return c.XML(&protocol.Message{
		Errcode: code,
		Errmsg:  msg,
		Data:    data,
	})
}

func (c *icecontext) appendBiz(args []interface{}) []interface{} {
	var newargs = make([]interface{}, len(args)+1)
	newargs[0] = "Bizid:" + c.Bizid() + " "
	for i := range args {
		newargs[i+1] = args[i]
	}
	return newargs
}

func (c *icecontext) appendf(fmt string, args []interface{}) (string, []interface{}) {
	return "%s " + fmt, c.appendBiz(args)
}

// Debug global debug
func (c *icecontext) Debug(args ...interface{}) {
	log.Default().FormatAndOutput(4, log.DEBUG, "", c.appendBiz(args)...)
}

// Warn defalut warn
func (c *icecontext) Warn(args ...interface{}) {
	log.Default().FormatAndOutput(4, log.WARNING, "", c.appendBiz(args)...)
}

// Info default info
func (c *icecontext) Info(args ...interface{}) {
	log.Default().FormatAndOutput(4, log.INFO, "", c.appendBiz(args)...)
}

// Error default error
func (c *icecontext) Error(args ...interface{}) {
	log.Default().FormatAndOutput(4, log.ERROR, "", c.appendBiz(args)...)
}

// Fatal default fatal
func (c *icecontext) Fatal(args ...interface{}) {
	log.Default().FormatAndOutput(4, log.FATAL, "", c.appendBiz(args)...)
}

// Debugf global debug
func (c *icecontext) Debugf(fmt string, args ...interface{}) {
	fmt, args = c.appendf(fmt, args)
	log.Default().FormatAndOutput(4, log.DEBUG, fmt, args...)
}

// Warnf defalut wawrn
func (c *icecontext) Warnf(fmt string, args ...interface{}) {
	fmt, args = c.appendf(fmt, args)
	log.Default().FormatAndOutput(4, log.WARNING, fmt, args...)
}

// Infof default info
func (c *icecontext) Infof(fmt string, args ...interface{}) {
	fmt, args = c.appendf(fmt, args)
	log.Default().FormatAndOutput(4, log.INFO, fmt, args...)
}

// Errorf default error
func (c *icecontext) Errorf(fmt string, args ...interface{}) {
	fmt, args = c.appendf(fmt, args)
	log.Default().FormatAndOutput(4, log.ERROR, fmt, args...)
}

// Fatalf default fatal
func (c *icecontext) Fatalf(fmt string, args ...interface{}) {
	fmt, args = c.appendf(fmt, args)
	log.Default().FormatAndOutput(4, log.FATAL, fmt, args...)
}

// TODO 默认
func TODO() Context {
	return defaultContext
}
