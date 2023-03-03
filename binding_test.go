package artgo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_bindMap(t *testing.T) {
	m := map[string]string{
		"s_1": "hello",
		"i":   "5",
		"b":   "1",
		"f":   "2.56",
	}
	var form struct {
		S1 *string `json:"s_1"`
		I  *int8
		B  bool
		F  float64
	}
	assert.Nil(t, bindMap(m, &form))
	t.Log(form, *form.S1, *form.I)
}

func Test_bindValidate(t *testing.T) {
	EnableValidate()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"s":"hello","i":20,"b": true,"f": 2.56}`))
	type form struct {
		S *string
		I *int8 `validate:"lt=10"`
		B bool
		F float64
	}
	ctx := &Context{Req: request}
	var f form
	t.Log(BindJson.Bind(ctx, &f))

}
