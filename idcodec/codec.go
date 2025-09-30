package idcodec

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
)

const DefaultAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	ErrInvalidLength     = errors.New("idcodec: invalid length")
	ErrVersionMismatch   = errors.New("idcodec: version mismatch")
	ErrMACVerification   = errors.New("idcodec: MAC verification failed")
	ErrInvalidBase62Char = errors.New("idcodec: invalid base62 character")
	ErrBadConfig         = errors.New("idcodec: bad config")
)

type Config struct {
	Secret   []byte
	Version  uint8
	MacLen   int
	Alphabet string
	Domain   []byte
	Kind     *byte
}

type Codec struct {
	version  uint8
	macLen   int
	alphabet string
	rev      [256]int8
	k1, k2   uint64
	k3, k4   uint64
	macKey   [32]byte
	domain   []byte
	kind     *byte
}

func NewCodecFromSecret(cfg Config) (*Codec, error) {
	if cfg.MacLen <= 0 {
		return nil, fmt.Errorf("%w: macLen must be > 0", ErrBadConfig)
	}
	if cfg.Version >= 62 {
		return nil, fmt.Errorf("%w: version must be < 62", ErrBadConfig)
	}
	alp := cfg.Alphabet
	if alp == "" {
		alp = DefaultAlphabet
	}
	if len(alp) < 62 {
		return nil, fmt.Errorf("%w: alphabet must have >=62 distinct chars", ErrBadConfig)
	}

	kMaster := sha256.Sum256(cfg.Secret)

	derive := func(label byte) [32]byte {
		h := sha256.New()
		h.Write(kMaster[:])
		h.Write([]byte{label})
		var out [32]byte
		copy(out[:], h.Sum(nil))
		return out
	}

	kRaw := derive('K')
	k1 := binary.BigEndian.Uint64(kRaw[0:8])
	k2 := binary.BigEndian.Uint64(kRaw[8:16])
	k3 := binary.BigEndian.Uint64(kRaw[16:24])
	k4 := binary.BigEndian.Uint64(kRaw[24:32])
	mac := derive('M')

	c := &Codec{
		version:  cfg.Version,
		macLen:   cfg.MacLen,
		alphabet: alp,
		k1:       k1,
		k2:       k2,
		k3:       k3,
		k4:       k4,
		macKey:   mac,
		domain:   append([]byte(nil), cfg.Domain...),
		kind:     cfg.Kind,
	}
	for i := range c.rev {
		c.rev[i] = -1
	}
	for i := 0; i < len(alp) && i < 62; i++ {
		c.rev[alp[i]] = int8(i)
	}
	return c, nil
}

func NewCodec(masterKey [32]byte, version uint8, macLen int) (*Codec, error) {
	return NewCodecFromSecret(Config{Secret: masterKey[:], Version: version, MacLen: macLen})
}

func MustNewCodecFromSecret(cfg Config) *Codec {
	c, err := NewCodecFromSecret(cfg)
	if err != nil {
		panic(err)
	}
	return c
}

func (c *Codec) base62EncodeFixed(u uint64) string {
	var buf [11]byte
	base := uint64(62)
	for i := 10; i >= 0; i-- {
		buf[i] = c.alphabet[u%base]
		u /= base
	}
	return string(buf[:])
}

func (c *Codec) base62DecodeFixed(s string) (uint64, error) {
	if len(s) != 11 {
		return 0, ErrInvalidLength
	}
	var u uint64
	base := uint64(62)
	for i := 0; i < 11; i++ {
		v := c.rev[s[i]]
		if v < 0 {
			return 0, ErrInvalidBase62Char
		}
		u = u*base + uint64(v)
	}
	return u, nil
}

func toBytes8(u uint64) [8]byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], u)
	return b
}

func (c *Codec) permutation(x uint64) uint64 {
	x ^= c.k1
	x = bits.RotateLeft64(x, 17)
	x += c.k2
	x = bits.RotateLeft64(x, 31)
	x ^= c.k3
	x += c.k4
	return x
}

func (c *Codec) inversePermutation(x uint64) uint64 {
	x -= c.k4
	x ^= c.k3
	x = bits.RotateLeft64(x, -31)
	x -= c.k2
	x = bits.RotateLeft64(x, -17)
	x ^= c.k1
	return x
}

func (c *Codec) hmacPadding(encrypted uint64, kind *byte) string {
	h := hmac.New(sha256.New, c.macKey[:])
	if len(c.domain) > 0 {
		h.Write(c.domain)
	}
	if kind != nil {
		h.Write([]byte{*kind})
	}
	h.Write([]byte{c.alphabet[c.version]})
	b8 := toBytes8(encrypted)
	h.Write(b8[:])
	sum := h.Sum(nil)

	out := make([]byte, c.macLen)
	var bitbuf uint64
	var bitsIn uint
	idx := 0
	src := 0
	for idx < c.macLen {
		for bitsIn < 6 && src < len(sum) {
			bitbuf = (bitbuf << 8) | uint64(sum[src])
			bitsIn += 8
			src++
		}
		if bitsIn < 6 {
			bitbuf <<= (6 - bitsIn)
			bitsIn = 6
		}
		shift := bitsIn - 6
		val := (bitbuf >> shift) & 0x3F
		bitsIn -= 6
		if val >= 62 {
			val -= 2
		}
		out[idx] = c.alphabet[val]
		idx++
	}
	return string(out)
}

func (c *Codec) token(enc uint64, kind *byte) string {
	body := c.base62EncodeFixed(enc)
	pad := c.hmacPadding(enc, kind)
	return string([]byte{c.alphabet[c.version]}) + body + pad
}

func (c *Codec) EncodeUint64(id uint64) string {
	return c.EncodeUint64WithKind(id, c.kind)
}

func (c *Codec) EncodeUint64WithKind(id uint64, kind *byte) string {
	enc := c.permutation(id)
	return c.token(enc, kind)
}

func (c *Codec) Validate(s string) error {
	_, err := c.decodeInternal(s, false, c.kind)
	return err
}

func (c *Codec) DecodeToUint64(s string) (uint64, error) {
	return c.DecodeToUint64WithKind(s, c.kind)
}

func (c *Codec) DecodeToUint64WithKind(s string, kind *byte) (uint64, error) {
	return c.decodeInternal(s, true, kind)
}

func (c *Codec) decodeInternal(s string, needValue bool, kind *byte) (uint64, error) {
	need := 1 + 11 + c.macLen
	if len(s) != need {
		return 0, ErrInvalidLength
	}
	verCh := s[0]
	if c.rev[verCh] != int8(c.version) {
		return 0, ErrVersionMismatch
	}
	body := s[1 : 1+11]
	enc, err := c.base62DecodeFixed(body)
	if err != nil {
		return 0, err
	}
	pad := s[1+11:]
	exp := c.hmacPadding(enc, kind)
	if subtle.ConstantTimeCompare([]byte(pad), []byte(exp)) != 1 {
		return 0, ErrMACVerification
	}
	if !needValue {
		return 0, nil
	}
	return c.inversePermutation(enc), nil
}

func (c *Codec) DecodeBodyOnly(s string) (uint64, error) {
	if len(s) < 1+11 {
		return 0, ErrInvalidLength
	}
	body := s[1 : 1+11]
	return c.base62DecodeFixed(body)
}

func (c *Codec) MustEncodeUint64(id uint64) string {
	return c.EncodeUint64(id)
}

func (c *Codec) MustDecodeToUint64(s string) uint64 {
	u, err := c.DecodeToUint64(s)
	if err != nil {
		panic(err)
	}
	return u
}

type MultiCodec struct {
	Cur *Codec
	Old []*Codec
}

func (m MultiCodec) EncodeUint64(id uint64) string {
	return m.Cur.EncodeUint64(id)
}

func (m MultiCodec) DecodeToUint64(s string) (uint64, error) {
	if u, err := m.Cur.DecodeToUint64(s); err == nil {
		return u, nil
	}
	for _, c := range m.Old {
		if u, err := c.DecodeToUint64(s); err == nil {
			return u, nil
		}
	}
	return 0, ErrMACVerification
}
