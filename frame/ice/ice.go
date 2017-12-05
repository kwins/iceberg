package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// iceberg version
const (
	version = "0.0.1"
)

// Command ice工具接口
type Command interface {
	Run(args []string)
	Name() string
}

var commands = map[string]Command{
	"new":     cmdNewLiasion,
	"make":    cmdMake,
	"fix":     cmdFix,
	"help":    cmdHelp,
	"version": cmdVersion,
}

func execCommand(command string, args ...string) *bytes.Buffer {
	cmd := exec.Command(command, args...)
	cmdBuf := bytes.NewBuffer(nil)
	cmd.Stdout = cmdBuf
	if err := cmd.Run(); err != nil {
		fmt.Printf("exec command: (%v) error: %s [%s]", cmd.Args, err.Error(), cmdBuf.String())
		return nil
	}
	return cmdBuf
}

// ice new [name]
// ice fix
// ice make [option] [project path]
// ice help
// ice version
func main() {
	if len(os.Args) < 2 {
		fmt.Println(cmdHelp.Long)
		return
	}
	if cmd, ok := commands[os.Args[1]]; ok {
		cmd.Run(os.Args)
	} else {
		fmt.Println(cmdHelp.Long)
	}
}

// 获取当前路径
func currentPath() string {
	cmd := exec.Command("pwd")
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	if err := cmd.Run(); err != nil {
		return ""
	}
	ret := buf.String()
	ret = strings.TrimRight(ret, "\n")
	currentpath := strings.Trim(ret, " ")
	return currentpath
}
