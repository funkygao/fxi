package servant

import (
	"github.com/funkygao/fae/servant/gen-go/fun/rpc"
	log "github.com/funkygao/log4go"
)

// Ticket server
func (this *FunServantImpl) IdNext(ctx *rpc.Context,
	flag int16) (r int64, backwards *rpc.TIdTimeBackwards, appErr error) {
	const IDENT = "id.next"

	if appErr = validateContext(ctx); appErr != nil {
		return
	}

	this.stats.inc(IDENT)
	profiler := this.getSession(ctx).startProfiler()

	r, appErr = this.idgen.Next()
	if appErr != nil {
		log.Error("Q=%s %s: clock backwards", IDENT, ctx.String())

		backwards = appErr.(*rpc.TIdTimeBackwards)
		appErr = nil
	}

	profiler.do(IDENT, ctx, "{flag^%d} {r^%d}", flag, r)

	return
}
