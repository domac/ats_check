package web

import (
	"github.com/domac/ats_check/app"
)

//cache api handler
type ATSHandler struct {
}

func NewATSHandler(f func(ctx *Context)) BaseHandler {
	return BaseHandler{
		Handle: f,
	}
}

func (self *ATSHandler) Parents(ctx *Context) {
	content := ""
	for k, _ := range app.CURRENT_PARENTS {
		content += k + "\n"
	}
	ctx.W.Write([]byte(content))
}
