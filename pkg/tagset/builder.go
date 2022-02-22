// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.Datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package tagset

import "github.com/twmb/murmur3"

// Builder is used to build tagsets tag-by-tag, before producing Tags instance
// when it is closed. Builders are not thread-safe.
//
// It is invalid to use a Builder after Close, as it may be re-used by other goroutines.
//
// In general, Builders are intended to be used in a single method. Avoid
// storing builders in structs.
type Builder struct {
	factory Factory
	tags    []string
	hashes  []uint64
	hash    uint64
}

// NewBuilder creates a new builder, using the given factory to create Tags
// instances.
//
// Builders can be re-used to save allocations, by calling Reset between
// each use.
func NewBuilder(factory Factory, capacity int) *Builder {
	bldr := &Builder{factory: factory}
	bldr.Reset(capacity)
	return bldr
}

// Reset the builder, preparing it for re-use.  This ensures that at least the
// given capacity is available, but will not _shrink_ the capacity.
func (bldr *Builder) Reset(capacity int) {
	// ensure at least the requested capacity for bldr.tags
	if bldr.tags == nil || cap(bldr.tags) < capacity {
		bldr.tags = make([]string, 0, capacity)
	} else {
		bldr.tags = bldr.tags[:0]
	}

	// ensure at least the requested capacity for bldr.hashes
	if bldr.hashes == nil || cap(bldr.hashes) < capacity {
		bldr.hashes = make([]uint64, 0, capacity)
	} else {
		bldr.hashes = bldr.hashes[:0]
	}

	bldr.hash = 0
}

// Add adds the given tag to the builder
func (bldr *Builder) Add(tag string) {
	h := murmur3.StringSum64(tag)
	for _, sh := range bldr.hashes {
		if sh == h {
			return
		}
	}
	bldr.tags = append(bldr.tags, tag)
	bldr.hashes = append(bldr.hashes, h)
	bldr.hash ^= h
}

// AddTags adds the contents of another Tags instance to this builder.
func (bldr *Builder) AddTags(tags *Tags) {
	tags.ForEach(bldr.Add)
}

// AddKV adds the tag "k:v" to the builder
func (bldr *Builder) AddKV(k, v string) {
	tag := k + ":" + v
	bldr.Add(tag)
}

// Contains checks whether the given tag is in the builder
func (bldr *Builder) Contains(tag string) bool {
	h := murmur3.StringSum64(tag)
	for _, sh := range bldr.hashes {
		if sh == h {
			return true
		}
	}
	return false
}

// Close builds the resulting *Tags, and frees resources associated with the Builder.
//
// If the Builder will be re-used, Reset() must be called first.
func (bldr *Builder) Close() *Tags {
	frozen := bldr.factory.getCachedTags(byTagsetHashCache, bldr.hash, func() *Tags {

		tags, hashes, hash := bldr.tags, bldr.hashes, bldr.hash
		// the Tags instance will own the storage in these slices, so reset them
		bldr.tags = []string{}
		bldr.hashes = []uint64{}

		return &Tags{tags, hashes, hash}
	})
	return frozen
}