package main

import ()

var cmdFix = &FixCommand{
	UsageLine: "fix",
	Short:     "Update a Iceberg Liasion server",
	Long: `
Update a Iceberg Liasion server using most version Iceberg frame
`,
}

// FixCommand 更新系统
type FixCommand struct {
	UsageLine string
	Short     string
	Long      string
}

// Run 运行命令
func (cmd *FixCommand) Run(args []string) {

}

// Name 获取命令名称
func (cmd *FixCommand) Name() string {
	return ""
}
