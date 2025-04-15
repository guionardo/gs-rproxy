package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type tVal string

func (v tVal) String() string {
	return string(v)
}

func Test_getDiff(t *testing.T) {
	old := map[string]tVal{
		"0": tVal("V0"),
		"1": tVal("V1"),
		"2": tVal("V2"),
	}
	new := map[string]tVal{
		"1": tVal("V1"),
		"2": tVal("V2.1"),
		"3": tVal("V3"),
	}
	diffs := getDiff(old, new)
	assert.Equal(t, []string{
		"# V2 -> V2.1",
		"+ V3",
		"- V0",
	}, diffs)
}
