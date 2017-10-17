package web

import (
	"github.com/domac/ats_check/app"
)

//Apache Traffic Server Handler
type ATSHandler struct {
	applicationContext *app.App
}

func NewATSHandler(f func(ctx *Context)) BaseHandler {
	return BaseHandler{
		Handle: f,
	}
}

func (self *ATSHandler) Parents(ctx *Context) {
	content := ""
	for _, k := range self.applicationContext.GetParentsHosts() {
		content += k + "\n"
	}
	ctx.W.Write([]byte(content))
}
