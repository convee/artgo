package artgo

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"strings"
)

var (
	ContentTypeTextPlain = "text/plain; charset=utf-8"
	ContentTypeJson      = "application/json"
	ContentTypeProtoBuf  = "application/x-protobuf"
)

type Render interface {
	Render(ctx *Context, code int, in interface{}) error
}

var (
	RenderJson     = renderJson{}
	RenderProtoBuf = renderProtoBuf{}
	RenderJsonPB   = renderJsonPB{}

	jsonPBMarshal = &jsonpb.Marshaler{}
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
	if strings.HasPrefix(ctx.Req.Header.Get("Content-Type"), ContentTypeJson) {
		return RenderJsonPB.Render(ctx, code, in)
	}
	bs, err := proto.Marshal(in.(proto.Message))
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
	bs, err := jsonPBMarshal.MarshalToString(in.(proto.Message))
	if err != nil {
		return err
	}
	ctx.SetHeader("Content-Type", ContentTypeProtoBuf)
	ctx.Data(code, []byte(bs))
	return nil
}
