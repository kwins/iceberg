package main

import (
	"net/http"
	"os"

	hello "github.com/kwins/iceberg/demo/s1/pb"
	hi "github.com/kwins/iceberg/demo/s2/pb"
	"github.com/kwins/iceberg/frame"
	log "github.com/kwins/iceberg/frame/icelog"
)

// Hi 对象
type Hi struct {
}

// SayHi handel message 01
func (id *Hi) SayHi(c frame.Context) error {

	var res hi.HiResponse

	var foo hello.HelloRequest
	foo.Name = "quinn-wang"
	log.Debug("####:", c.Ctx())
	resp, err := hello.SayHello(c, &foo, frame.Header(http.Header{
		"A": []string{
			"AAAAA",
		},
	}))
	if err != nil {
		res.Message = "SayHi call SayHello!!"
	} else {
		res.Message = resp.Message
		log.Info("SayHi call SayHello ....", c.Bizid(), " ", resp.String())
	}

	resp1, err := hello.PostExample(c, &foo)
	if err != nil {
		res.Message = "SayHi call PostExample FAIL!!"
	} else {
		log.Info("SayHi call PostExample...", c.Bizid(), " ", resp1.String())
	}

	resp2, err := hello.PostFormExample(c, &foo, frame.From(map[string]string{
		"B": "B",
	}))
	if err != nil {
		res.Message = "SayHi call PostFormExample FAIL!!"
	} else {
		log.Info("SayHi call PostFormExample...", c.Bizid(), " ", resp2.String())
	}

	log.Info("SayHi exec finish....", c.Bizid())
	return c.JSON(resp)
}

// Stop Stop
func (id *Hi) Stop(s os.Signal) bool {
	log.Infof("hi graceful exit.")
	return true
}
