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
	oneOfCtx
	enumCtx
	rpcCtx
	extendCtx
	serviceCtx
)

var ctxTypeToStringMap = [...]string{
	fileCtx:    "file",
	msgCtx:     "message",
	oneOfCtx:   "oneof",
	enumCtx:    "enum",
	rpcCtx:     "rpc",
	extendCtx:  "extend",
	serviceCtx: "service",
}

// String returns the human-readable name of the parse context type.
func (pc parseCtx) String() string {
	return ctxTypeToStringMap[pc.ctxType]
}

// permitsPackage reports whether this context permits package declarations.
func (pc parseCtx) permitsPackage() bool {
	return pc.ctxType == fileCtx
}

// permitsSyntax reports whether this context permits syntax declarations.
func (pc parseCtx) permitsSyntax() bool {
	return pc.ctxType == fileCtx
}

// permitsEdition reports whether this context permits edition declarations.
func (pc parseCtx) permitsEdition() bool {
	return pc.ctxType == fileCtx
}

// permitsImport reports whether this context permits import declarations.
func (pc parseCtx) permitsImport() bool {
	return pc.ctxType == fileCtx
}

// permitsField reports whether this context permits field declarations.
func (pc parseCtx) permitsField() bool {
	return pc.ctxType == msgCtx || pc.ctxType == oneOfCtx || pc.ctxType == extendCtx
}

// permitsOption reports whether this context permits option declarations.
func (pc parseCtx) permitsOption() bool {
	return pc.ctxType == fileCtx ||
		pc.ctxType == msgCtx ||
		pc.ctxType == oneOfCtx ||
		pc.ctxType == enumCtx ||
		pc.ctxType == serviceCtx ||
		pc.ctxType == rpcCtx
}

// permitsExtensions reports whether this context permits extensions declarations.
func (pc parseCtx) permitsExtensions() bool {
	return pc.ctxType == msgCtx
}

// permitsExtend reports whether this context permits extend declarations.
func (pc parseCtx) permitsExtend() bool {
	return pc.ctxType == fileCtx || pc.ctxType == msgCtx
}

// permitsReserved reports whether this context permits reserved declarations.
func (pc parseCtx) permitsReserved() bool {
	return pc.ctxType == msgCtx || pc.ctxType == enumCtx
}

// permitsRPC reports whether this context permits RPC declarations.
func (pc parseCtx) permitsRPC() bool {
	return pc.ctxType == serviceCtx
}

// permitsOneOf reports whether this context permits oneof declarations.
func (pc parseCtx) permitsOneOf() bool {
	return pc.ctxType == msgCtx
}

// permitsEnum reports whether this context permits enum declarations.
func (pc parseCtx) permitsEnum() bool {
	return pc.ctxType == fileCtx || pc.ctxType == msgCtx
}

// permitsMsg reports whether this context permits message declarations.
func (pc parseCtx) permitsMsg() bool {
	return pc.ctxType == fileCtx || pc.ctxType == msgCtx
}
