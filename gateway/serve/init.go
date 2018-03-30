package serve

import (
	// sql driver
	_ "github.com/go-sql-driver/mysql"
)

var errGatewayTimeout = `{"errcode":504,"errmsg":"请求超时"}`
var errRequestInvalide = `{"errcode":400,"errmsg":"请求无效"}`
var errAuthFail = `{"errcode":-1002,"errmsg":"认证失败"}`
var errNotFounHTTPMethod = `{"errcode":404,"errmsg":"资源不存在"}`
var errInternalError = `{"errcode":500,"errmsg":"服务器开了点小差，请稍后再试～"}`
