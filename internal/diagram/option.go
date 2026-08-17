package diagram

// Option turns on a rendering the picture leaves out by default. Only the two
// picture formats take options: mermaid and ASCII draw no card, so their
// signatures state that rather than accepting an option they would ignore.
type Option func(*renderOptions)

type renderOptions struct {
	specs bool
}

// WithSpecs draws each spec-stating slice's scenarios as a Given-When-Then card
// in a band below the lowest lane. A model whose slices state no spec renders
// exactly as it does without it.
func WithSpecs() Option {
	return func(o *renderOptions) {
		o.specs = true
	}
}

func resolveOptions(opts []Option) renderOptions {
	var resolved renderOptions
	for _, opt := range opts {
		opt(&resolved)
	}

	return resolved
}
