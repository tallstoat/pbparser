package internal

// ParseContext the parsing context...
type ParseContext struct {
	Obj     interface{}
	CtxType ctxType
}

// the various types of parse contexts...
type ctxType int

const (
	fileCtx ctxType = iota
	msgCtx
	enumCtx
	rpcCtx
	extendCtx
	serviceCtx
)

func (pc ParseContext) permitsPackage() bool {
	return pc.CtxType == fileCtx
}

func (pc ParseContext) permitsSyntax() bool {
	return pc.CtxType == fileCtx
}

func (pc ParseContext) permitsImport() bool {
	return pc.CtxType == fileCtx
}

func (pc ParseContext) permitsField() bool {
	return pc.CtxType == msgCtx || pc.CtxType == extendCtx
}

func (pc ParseContext) permitsExtensions() bool {
	return pc.CtxType != fileCtx
}

func (pc ParseContext) permitsRPC() bool {
	return pc.CtxType == serviceCtx
}

func (pc ParseContext) permitsOneOf() bool {
	return pc.CtxType == msgCtx
}
