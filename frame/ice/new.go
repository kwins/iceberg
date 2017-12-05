package main

var cmdNewLiasion = &NewLiasionCommand{
	UsageLine: "new [svrname]",
	Short:     "Create a Iceberg Liasion server",
	Long: `
Creates a Iceberg Liasion server for the given server name in the current directory.

The command 'new' creates a folder named [svrname] and inside the folder deploy
the following files/directories structure:

    |- config
        |-  config.go 
        |-  config_test.go
    |- model
        |-  base.go
    |- operator
        |-  operator.go
        |-  operator_test.go
    |- server
        |-  server.go
    |- web
        |-  web.go
        |-  web_test.go
    |- main.go
    |- conf.json
    |- seelog.xml
`,
}

// NewLiasionCommand 更新系统
type NewLiasionCommand struct {
	UsageLine string
	Short     string
	Long      string
}

// Run 运行命令
func (cmd *NewLiasionCommand) Run(args []string) {

}

// Name 获取命令名称
func (cmd *NewLiasionCommand) Name() string {
	return ""
}
