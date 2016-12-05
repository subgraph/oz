package network

import (
	"testing"
)

func TestParseRanges(t *testing.T) {
	data := []struct {
		r    string
		ones int
		ip   string
	}{{"1.2.3.4/12", 12, "1.0.0.0"},
		{"192.168.1.1/16", 16, "192.168.0.0"}}

	for _, d := range data {
		rs := parseRanges(d.r)
		if len(rs) != 1 {
			t.Errorf("expecting 1 network got %d", len(rs))
		}
		n := rs[0]
		ones, _ := n.Mask.Size()
		if d.ones != ones {
			t.Errorf("expecting %d bit mask for %s and got %d", d.ones, d.r, ones)
		}
		if d.ip != n.IP.String() {
			t.Errorf("expecting network base of %s and got %s", d.ip, n.IP.String())
		}
	}
	assertPanic(t, "parseRanges(\"1.2.3.4\")", func() {
		parseRanges("1.2.3.4")
	})
}
