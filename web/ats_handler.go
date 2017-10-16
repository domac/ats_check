package web

//cache api handler
type ATSHandler struct {
}

func NewATSHandler(f func(ctx *Context)) BaseHandler {
	return BaseHandler{
		Handle: f,
	}
}

func (self *ATSHandler) Parents(ctx *Context) {
	ctx.W.Write([]byte("hello parent"))
}
