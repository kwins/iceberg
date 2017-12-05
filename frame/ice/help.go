package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

// HelpCommand 更新系统
type HelpCommand struct {
	UsageLine string
	Short     string
	Long      string
}

// Run 运行命令
func (cmd *HelpCommand) Run(args []string) {
	fmt.Println(cmd.Long)
}

// Name 获取命令名称
func (cmd *HelpCommand) Name() string {
	return ""
}

var cmdHelp = &HelpCommand{
	UsageLine: "help",
	Short:     "Iceberg command useage",
	Long: `
Iceberg command useage
	ice new  [name] [project path]     >   Create a Iceberg Liasion server
	ice make [option]                  >   Compile a Iceberg Liasion server
	ice fix                            >   Update a Iceberg Liasion server
	ice help                           >   Iceberg command useage
	ice version                        >   Show Iceberg version
`,
}

// VersionCommand 更新系统
type VersionCommand struct {
	UsageLine   string
	Short       string
	Long        string
	currentPath string
}

// Run 运行命令
func (cmd *VersionCommand) Run(args []string) {
	command := exec.Command("go", "version")
	out := bytes.NewBuffer(nil)
	command.Stdout = out
	command.Run()
	fmt.Println(out.String() + "iceberg version: " + version)
}

// Name 获取命令名称
func (cmd *VersionCommand) Name() string {
	return ""
}

var cmdVersion = &VersionCommand{
	UsageLine: "version",
	Short:     "Show Iceberg version",
	Long: `
    Show Iceberg current version
`,
}
