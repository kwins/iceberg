package irpc

import "testing"
import "strconv"

func TestQuote(t *testing.T) {
	t.Log(strconv.Quote(`aaaaaaa
		`))
}
