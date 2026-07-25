package test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func RequireEqual(t *testing.T, expected, actual any, opts ...cmp.Option) {
	t.Helper()
	if cmp.Equal(expected, actual, opts...) {
		return
	}
	diff := cmp.Diff(expected, actual, opts...)
	require.Fail(t, fmt.Sprintf("Not equal: \n"+
		"expected: %+v\n"+
		"actual  : %+v\n"+
		"diff    : %s", expected, actual, diff))
}
