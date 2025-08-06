// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"
	"runtime"
	"strconv"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/process"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
)

// Stacktrace show admin monitor goroutines page
func Stacktrace(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.monitor")
	ctx.Data["PageIsAdminMonitorStacktrace"] = true

	ctx.Data["GoroutineCount"] = runtime.NumGoroutine()

	show := ctx.FormString("show")
	ctx.Data["ShowGoroutineList"] = show
	// by default, do not do anything which might cause server errors, to avoid unnecessary 500 pages.
	// this page is the entrance of the chance to collect diagnosis report.
	auditParams := make(map[string]string)
	if show != "" {
		auditParams["tab"] = show
		showNoSystem := show == "process"
		processStacks, processCount, _, err := process.GetManager().ProcessStacktraces(false, showNoSystem)
		if err != nil {
			ctx.ServerError("GoroutineStacktrace", err)
			auditParams["error"] = "Error has occurred while getting goroutine stacktrace"
			audit.CreateAndSendEvent(audit.MonitoringStacktraceOpen, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}

		ctx.Data["ProcessStacks"] = processStacks
		ctx.Data["ProcessCount"] = processCount
	}

	audit.CreateAndSendEvent(audit.MonitoringStacktraceOpen, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.HTML(http.StatusOK, tplStacktrace)
}

// StacktraceCancel cancels a process
func StacktraceCancel(ctx *context.Context) {
	pid := ctx.Params("pid")
	process.GetManager().Cancel(process.IDType(pid))
	audit.CreateAndSendEvent(audit.StacktraceProcessCancel, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, nil)
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": setting.AppSubURL + "/admin/monitor/stacktrace",
	})
}
