package methods

import (
	"atgc/src/types"
)

func (m *Method) DotProduct(ctx types.Context, body types.MethodRequestBody) [][]int {
	// Implementation for dot product
	seq1 := body.Sequence1
	seq2 := body.Sequence2
	arSeq1, arSeq2 := m.processSequenceToArray(seq1, seq2)
	return m.convertToDotProductResult(arSeq1, arSeq2)
}

func (m *Method) convertToDotProductResult(
	sequence1, sequence2 []string,
) [][]int {
	resultMatix := make([][]int, len(sequence1)+1)
	for i, v := range sequence1 {
		for j, w := range sequence2 {
			tba := 0
			if v == w {
				tba += 1
			}
			resultMatix[i][j] = tba
		}
	}
	return resultMatix
}
