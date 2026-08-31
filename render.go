package artgo

import (
	"errors"
	"mime"
	"reflect"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

var (
	ContentTypeTextPlain = "text/plain; charset=utf-8"
	ContentTypeJson      = "application/json"
	ContentTypeHtml      = "text/html; charset=utf-8"
	ContentTypeProtoBuf  = "application/x-protobuf"
)

type Render interface {
	Render(ctx *Context, code int, in interface{}) error
}

var (
	RenderJson     = renderJson{}
	RenderProtoBuf = renderProtoBuf{}
	RenderJsonPB   = renderJsonPB{}
)

type renderJson struct {
}

func (renderJson) Render(ctx *Context, code int, in interface{}) error {
	bs, err := JSON.Marshal(in)
	if err != nil {
		return err
	}
	ctx.SetHeader("Content-Type", ContentTypeJson)
	ctx.Data(code, bs)
	return nil
}

type renderProtoBuf struct {
}

func (renderProtoBuf) Render(ctx *Context, code int, in interface{}) error {
	if requestIsJSON(ctx) {
		return RenderJsonPB.Render(ctx, code, in)
	}
	message, err := protobufMessage(in)
	if err != nil {
		return err
	}
	bs, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	ctx.SetHeader("Content-Type", ContentTypeProtoBuf)
	ctx.Data(code, bs)
	return nil
}

type renderJsonPB struct {
}

func (renderJsonPB) Render(ctx *Context, code int, in interface{}) error {
	message, err := protobufMessage(in)
	if err != nil {
		return err
	}
	bs, err := protojson.Marshal(message)
	if err != nil {
		return err
	}
	ctx.SetHeader("Content-Type", ContentTypeJson)
	ctx.Data(code, []byte(bs))
	return nil
}

func requestIsJSON(ctx *Context) bool {
	return requestMediaType(ctx) == ContentTypeJson
}

func requestMediaType(ctx *Context) string {
	if ctx == nil || ctx.Req == nil {
		return ""
	}
	contentType := ctx.Req.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	return strings.ToLower(mediaType)
}

func protobufMessage(value interface{}) (proto.Message, error) {
	if value == nil {
		return nil, errors.New("in must be a non-nil protobuf message")
	}
	reflected := reflect.ValueOf(value)
	if (reflected.Kind() == reflect.Ptr || reflected.Kind() == reflect.Interface) && reflected.IsNil() {
		return nil, errors.New("in must be a non-nil protobuf message")
	}
	switch message := value.(type) {
	case proto.Message:
		return message, nil
	case protoadapt.MessageV1:
		return protoadapt.MessageV2Of(message), nil
	default:
		return nil, errors.New("in must be a protobuf message")
	}
}
