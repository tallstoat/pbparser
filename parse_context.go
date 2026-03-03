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

// keywordPermissions maps each protobuf keyword to the set of context types
// that permit it. Keywords not in the map are always permitted.
var keywordPermissions = map[string]map[ctxType]bool{
	"package":    {fileCtx: true},
	"syntax":     {fileCtx: true},
	"edition":    {fileCtx: true},
	"import":     {fileCtx: true},
	"option":     {fileCtx: true, msgCtx: true, oneOfCtx: true, enumCtx: true, serviceCtx: true, rpcCtx: true},
	"message":    {fileCtx: true, msgCtx: true},
	"enum":       {fileCtx: true, msgCtx: true},
	"extend":     {fileCtx: true, msgCtx: true},
	"service":    {fileCtx: true, msgCtx: true, oneOfCtx: true, enumCtx: true, rpcCtx: true, extendCtx: true, serviceCtx: true},
	"rpc":        {serviceCtx: true},
	"oneof":      {msgCtx: true},
	"extensions": {msgCtx: true},
	"reserved":   {msgCtx: true, enumCtx: true},
}

// permits reports whether this context permits a given keyword declaration.
func (pc parseCtx) permits(keyword string) bool {
	allowed, ok := keywordPermissions[keyword]
	if !ok {
		return true
	}
	return allowed[pc.ctxType]
}

// permitsField reports whether this context permits field declarations.
func (pc parseCtx) permitsField() bool {
	return pc.ctxType == msgCtx || pc.ctxType == oneOfCtx || pc.ctxType == extendCtx
}
