// Package golden — options for Golden test assertions.
package golden

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/stellar-optics/stellar-focus/pkg/focus"
)

// Option configures a Golden instance. It is an alias for Opt.
type Option = Opt

// WithCompact sets the golden file format to Compact JSON (single-line
// .min.json or .compact.json).
//
// When compact mode is enabled, fixtures are written as single-line JSON with
// canonical sorted keys, saving repository disk space for large fixture suites.
func WithCompact(compact bool) Option {
	return func(g *Golden) {
		if compact {
			g.format = Compact
			g.suffix = Compact.DefaultSuffix()
		} else if g.format == Compact {
			g.format = JSON
			g.suffix = JSON.DefaultSuffix()
		}
	}
}

// NormalizerFunc transforms the value found at path in the decoded value
// tree, returning the value to use in its place. path is a slash-separated
// walk of the map keys and array indices leading to value, e.g.
// "operations/0/body/paymentOp/amount". Returning value unchanged leaves
// that node, and the fixture, untouched.
//
// A normalizer is a pure function of (path, value): it must not read or
// mutate any state outside its arguments, since the same registered
// normalizer runs over every node of every fixture a Golden compares,
// including concurrently under AssertAllParallel.
type NormalizerFunc func(path string, value any) any

// WithCustomNormalizer registers a hook run over every node of the decoded
// value tree during golden-file generation and comparison. It is for
// domain-specific transforms a static WithOptions ignore rule cannot
// express: hashing a user id, zeroing an ephemeral salt, or replacing a
// contract address with a stable mock alias before the value is compared or
// written to disk.
//
// Multiple normalizers accumulate rather than replace one another, in
// registration order, and each one sees the value the previous one left
// at that node.
//
// Registering a normalizer changes how this Golden compares values: the
// comparison moves from the stored XDR onto the normalized decoded tree, so
// a difference a normalizer erases no longer fails the assertion.
func WithCustomNormalizer(fn NormalizerFunc) Option {
	return func(g *Golden) {
		g.normalizers = append(g.normalizers, fn)
	}
}

// TrackedIgnoreRule tracks an ignore pattern along with the source location
// where it was registered and how many times it was matched.
type TrackedIgnoreRule struct {
	Pattern  string
	Location string
	Hits     int
}

// TrackIgnore creates a new TrackedIgnoreRule with the given pattern and location.
func TrackIgnore(pattern, location string) *TrackedIgnoreRule {
	return &TrackedIgnoreRule{
		Pattern:  pattern,
		Location: location,
		Hits:     0,
	}
}

// WithIgnorePaths registers ignore rules with caller file and line tracking,
// so unused rules can be detected and pruned with -focus.prune-ignores.
func WithIgnorePaths(patterns ...string) Opt {
	loc := callerLocation(1)
	return func(g *Golden) {
		for _, pat := range patterns {
			g.trackIgnoreRule(pat, loc)
		}
		g.opts = append(g.opts, focus.IgnorePaths(patterns...))
	}
}

// callerLocation returns the filename and line number of the caller, e.g. "payment_test.go:42".
func callerLocation(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}
