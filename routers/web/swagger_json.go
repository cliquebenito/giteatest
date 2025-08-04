// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/log"
)

// tplSwaggerV1Json swagger v1 json template
const tplSwaggerV1Json base.TplName = "swagger/v1_json"
const tplSwaggerV2Json base.TplName = "swagger/v2_json"
const tplSwaggerV3Json base.TplName = "swagger/v3_json"

// SwaggerV1Json render swagger v1 json
func SwaggerV1Json(ctx *context.Context) {
	t, err := ctx.Render.TemplateLookup(string(tplSwaggerV1Json))
	if err != nil {
		log.Error("Error has occurred while find template")
		ctx.ServerError("unable to find template", err)
		return
	}
	ctx.Resp.Header().Set("Content-Type", "application/json")
	if err = t.Execute(ctx.Resp, ctx.Data); err != nil {
		ctx.ServerError("unable to execute template", err)
	}
}

// SwaggerV2Json render swagger v2 json
func SwaggerV2Json(ctx *context.Context) {
	t, err := ctx.Render.TemplateLookup(string(tplSwaggerV2Json))
	if err != nil {
		log.Error("Error has occurred while find template")
		ctx.ServerError("unable to find template", err)
		return
	}
	ctx.Resp.Header().Set("Content-Type", "application/json")
	if err = t.Execute(ctx.Resp, ctx.Data); err != nil {
		ctx.ServerError("unable to execute template", err)
	}
}

// SwaggerV3Json render swagger v3 json
func SwaggerV3Json(ctx *context.Context) {
	t, err := ctx.Render.TemplateLookup(string(tplSwaggerV3Json))
	if err != nil {
		log.Error("Error has occurred while find template")
		ctx.ServerError("unable to find template", err)
		return
	}
	ctx.Resp.Header().Set("Content-Type", "application/json")
	if err = t.Execute(ctx.Resp, ctx.Data); err != nil {
		ctx.ServerError("unable to execute template", err)
	}
}
