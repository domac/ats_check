package web

import (
	"github.com/domac/ats_check/app"
)

//cache api handler
type VerHandler struct {
}

func NewVerHandler(f func(ctx *Context)) BaseHandler {
	return BaseHandler{
		Handle: f,
	}
}

func (self *VerHandler) Version(ctx *Context) {
	ctx.W.Write([]byte(app.Version))
}
