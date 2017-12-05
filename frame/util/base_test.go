package util

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type ABC struct {
	g string
}

func (c *ABC) Handlefunc1(a, b int) int {
	return a + b
}

func (c *ABC) Handlefunc2(a, b int) int {
	return a - b
}

func (c *ABC) Func2(a, b int) int {
	return a - b
}

type MyType struct {
	i    int
	name string
}

func (mt *MyType) SetI(i int) {
	mt.i = i
}

func (mt *MyType) SetName(name string) {
	mt.name = name
}

func (mt *MyType) String() string {
	return fmt.Sprintf("%p", mt) + "--name:" + mt.name + " i:" + strconv.Itoa(mt.i)
}

func Test_A(t *testing.T) {

	A := new(ABC)

	rg := reflect.ValueOf(A)
	vft := rg.Type()

	AAA := make(map[string]reflect.Value, 5)
	num := rg.NumMethod()
	for i := 0; i < num; i++ {

		name := vft.Method(i).Name
		// 注册前缀为Handle的方法
		if strings.HasPrefix(name, "Handle") {
			AAA[name] = rg.Method(i)
			fmt.Printf("method %s registed\n", name)
		}

		m := rg.Method(i).Call([]reflect.Value{reflect.ValueOf(1), reflect.ValueOf(2)})
		fmt.Printf(" %d\n", m[0])
	}

	// //调用方法
	r := AAA["Handlefunc1"].Call([]reflect.Value{reflect.ValueOf(1), reflect.ValueOf(2)})
	fmt.Printf(" %d\n", r[0])

}

func TestGetHostname(t *testing.T) {
	t.Log(GetHostname())
}
