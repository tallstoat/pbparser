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

// the veritable tostring() for java lovers...
func (pc parseCtx) String() string {
	return ctxTypeToStringMap[pc.ctxType]
}

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
	return pc.ctxType == msgCtx || pc.ctxType == oneOfCtx || pc.ctxType == extendCtx
}

// does this ctx permit option?
func (pc parseCtx) permitsOption() bool {
	return pc.ctxType == fileCtx ||
		pc.ctxType == msgCtx ||
		pc.ctxType == oneOfCtx ||
		pc.ctxType == enumCtx ||
		pc.ctxType == serviceCtx ||
		pc.ctxType == rpcCtx
}

// does this ctx permit extensions support?
func (pc parseCtx) permitsExtensions() bool {
	return pc.ctxType == msgCtx
}

// does this ctx permit extend declarations?
func (pc parseCtx) permitsExtend() bool {
	return pc.ctxType == fileCtx || pc.ctxType == msgCtx
}

// does this ctx permit reserved keyword support?
func (pc parseCtx) permitsReserved() bool {
	return pc.ctxType == msgCtx
}

// does this ctx permit rpc support?
func (pc parseCtx) permitsRPC() bool {
	return pc.ctxType == serviceCtx
}

// does this ctx permit OneOf support?
func (pc parseCtx) permitsOneOf() bool {
	return pc.ctxType == msgCtx
}

// does this ctx permit enum support?
func (pc parseCtx) permitsEnum() bool {
	return pc.ctxType == fileCtx || pc.ctxType == msgCtx
}

// does this ctx permit msg support?
func (pc parseCtx) permitsMsg() bool {
	return pc.ctxType == fileCtx || pc.ctxType == msgCtx
}
