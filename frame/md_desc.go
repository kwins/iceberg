package frame

// Medesc 方法描述
// 是否能无认证访问
// 流量统计
// 失败统计
type Medesc struct {
	MdName  string `json:"md_name"`
	Allowed bool   `json:"allowed"`
	FailCnt int64  `json:"fail_cnt"`
	Cnt     int64  `json:"cnt"`
}
