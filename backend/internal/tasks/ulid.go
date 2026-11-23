package tasks

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"
)

// ulid generates lexicographically sortable unique IDs (ULID-style: 48-bit
// millisecond timestamp + 80 bits of randomness, Crockford base32). A tiny
// in-package implementation keeps the dependency graph lean for a single use.
const (
	ulidTimeBits  = 48
	ulidEntropy   = 10 // bytes of randomness (80 bits)
	ulidEncoded   = 26 // chars
	ulidMask      = (1 << 5) - 1
	ulidShiftHigh = 5
)

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var (
	ulidMu   sync.Mutex
	ulidLast uint64 // last ms timestamp issued; monotonic guard
)

// NewULID returns a new ULID string.
func NewULID() string {
	return newULID()
}

// newULID returns a new ULID string. Concurrent calls are serialized to keep
// monotonicity of the timestamp component.
func newULID() string {
	ulidMu.Lock()
	defer ulidMu.Unlock()

	ms := uint64(time.Now().UnixMilli())
	if ms <= ulidLast {
		ms = ulidLast + 1
	}
	ulidLast = ms

	var entropy [ulidEntropy]byte
	if err := binary.Read(rand.Reader, binary.BigEndian, &entropy); err != nil {
		// crypto/rand failing is unrecoverable; degrade to time-only IDs
		// rather than panicking the worker pool.
		for i := range entropy {
			entropy[i] = byte(ms >> (uint(i%8) * 8))
		}
	}

	// 128 bits → 26 base32 chars, MSB first (like oklog/ulid encoding).
	var bytes [16]byte
	bytes[0] = byte(ms >> 40)
	bytes[1] = byte(ms >> 32)
	bytes[2] = byte(ms >> 24)
	bytes[3] = byte(ms >> 16)
	bytes[4] = byte(ms >> 8)
	bytes[5] = byte(ms)
	copy(bytes[6:], entropy[:])

	var sb [ulidEncoded]byte
	// Encode 128 bits in 26 groups: first group is 3 bits, the rest 5 bits.
	acc := uint(0)
	bits := uint(0)
	pos := 0
	for _, b := range bytes {
		acc = acc<<8 | uint(b)
		bits += 8
		for bits >= ulidShiftHigh {
			bits -= ulidShiftHigh
			sb[pos] = crockford[(acc>>bits)&ulidMask]
			pos++
		}
	}
	// Remaining bits (2) shifted up to 5 bits, LSB-aligned.
	if bits > 0 {
		sb[pos] = crockford[(acc<<(ulidShiftHigh-bits))&ulidMask]
	}
	return string(sb[:])
}
