// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"
	"strconv"

	"code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/queue"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
)

func Queues(ctx *context.Context) {
	if !setting.IsProd {
		initTestQueueOnce()
	}
	ctx.Data["Title"] = ctx.Tr("admin.monitor.queue")
	ctx.Data["PageIsAdminMonitorQueue"] = true
	ctx.Data["Queues"] = queue.GetManager().ManagedQueues()
	audit.CreateAndSendEvent(audit.MonitorQueuesPanelOpen, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, nil)
	ctx.HTML(http.StatusOK, tplQueue)
}

// QueueManage shows details for a specific queue
func QueueManage(ctx *context.Context) {
	qid := ctx.ParamsInt64("qid")
	auditParams := map[string]string{
		"queue_id": strconv.FormatInt(qid, 10),
	}
	mq := queue.GetManager().GetManagedQueue(qid)
	if mq == nil {
		ctx.Status(http.StatusNotFound)
		auditParams["error"] = "Queue not found"
		audit.CreateAndSendEvent(audit.MonitorQueueOpen, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}
	ctx.Data["Title"] = ctx.Tr("admin.monitor.queue", mq.GetName())
	ctx.Data["PageIsAdminMonitor"] = true
	ctx.Data["Queue"] = mq
	audit.CreateAndSendEvent(audit.MonitorQueueOpen, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.HTML(http.StatusOK, tplQueueManage)
}

// QueueSet sets the maximum number of workers and other settings for this queue
func QueueSet(ctx *context.Context) {
	qid := ctx.ParamsInt64("qid")
	auditParams := map[string]string{
		"queue_id":  strconv.FormatInt(qid, 10),
		"new_value": ctx.FormString("max-number"),
	}
	mq := queue.GetManager().GetManagedQueue(qid)
	if mq == nil {
		ctx.Status(http.StatusNotFound)
		auditParams["error"] = "Queue not found"
		audit.CreateAndSendEvent(audit.QueueNumberOfWorkersChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	maxNumberStr := ctx.FormString("max-number")

	var err error
	maxNumber := mq.GetWorkerMaxNumber()
	auditParams["old_value"] = strconv.Itoa(maxNumber)

	if len(maxNumberStr) > 0 {
		maxNumber, err = strconv.Atoi(maxNumberStr)
		if err != nil {
			ctx.Flash.Error(ctx.Tr("admin.monitor.queue.settings.maxnumberworkers.error"))
			ctx.Redirect(setting.AppSubURL + "/admin/monitor/queue/" + strconv.FormatInt(qid, 10))
			auditParams["error"] = "Max number of workers must be a number"
			audit.CreateAndSendEvent(audit.QueueNumberOfWorkersChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
			return
		}
		if maxNumber < -1 {
			maxNumber = -1
		}
		auditParams["new_value"] = strconv.Itoa(maxNumber)
	}

	mq.SetWorkerMaxNumber(maxNumber)
	audit.CreateAndSendEvent(audit.QueueNumberOfWorkersChange, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("admin.monitor.queue.settings.changed"))
	ctx.Redirect(setting.AppSubURL + "/admin/monitor/queue/" + strconv.FormatInt(qid, 10))
}

func QueueRemoveAllItems(ctx *context.Context) {
	// Gitea's queue doesn't have transaction support
	// So in rare cases, the queue could be corrupted/out-of-sync
	// Site admin could remove all items from the queue to make it work again
	qid := ctx.ParamsInt64("qid")
	auditParams := map[string]string{
		"queue_id": strconv.FormatInt(qid, 10),
	}
	mq := queue.GetManager().GetManagedQueue(qid)
	if mq == nil {
		ctx.Status(http.StatusNotFound)
		auditParams["error"] = "Queue not found"
		audit.CreateAndSendEvent(audit.QueueAllItemsRemove, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	if err := mq.RemoveAllItems(ctx); err != nil {
		ctx.ServerError("RemoveAllItems", err)
		auditParams["error"] = "Error has occurred while removing all items"
		audit.CreateAndSendEvent(audit.QueueAllItemsRemove, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusFailure, ctx.Req.RemoteAddr, auditParams)
		return
	}

	audit.CreateAndSendEvent(audit.QueueAllItemsRemove, ctx.Doer.Name, strconv.FormatInt(ctx.Doer.ID, 10), audit.StatusSuccess, ctx.Req.RemoteAddr, auditParams)
	ctx.Flash.Success(ctx.Tr("admin.monitor.queue.settings.remove_all_items_done"))
	ctx.Redirect(setting.AppSubURL + "/admin/monitor/queue/" + strconv.FormatInt(qid, 10))
}
