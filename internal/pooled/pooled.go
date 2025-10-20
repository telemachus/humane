// Copyright 2025 Peter Aronoff
// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pooled provides pool-allocated types.
package pooled

import "sync"

// Buffer is a byte buffer.
//
// This implementation is adapted from the unexported type buffer
// in go/src/fmt/print.go.
type Buffer []byte

// Having an initial size gives a dramatic speedup.
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return (*Buffer)(&b)
	},
}

// NewBuffer returns a Buffer from the pool.
func NewBuffer() *Buffer {
	return bufPool.Get().(*Buffer) //nolint:errcheck // This cannot panic.
}

// Free returns buffers to the pool. To reduce peak allocation, only smaller
// buffers are returned to the pool.
func (b *Buffer) Free() {
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}

// Reset clears a buffer.
func (b *Buffer) Reset() {
	b.SetLen(0)
}

// Write appends a slice of bytes to a buffer.
func (b *Buffer) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

// WriteString appends a string to a buffer.
func (b *Buffer) WriteString(s string) (int, error) {
	*b = append(*b, s...)
	return len(s), nil
}

// WriteByte appends a byte to a buffer.
func (b *Buffer) WriteByte(c byte) error {
	*b = append(*b, c)
	return nil
}

// String returns a buffer as a string.
func (b *Buffer) String() string {
	return string(*b)
}

// Len returns the length of a buffer.
func (b *Buffer) Len() int {
	return len(*b)
}

// SetLen assigns n as the length of a buffer. Panics if n is larger than the
// buffer's capacity.
func (b *Buffer) SetLen(n int) {
	*b = (*b)[:n]
}

// StringSlice is a string slice.
type StringSlice []string

var stringSlicePool = sync.Pool{
	New: func() any {
		ss := make(StringSlice, 0, 10)
		return &ss
	},
}

// NewStringSlice returns a StringSlice from the pool.
func NewStringSlice() *StringSlice {
	return stringSlicePool.Get().(*StringSlice) //nolint:errcheck // This cannot panic.
}

// Free returns a StringSlice to the pool. (Note: there is no check on the
// length of the StringSlice.)
func (s *StringSlice) Free() {
	*s = (*s)[:0]
	stringSlicePool.Put(s)
}

// Append adds one or more strings to a StringSlice.
func (s *StringSlice) Append(strs ...string) {
	*s = append(*s, strs...)
}

// Len returns the length of a StringSlice.
func (s *StringSlice) Len() int {
	return len(*s)
}
