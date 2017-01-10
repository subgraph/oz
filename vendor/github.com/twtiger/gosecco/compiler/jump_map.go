package compiler

type jumpMap struct {
	labelToPosition map[label][]int
	positionToLabel map[int]label
}

func createJumpMap() *jumpMap {
	return &jumpMap{
		labelToPosition: make(map[label][]int),
		positionToLabel: make(map[int]label),
	}
}

func (j *jumpMap) registerJump(l label, i int) {
	j.labelToPosition[l] = append(j.labelToPosition[l], i)
	// If the jump maps are used correctly, this should never overwrite an index
	j.positionToLabel[i] = l
}

func (j *jumpMap) allJumpsTo(l label) []int {
	return j.labelToPosition[l]
}

func (j *jumpMap) jumpTargetOf(pos int) label {
	return j.positionToLabel[pos]
}

func (j *jumpMap) removeJumpTarget(pos int) {
	l, ok := j.positionToLabel[pos]
	if ok {
		delete(j.positionToLabel, pos)

		newPos := []int{}
		for _, x := range j.labelToPosition[l] {
			if x != pos {
				newPos = append(newPos, x)
			}
		}
		j.labelToPosition[l] = newPos
	}
}

func (j *jumpMap) hasJumpFrom(i int) bool {
	_, ok := j.positionToLabel[i]
	return ok
}

func (j *jumpMap) redirectJump(old, new label) {
	poses := j.labelToPosition[old]
	delete(j.labelToPosition, old)

	for _, pos := range poses {
		j.positionToLabel[pos] = new
	}

	j.labelToPosition[new] = append(j.labelToPosition[new], poses...)
}

func (j *jumpMap) shift(from int, incr int) {
	newLabelToPosition := make(map[label][]int)
	newPositionToLabel := make(map[int]label)

	for l, v := range j.labelToPosition {
		for _, ix := range v {
			if ix > from {
				ix += incr
			}

			newLabelToPosition[l] = append(newLabelToPosition[l], ix)
			newPositionToLabel[ix] = l
		}
	}

	j.labelToPosition = newLabelToPosition
	j.positionToLabel = newPositionToLabel
}

func jumpMapFrom(m map[label][]int) *jumpMap {
	jm := createJumpMap()
	for l, vs := range m {
		for _, v := range vs {
			jm.registerJump(l, v)
		}
	}
	return jm
}

// hasJumpTarget will return true if the jump map has a target that is the fromIndex
func (j *jumpMap) hasJumpTarget(c *compilerContext, fromIndex, potentialTarget int) bool {
	l, ok := j.positionToLabel[fromIndex]
	return ok && c.labels.hasLabelWithPosition(l, potentialTarget)
}

// countJumpsFrom will return the number of jumps this jump map has from the expectedFrom to the jumpTarget
func (j *jumpMap) countJumpsFrom(c *compilerContext, jumpTarget, expectedFrom int) int {
	count := 0

	for _, ll := range c.labels.labelsAt(jumpTarget) {
		for _, xx := range j.labelToPosition[ll] {
			if xx == expectedFrom {
				count++
			}
		}
	}

	return count
}

// countJumpsFromAny will count how many jumps in this map jump to the specific target
func (j *jumpMap) countJumpsFromAny(c *compilerContext, jumpTarget int) int {
	count := 0

	for _, ll := range c.labels.labelsAt(jumpTarget) {
		if _, exist := j.labelToPosition[ll]; exist {
			count++
		}
	}

	return count
}

// jumpSizeIsOversized will return true if the given unconditional jump is oversized
func (j *jumpMap) jumpSizeIsOversized(c *compilerContext, fromJump int) bool {
	l := j.positionToLabel[fromJump]
	to := c.labels.positionOf(l)
	return (to-fromJump)-1 > c.maxJumpSize
}
