package frame

// SrvVersion 服务版本
type SrvVersion int8

// 定义服务版本
const (
	SV1 SrvVersion = 1
	SV2 SrvVersion = 2
)

// SrvVersionName 定义服务版本映射
var SrvVersionName = map[SrvVersion]string{
	SV1: "v1",
	SV2: "v2",
}
