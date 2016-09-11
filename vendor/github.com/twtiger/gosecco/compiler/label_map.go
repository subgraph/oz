package compiler

type labelMap struct {
	labelToPosition map[label]int
	positionToLabel map[int][]label
}

func createLabelMap() *labelMap {
	return &labelMap{
		labelToPosition: make(map[label]int),
		positionToLabel: make(map[int][]label),
	}
}

func (l *labelMap) labelsAt(position int) []label {
	return l.positionToLabel[position]
}

func (l *labelMap) addLabelAt(ll label, position int) {
	l.labelToPosition[ll] = position
	l.positionToLabel[position] = append(l.positionToLabel[position], ll)
}

func (l *labelMap) positionOf(ll label) int {
	return l.labelToPosition[ll]
}

func (l *labelMap) allLabels() map[label]int {
	return l.labelToPosition
}

func (l *labelMap) removeLabel(ll label) {
	pos := l.labelToPosition[ll]
	delete(l.labelToPosition, ll)

	res := []label{}
	for _, ll2 := range l.positionToLabel[pos] {
		if ll2 != ll {
			res = append(res, ll2)
		}
	}
	l.positionToLabel[pos] = res
}

func (l *labelMap) hasLabelWithPosition(ll label, position int) bool {
	p, pok := l.labelToPosition[ll]
	return pok && p == position
}

func (l *labelMap) shiftLabels(from int, incr int) {
	labels := make(map[label]int, 0)
	positionsToLabel := make(map[int][]label)

	for k, v := range l.labelToPosition {
		if v > from {
			v += incr
		}
		labels[k] = v
		positionsToLabel[v] = append(positionsToLabel[v], k)
	}

	l.labelToPosition = labels
	l.positionToLabel = positionsToLabel
}

func labelMapFrom(m map[label]int) *labelMap {
	lm := createLabelMap()
	for l, vs := range m {
		lm.addLabelAt(l, vs)
	}
	return lm
}
