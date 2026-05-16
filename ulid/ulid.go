package ulid

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"
)

// ULID is the parsed binary ULID value.
type ULID [16]byte

// EncodedSize is the encoded length of a ulid.
const EncodedSize = 26

const minUnixMilli = 1257893000000

const crockfordLower = "0123456789abcdefghjkmnpqrstvwxyz"

// ErrInvalidULID is returned if the ULID is in an invalid format.
var ErrInvalidULID = errors.New("invalid ulid")

// NewULID constructs a new randomized ulid in lowercase.
func NewULID() string {
	var id ULID
	var ts [8]byte
	now := uint64(time.Now().UnixMilli())
	binary.BigEndian.PutUint64(ts[:], now)
	copy(id[:6], ts[2:])
	if _, err := rand.Read(id[6:]); err != nil {
		panic(err)
	}
	return id.stringLower()
}

// ParseULID parses and validates the lowercase ulid is the correct format.
func ParseULID(id string) (ULID, error) {
	var result ULID
	if len(id) != EncodedSize {
		return result, ErrInvalidULID
	}
	if id[0] > '7' {
		return result, ErrInvalidULID
	}
	var dec [EncodedSize]byte
	for i := 0; i < len(id); i++ {
		decoded, ok := decodeULIDChar(id[i])
		if !ok {
			return result, ErrInvalidULID
		}
		dec[i] = decoded
	}

	result[0] = (dec[0] << 5) | dec[1]
	result[1] = (dec[2] << 3) | (dec[3] >> 2)
	result[2] = (dec[3] << 6) | (dec[4] << 1) | (dec[5] >> 4)
	result[3] = (dec[5] << 4) | (dec[6] >> 1)
	result[4] = (dec[6] << 7) | (dec[7] << 2) | (dec[8] >> 3)
	result[5] = (dec[8] << 5) | dec[9]
	result[6] = (dec[10] << 3) | (dec[11] >> 2)
	result[7] = (dec[11] << 6) | (dec[12] << 1) | (dec[13] >> 4)
	result[8] = (dec[13] << 4) | (dec[14] >> 1)
	result[9] = (dec[14] << 7) | (dec[15] << 2) | (dec[16] >> 3)
	result[10] = (dec[16] << 5) | dec[17]
	result[11] = (dec[18] << 3) | dec[19]>>2
	result[12] = (dec[19] << 6) | (dec[20] << 1) | (dec[21] >> 4)
	result[13] = (dec[21] << 4) | (dec[22] >> 1)
	result[14] = (dec[22] << 7) | (dec[23] << 2) | (dec[24] >> 3)
	result[15] = (dec[24] << 5) | dec[25]

	if result.Time() < minUnixMilli {
		return result, ErrInvalidULID
	}
	return result, nil
}

// Time returns the Unix millisecond timestamp component.
func (u ULID) Time() uint64 {
	return uint64(u[5]) | uint64(u[4])<<8 |
		uint64(u[3])<<16 | uint64(u[2])<<24 |
		uint64(u[1])<<32 | uint64(u[0])<<40
}

// String returns the lowercase string encoding.
func (u ULID) String() string {
	return u.stringLower()
}

func (u ULID) stringLower() string {
	var out [EncodedSize]byte
	out[0] = crockfordLower[(u[0]&224)>>5]
	out[1] = crockfordLower[u[0]&31]
	out[2] = crockfordLower[(u[1]&248)>>3]
	out[3] = crockfordLower[((u[1]&7)<<2)|((u[2]&192)>>6)]
	out[4] = crockfordLower[(u[2]&62)>>1]
	out[5] = crockfordLower[((u[2]&1)<<4)|((u[3]&240)>>4)]
	out[6] = crockfordLower[((u[3]&15)<<1)|((u[4]&128)>>7)]
	out[7] = crockfordLower[(u[4]&124)>>2]
	out[8] = crockfordLower[((u[4]&3)<<3)|((u[5]&224)>>5)]
	out[9] = crockfordLower[u[5]&31]
	out[10] = crockfordLower[(u[6]&248)>>3]
	out[11] = crockfordLower[((u[6]&7)<<2)|((u[7]&192)>>6)]
	out[12] = crockfordLower[(u[7]&62)>>1]
	out[13] = crockfordLower[((u[7]&1)<<4)|((u[8]&240)>>4)]
	out[14] = crockfordLower[((u[8]&15)<<1)|((u[9]&128)>>7)]
	out[15] = crockfordLower[(u[9]&124)>>2]
	out[16] = crockfordLower[((u[9]&3)<<3)|((u[10]&224)>>5)]
	out[17] = crockfordLower[u[10]&31]
	out[18] = crockfordLower[(u[11]&248)>>3]
	out[19] = crockfordLower[((u[11]&7)<<2)|((u[12]&192)>>6)]
	out[20] = crockfordLower[(u[12]&62)>>1]
	out[21] = crockfordLower[((u[12]&1)<<4)|((u[13]&240)>>4)]
	out[22] = crockfordLower[((u[13]&15)<<1)|((u[14]&128)>>7)]
	out[23] = crockfordLower[(u[14]&124)>>2]
	out[24] = crockfordLower[((u[14]&3)<<3)|((u[15]&224)>>5)]
	out[25] = crockfordLower[u[15]&31]
	return string(out[:])
}

func decodeULIDChar(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'h':
		return c - 'a' + 10, true
	case c >= 'j' && c <= 'k':
		return c - 'j' + 18, true
	case c >= 'm' && c <= 'n':
		return c - 'm' + 20, true
	case c >= 'p' && c <= 't':
		return c - 'p' + 22, true
	case c >= 'v' && c <= 'z':
		return c - 'v' + 27, true
	default:
		return 0, false
	}
}
