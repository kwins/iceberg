package main

import (
	"testing"
)

func TestSplitPathAndMethod(t *testing.T) {
	srvPath, srvMethod, srvRawQuery := splitPathAndMethod("/services/v1/orderpay/order/CreateOrderWithPay?name=quinn&id=1001")
	t.Log("srvPath:", srvPath, " srvMethod:", srvMethod, " srvRawQuery:", srvRawQuery)
}
