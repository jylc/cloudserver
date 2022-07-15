package fsctx

type key int

const (
	GinCtx key = iota
	PathCtx
	FileModelCtx
	LimitParentCtx
)
