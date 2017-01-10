package simplifier

import "github.com/twtiger/gosecco/tree"

type combiner func([]tree.Expression) tree.Expression

func combineAsOrs(parts []tree.Expression) tree.Expression {
	if len(parts) == 1 {
		return parts[0]
	}
	return tree.Or{Left: parts[0], Right: combineAsOrs(parts[1:])}
}

func combineAsAnds(parts []tree.Expression) tree.Expression {
	if len(parts) == 1 {
		return parts[0]
	}
	return tree.And{Left: parts[0], Right: combineAsAnds(parts[1:])}
}

// AcceptInclusion implements Visitor
func (s *inclusionRemoverSimplifier) AcceptInclusion(a tree.Inclusion) {
	l := s.Transform(a.Left)
	op := tree.EQL
	combiner := combineAsOrs

	if !a.Positive {
		op = tree.NEQL
		combiner = combineAsAnds
	}

	result := make([]tree.Expression, len(a.Rights))
	for ix, v := range a.Rights {
		result[ix] = tree.Comparison{Op: op, Left: l, Right: s.Transform(v)}
	}

	s.Result = combiner(result)
}

// inclusionRemoverSimplifier removes inclusion statements and replaces them with the equivalent simpler version of comparisons composed with ORs/ANDs
type inclusionRemoverSimplifier struct {
	tree.EmptyTransformer
}

func createInclusionRemoverSimplifier() tree.Transformer {
	s := &inclusionRemoverSimplifier{}
	s.RealSelf = s
	return s
}
