package core

import "callcenter-middleware/internal/domain"

// transferExternal dials an outside number and bridges the caller to it.
func (c *Core) transferExternal(call *liveCall, number string) {
	if NormalizePhone(number) == "" {
		c.execute(call, c.defaultAction(call.TenantID))
		return
	}
	call.State = domain.CallDialing
	c.changed(call)
	c.startMOH(call)
	c.originateRemote(call, roleExternal, number)
}

func (c *Core) onExternalGone(call *liveCall, l *leg) {
	if l.answered {
		c.finish(call, "external")
		return
	}
	call.extChan = ""
	c.timeline(call, "external_failed", "")
	if call.queue != nil {
		c.enqueue(call, call.queue)
		return
	}
	c.execute(call, c.defaultAction(call.TenantID))
}
