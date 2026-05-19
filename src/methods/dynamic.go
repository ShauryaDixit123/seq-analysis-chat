package methods

import (
	"atgc/src/types"
	"strings"
)

func (m *Method) DyanmicProgrammingMatch(ctx types.Context, body types.MethodRequestBody) [][]int {
	// Implementation for dynamic programming matching
	seq1 := body.Sequence1
	seq2 := body.Sequence2
	arSeq1, arSeq2 := m.processSequenceToArray(seq1, seq2)
	return m.convertToDynamicProgrammingMatchResult(arSeq1, arSeq2)
}

func (m *Method) convertToDynamicProgrammingMatchResult(
	sequence1, sequence2 []string,
) [][]int {
	resultMatix := make([][]int, len(sequence1)+1)
	for i := range resultMatix {
		resultMatix[i] = make([]int, len(sequence2)+1)
	}
	for i, v := range sequence1 {
		for j, w := range sequence2 {
			// Perform matching logic here
			if v == w {
				tba := 1
				if i > 0 && j > 0 {
					tba += resultMatix[i-1][j]
				}
				resultMatix[i][j] = tba
				// Match found, update the method's state or perform necessary actions
				// This is a placeholder for the actual dynamic programming logic
				// You can implement the logic to fill a DP table or any other structure as needed
			} else {
				// No match, update the method's state or perform necessary actions
				max := 0
				if i > 0 && j > 0 {
					if resultMatix[i-1][j] > max {
						max = resultMatix[i-1][j]
					}
					if resultMatix[i][j-1] > max {
						max = resultMatix[i][j-1]
					}
					if resultMatix[i-1][j-1] > max {
						max = resultMatix[i-1][j-1]
					}
				}
				resultMatix[i][j] = max
			}
		}
	}
	return resultMatix
}

func (m *Method) processSequenceToArray(seq1, seq2 string) ([]string, []string) {
	arSeq1 := make([]string, len(seq1))
	arSeq2 := make([]string, len(seq2))
	for i, v := range strings.Split(seq1, "") {
		arSeq1[i] = v
	}
	for i, v := range strings.Split(seq2, "") {
		arSeq2[i] = v
	}
	return arSeq1, arSeq2
}

func NewMatchMethod(name string,
	startTime string,
	endTime string,
) *Method {
	return &Method{
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
	}
}
