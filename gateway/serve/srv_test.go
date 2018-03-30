package serve

import (
	"testing"
)

func TestAddMethod(t *testing.T) {
	d := NewDiscover()
	d.addMethod("/services/v1/coupon/query/provider/allowed/false", "query")
	d.addMethod("/services/v1/coupon/send/provider/allowed/false", "send")
	t.Log(d.mdtables)
}
