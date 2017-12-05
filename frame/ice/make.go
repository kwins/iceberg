package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

const (
	linux = "linux"
	mac   = "mac"
)

var cmdMake = &MakeCommand{
	UsageLine: "make [option]",
	Short:     "Compile a Iceberg Liasion server",
	Long: `
Compile a Iceberg Liasion server in different platform
        linux    compile program in linux platform
        mac      compile program in macOs platform   
        clean    clean program in current floder
`,
}
var content = `package main
import "fmt"

func showGitFinger() {
%s
}
`
var content1 = "\tfmt.Println(\"repo[%s]'s git finger: %s\")\n"

// MakeCommand 更新系统
type MakeCommand struct {
	UsageLine   string
	Short       string
	Long        string
	currentPath string
}

// Run 运行命令
func (cmd *MakeCommand) Run(args []string) {
	if len(args) < 2 {
		fmt.Println(cmdHelp.Long)
	}
	if len(args) == 2 {
		compile("")
		return
	}
	compile(args[2])
}

// Name 获取命令名称
func (cmd *MakeCommand) Name() string {
	cmdArr := strings.Split(cmd.UsageLine, " ")
	return cmdArr[0]
}

// 获取一个目录的git 哈希值
func gitFinger(path string) {
	if err := os.Chdir(path); err != nil {
		fmt.Printf("ch dir [%s] error: %s", path, err.Error())
		return
	}
	fmt.Printf("Enter repo path: %s\n", path)
	cmd := exec.Command("git", "log")
	out := bytes.NewBuffer(nil)
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		fmt.Printf("exec git log error: %s", out.String())
		return
	}
	line, err := out.ReadBytes('\n')
	if err != nil {
		fmt.Printf("read git log error: %s", err.Error())
		return
	}
	lineArr := strings.Split(string(line), " ")
	pathArr := strings.Split(path, "/")
	pathLength := len(pathArr)
	if pathLength < 2 || len(lineArr) != 2 {
		return
	}
	hash[pathArr[pathLength-1]] = strings.TrimRight(lineArr[1], "\n")
	fmt.Printf("Repo %s 's git finger: %s\n", pathArr[pathLength-1], lineArr[1])
}

var hash = make(map[string]string)

// 遍历当前目录，找到src仓库
func walk(root string) bool {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name(), ".git") {
			return true
		}
	}
	return false
}

func listGitDependence(root string) {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return
	}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		dir := root + "/" + file.Name()
		gitFinger(dir)
	}
	createOrModifyGitFinger()
}

func createOrModifyGitFinger() {
	var fileContent string
	for key, value := range hash {
		fileContent += fmt.Sprintf(content1, key, value)
	}
	// fmt.Println(fileContent)
	file := fmt.Sprintf(content, fileContent)
	fHandler, err := os.Create(cmdMake.currentPath + "/gitfinger.go")
	if err != nil {
		fmt.Printf("create gitfinger file error: %s", err.Error())
		return
	}
	_, err = fHandler.WriteString(file)
	if err != nil {
		fmt.Printf("write conten to [%s] error: %s", cmdMake.currentPath+"/gitfinger.go", err.Error())
	}
}

func makeBin() {
	startPath := currentPath()
	cmdMake.currentPath = startPath
	fmt.Printf("Make work in: %s\n", startPath)
	for {
		if walk(startPath) {
			break
		}
		startPath = upper(startPath)
	}
	root := upper(startPath)
	if err := os.Chdir(root); err != nil {
		fmt.Printf("ch dir [%s] error: %s", root, err.Error())
		return
	}
	listGitDependence(root)
}

func upper(path string) string {
	pathArr := strings.Split(path, "/")
	if len(pathArr) < 1 {
		return ""
	}
	return strings.Join(pathArr[:len(pathArr)-1], "/")
}

func compile(arg string) {
	if arg == "clean" {
		cmd := exec.Command("go", arg)
		cmd.Run()
		fmt.Println("clean success")
		return
	}
	makeBin()
	var params1, params2, params3 string
	var goos string
	pathArr := strings.Split(cmdMake.currentPath, "/")
	params3 = pathArr[len(pathArr)-1] + ".exe"
	switch arg {
	case "linux":
		{
			params1 = "build"
			params2 = "-o"
			goos = fmt.Sprintf("GOOS=%s", "linux")
		}

	case "mac":
		{
			params1 = "build"
			params2 = "-o"
			goos = fmt.Sprintf("GOOS=%s", "darwin")
		}

	case "":
		{
			params1 = "build"
			params2 = "-o"
			arg = "darwin"
			goos = fmt.Sprintf("GOOS=%s", "darwin")
		}
	default: // 默认编译macOs
		fmt.Printf(cmdHelp.Long)
		return
	}

	cmd := exec.Command("go", params1, params2, params3)
	cmd.Stdout = os.Stdout
	cmd.Env = append(cmd.Env, goos)
	cmd.Env = append(cmd.Env, "GOARCH=amd64")
	cmd.Env = append(cmd.Env, "GOPATH="+os.Getenv("GOPATH"))

	errBuf := bytes.NewBuffer(nil)
	cmd.Stderr = errBuf
	if err := os.Chdir(cmdMake.currentPath); err != nil {
		fmt.Printf("ch dir [%s] error: %s", cmdMake.currentPath, err.Error())
	}
	if err := cmd.Run(); err != nil {
		fmt.Printf("complie error\n: %s\n", errBuf.String())
		return
	}
	fmt.Printf("complie target: %s success (%s)", params3, arg)
}
