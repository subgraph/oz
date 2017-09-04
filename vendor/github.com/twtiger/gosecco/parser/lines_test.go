package parser

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type LinesSuite struct{}

var _ = Suite(&LinesSuite{})

func (s *LinesSuite) Test_lineType_recognizesLineTypes(c *C) {
	c.Check(lineType("DEFAULT_POSITIVE = trace"), Equals, defaultAssignmentLine)
	c.Check(lineType("DEFAULT_POSITIVE=42"), Equals, defaultAssignmentLine)
	c.Check(lineType("DEFAULT_NEGATIVE=kill"), Equals, defaultAssignmentLine)
	c.Check(lineType("DEFAULT_POLICY=kill"), Equals, defaultAssignmentLine)

	c.Check(lineType("foo=42*42"), Equals, assignmentLine)
	c.Check(lineType("bar = 42"), Equals, assignmentLine)
	c.Check(lineType("BAR = 42"), Equals, assignmentLine)
	c.Check(lineType("bar() = 42"), Equals, assignmentLine)
	c.Check(lineType("bar (x, y) = 42"), Equals, assignmentLine)

	c.Check(lineType("#hello world"), Equals, commentLine)
	c.Check(lineType("# something: cool"), Equals, commentLine)
	c.Check(lineType(" # something = else"), Equals, commentLine)

	c.Check(lineType("read: 1"), Equals, ruleLine)
	c.Check(lineType("read[]: return 42"), Equals, ruleLine)
	c.Check(lineType("write[+kill] :return 42"), Equals, ruleLine)
	c.Check(lineType("write[+hello, -foo]: return 42"), Equals, ruleLine)

	c.Check(lineType("hmm"), Equals, unknownLine)
}
