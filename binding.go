package artgo

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Binding interface {
	Bind(ctx *Context, out interface{}) error
}

var (
	BindJson                   = bindJson{}
	BindProtoBuf               = bindProtoBuf{}
	BindJsonPB                 = bindJsonPB{}
	BindQuery                  = bindQuery{}
	BindForm                   = bindForm{}
	EnableBindProtoBufAsJsonPB = false
)

type bindJson struct {
}

func (bindJson) Bind(ctx *Context, out interface{}) error {
	body, err := ctx.postBody()
	if err != nil {
		return err
	}

	err = JSON.Unmarshal(body, out)
	if err == nil && structValidator.enabled {
		err = structValidator.ValidateStruct(out)
	}
	return err
}

type bindProtoBuf struct {
}

func (bindProtoBuf) Bind(ctx *Context, out interface{}) error {
	if EnableBindProtoBufAsJsonPB || requestIsJSON(ctx) {
		return BindJsonPB.Bind(ctx, out)
	}
	message, err := protobufMessage(out)
	if err != nil {
		return err
	}
	body, err := ctx.postBody()
	if err != nil {
		return err
	}
	return proto.Unmarshal(body, message)
}

type bindJsonPB struct {
}

func (bindJsonPB) Bind(ctx *Context, out interface{}) error {
	message, err := protobufMessage(out)
	if err != nil {
		return err
	}
	body, err := ctx.postBody()
	if err != nil {
		return err
	}
	return protojson.Unmarshal(body, message)
}

type bindQuery struct {
}

func (b bindQuery) Bind(ctx *Context, out interface{}) error {
	m := map[string]string{}
	query := ctx.Req.URL.Query()
	for key, value := range query {
		m[strings.ToLower(key)] = firstValue(value)
	}
	return bindMap(m, out)
}

type bindForm struct {
}

func (b bindForm) Bind(ctx *Context, out interface{}) error {
	if ctx.Req.MultipartForm == nil {
		if err := ctx.Req.ParseMultipartForm(32 << 20); err != nil && !errors.Is(err, http.ErrNotMultipart) {
			return err
		}
	}
	m := map[string]string{}
	if ctx.Req.MultipartForm != nil {
		for key, vs := range ctx.Req.MultipartForm.Value {
			m[strings.ToLower(key)] = lastValue(vs)
		}
	}
	post := ctx.Req.PostForm
	for key, value := range post {
		m[strings.ToLower(key)] = firstValue(value)
	}
	query := ctx.Req.URL.Query()
	for key, value := range query {
		m[strings.ToLower(key)] = firstValue(value)
	}
	return bindMap(m, out)
}

func firstValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func lastValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

func bindMap(args map[string]string, out interface{}) error {
	ptr := reflect.ValueOf(out)
	if ptr.Kind() != reflect.Ptr {
		return errors.New("out must be struct ptr")
	}
	rv := ptr.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("out must be struct ptr")
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		rf := rv.Field(i)
		if !rf.CanSet() {
			continue
		}
		var name string
		tagParts := strings.Split(rt.Field(i).Tag.Get("json"), ",")
		if tagParts[0] != "" {
			name = tagParts[0]
		} else {
			name = rt.Field(i).Name
		}
		value, ok := args[strings.ToLower(name)]
		if !ok {
			continue
		}
		err := setValueFromString(rf, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func setValueFromString(v reflect.Value, str string) (err error) {
	switch v.Kind() {
	case reflect.Ptr:
		pv := reflect.New(v.Type().Elem())
		err := setValueFromString(pv.Elem(), str)
		if err != nil {
			return err
		}
		v.Set(pv)
	case reflect.String:
		v.SetString(str)
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return err
		}
		v.SetFloat(val)
	case reflect.Bool:
		val, err := strconv.ParseBool(str)
		if err != nil {
			return err
		}
		v.SetBool(val)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(str, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(val)
	default:
		return errors.New("fields must be int,float,bool,string")
	}
	return nil
}
