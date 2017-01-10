package simplifier

import "github.com/twtiger/gosecco/tree"

// AcceptInclusion implements Visitor
func (s *inclusionSimplifier) AcceptInclusion(a tree.Inclusion) {
	l := s.Transform(a.Left)
	pl, pok := potentialExtractValue(l)

	result := make([]tree.Numeric, len(a.Rights))
	resultVals := make([]uint64, len(a.Rights))
	resultOks := make([]bool, len(a.Rights))
	for ix, v := range a.Rights {
		result[ix] = s.Transform(v)
		resultVals[ix], resultOks[ix] = potentialExtractValue(result[ix])
	}

	if pok {
		newResults := []tree.Numeric{}
		for ix, v := range result {
			if resultOks[ix] {
				if resultVals[ix] == pl {
					s.Result = tree.BooleanLiteral{a.Positive}
					return
				}
			} else {
				switch v.(type) {
				case tree.NumericLiteral:
					// Don't append value to the list because it is not equal to the left value
					break
				default:
					newResults = append(newResults, v)
				}
			}
		}
		if len(newResults) == 0 {
			s.Result = tree.BooleanLiteral{!a.Positive}
		} else if a.Positive == true && len(newResults) == 1 {
			s.Result = tree.Comparison{Op: tree.EQL, Left: l, Right: newResults[0]}
		} else {
			s.Result = tree.Inclusion{Positive: a.Positive, Left: l, Rights: newResults}
		}
	} else {
		s.Result = tree.Inclusion{Positive: a.Positive, Left: l, Rights: result}
	}
}

// inclusionSimplifier simplifies inclusion expressions
type inclusionSimplifier struct {
	tree.EmptyTransformer
}

func createInclusionSimplifier() tree.Transformer {
	s := &inclusionSimplifier{}
	s.RealSelf = s
	return s
}
