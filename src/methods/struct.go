package methods

import "atgc/src/types"

type MatchingMethods interface {
	DyanmicProgrammingMatch(ctx types.Context)
	DotProductMethod(ctx types.Context)
}

type Method struct {
	Name      string
	StartTime string
	EndTime   string
}
