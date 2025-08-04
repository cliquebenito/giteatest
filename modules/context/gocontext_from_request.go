package context

import (
	gocontext "context"
)

func GetGoContextFromRequestOrDefault(ctx *Context) gocontext.Context {
	if ctx == nil || ctx.Req == nil {
		return gocontext.Background()
	}

	return ctx.Req.Context()
}
