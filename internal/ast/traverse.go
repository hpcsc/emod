package ast

import (
	"cmp"
	"slices"
)

// Compare orders positions by filename, then line, then column.
func (p Position) Compare(o Position) int {
	if c := cmp.Compare(p.Filename, o.Filename); c != 0 {
		return c
	}
	if c := cmp.Compare(p.Line, o.Line); c != 0 {
		return c
	}
	return cmp.Compare(p.Column, o.Column)
}

// SliceRef pairs a slice with where it was declared: Aggregate is nil for the
// slices a `mode dcb` context declares directly.
type SliceRef struct {
	Slice     *Slice
	Context   *Context
	Aggregate *Aggregate
}

// SliceRefs returns every slice the context declares — directly or via its
// aggregates — in source order. Direct slices and aggregates live in separate
// collections, so sorting by position is what recovers how they were written.
func (c *Context) SliceRefs() []SliceRef {
	if c == nil {
		return nil
	}

	refs := make([]SliceRef, 0, len(c.Slices))
	for _, s := range c.Slices {
		refs = append(refs, SliceRef{Slice: s, Context: c})
	}
	for _, agg := range c.Aggregates {
		for _, s := range agg.Slices {
			refs = append(refs, SliceRef{Slice: s, Context: c, Aggregate: agg})
		}
	}

	slices.SortStableFunc(refs, func(a, b SliceRef) int {
		return a.Slice.NamePos.Compare(b.Slice.NamePos)
	})

	return refs
}

// AllSlices returns every slice the context declares — directly or via its
// aggregates — in source order.
func (c *Context) AllSlices() []*Slice {
	refs := c.SliceRefs()
	all := make([]*Slice, len(refs))
	for i, ref := range refs {
		all[i] = ref.Slice
	}

	return all
}

// SliceRefs returns every slice in the model in source order, each paired with
// the context and (when present) aggregate that declares it.
func (m *Model) SliceRefs() []SliceRef {
	if m == nil {
		return nil
	}

	var refs []SliceRef
	for _, ctx := range m.Contexts {
		refs = append(refs, ctx.SliceRefs()...)
	}

	return refs
}

// AllSlices returns every slice in the model in source order.
func (m *Model) AllSlices() []*Slice {
	refs := m.SliceRefs()
	all := make([]*Slice, len(refs))
	for i, ref := range refs {
		all[i] = ref.Slice
	}

	return all
}
