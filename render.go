package artgo

type Render interface {
	Render(ctx *Context, code int, in interface{}) error
}
