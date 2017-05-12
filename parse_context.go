package pbparser

// The parsing context. We need to pass this around during parsing
// to distinguish between various cases when the same code gets executed
// recursively as well as to pass in objects to hang nested objects to.
type parseCtx struct {
	obj     interface{}
	ctxType ctxType
}

// Type of context
type ctxType int

// The various context types
const (
	fileCtx ctxType = iota
	msgCtx
	enumCtx
	rpcCtx
	extendCtx
	serviceCtx
)

// does this ctx permit package support?
func (pc parseCtx) permitsPackage() bool {
	return pc.ctxType == fileCtx
}

// does this ctx permit syntax support?
func (pc parseCtx) permitsSyntax() bool {
	return pc.ctxType == fileCtx
}

// does this ctx permit import support?
func (pc parseCtx) permitsImport() bool {
	return pc.ctxType == fileCtx
}

// does this ctx permit field support?
func (pc parseCtx) permitsField() bool {
	return pc.ctxType == msgCtx || pc.ctxType == extendCtx
}

// does this ctx permit extensions support?
func (pc parseCtx) permitsExtensions() bool {
	return pc.ctxType != fileCtx
}

// does this ctx permit rpc support?
func (pc parseCtx) permitsRPC() bool {
	return pc.ctxType == serviceCtx
}

// PermitsOneOf does this ctx permit OneOf support?
func (pc parseCtx) permitsOneOf() bool {
	return pc.ctxType == msgCtx
}
