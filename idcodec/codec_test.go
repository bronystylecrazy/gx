package idcodec

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func newCodecForTest(t *testing.T, ver uint8, macLen int, domain []byte, kind *byte) *Codec {
	t.Helper()
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	c, err := NewCodecFromSecret(Config{
		Secret:  secret,
		Version: ver,
		MacLen:  macLen,
		Domain:  domain,
		Kind:    kind,
	})
	if err != nil {
		t.Fatalf("NewCodecFromSecret: %v", err)
	}
	return c
}

func TestNewCodecFromSecret_ConfigErrors(t *testing.T) {
	_, err := NewCodecFromSecret(Config{Secret: []byte("x"), Version: 62, MacLen: 6})
	if err == nil {
		t.Fatalf("expected error for version")
	}
	_, err = NewCodecFromSecret(Config{Secret: []byte("x"), Version: 0, MacLen: 0})
	if err == nil {
		t.Fatalf("expected error for macLen")
	}
	_, err = NewCodecFromSecret(Config{Secret: []byte("x"), Version: 0, MacLen: 4, Alphabet: "abc"})
	if err == nil {
		t.Fatalf("expected error for alphabet")
	}
}

func TestEncodeDecode_RoundTrip_VariousVerMac(t *testing.T) {
	for _, mac := range []int{3, 4, 6, 8} {
		for _, ver := range []uint8{0, 1, 7, 61} {
			c := newCodecForTest(t, ver, mac, nil, nil)
			cases := []uint64{0, 1, 42, 1234567890, 1<<63 - 1, 1 << 63, 18446744073709551615}
			for _, id := range cases {
				s := c.EncodeUint64(id)
				if len(s) != 1+11+mac {
					t.Fatalf("length mismatch")
				}
				if s[0] != c.alphabet[ver] {
					t.Fatalf("version prefix mismatch")
				}
				u, err := c.DecodeToUint64(s)
				if err != nil || u != id {
					t.Fatalf("round-trip mismatch")
				}
			}
		}
	}
}

func TestDifferentKeys_ProduceDifferentTokens(t *testing.T) {
	k1 := sha256.Sum256([]byte("secret-A"))
	k2 := sha256.Sum256([]byte("secret-B"))
	c1, _ := NewCodec(k1, 0, 6)
	c2, _ := NewCodec(k2, 0, 6)
	id := uint64(1234567890)
	s1 := c1.EncodeUint64(id)
	s2 := c2.EncodeUint64(id)
	if s1 == s2 {
		t.Fatalf("tokens equal for different keys")
	}
}

func TestStableEncoding_SameKeySameToken(t *testing.T) {
	k := sha256.Sum256([]byte("stable-key"))
	c1, _ := NewCodec(k, 7, 5)
	c2, _ := NewCodec(k, 7, 5)
	s1 := c1.EncodeUint64(987654321)
	s2 := c2.EncodeUint64(987654321)
	if s1 != s2 {
		t.Fatalf("tokens differ for same key")
	}
}

func TestValidate_SuccessAndFail(t *testing.T) {
	c := newCodecForTest(t, 0, 6, []byte("dom"), nil)
	s := c.EncodeUint64(555)
	if err := c.Validate(s); err != nil {
		t.Fatalf("validate failed")
	}
	b := []byte(s)
	b[len(b)-1] = c.alphabet[(int(b[len(b)-1])%62+1)%62]
	if err := c.Validate(string(b)); !errors.Is(err, ErrMACVerification) {
		t.Fatalf("validate should fail")
	}
}

func TestDecode_MACMismatch_OnPadTamper(t *testing.T) {
	c := newCodecForTest(t, 0, 6, nil, nil)
	s := c.EncodeUint64(777)
	b := []byte(s)
	b[len(b)-1] = c.alphabet[(int(b[len(b)-1])%62+7)%62]
	if _, err := c.DecodeToUint64(string(b)); !errors.Is(err, ErrMACVerification) {
		t.Fatalf("expected MAC failure")
	}
}

func TestDecode_VersionMismatch(t *testing.T) {
	c0 := newCodecForTest(t, 0, 6, nil, nil)
	c1 := newCodecForTest(t, 1, 6, nil, nil)
	s := c0.EncodeUint64(12345)
	if _, err := c1.DecodeToUint64(s); !errors.Is(err, ErrVersionMismatch) {
		t.Fatalf("expected version mismatch")
	}
}

func TestDecode_InvalidChars(t *testing.T) {
	c := newCodecForTest(t, 0, 6, nil, nil)
	s := c.EncodeUint64(999)
	b := []byte(s)
	i := 1 + 3
	b[i] = '~'
	if _, err := c.DecodeToUint64(string(b)); !errors.Is(err, ErrInvalidBase62Char) {
		t.Fatalf("expected invalid char error")
	}
}

func TestDecode_LengthErrors(t *testing.T) {
	c := newCodecForTest(t, 0, 6, nil, nil)
	s := c.EncodeUint64(888)
	if _, err := c.DecodeToUint64(s[:len(s)-1]); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("trim error")
	}
	if _, err := c.DecodeToUint64(s + "A"); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("append error")
	}
	if _, err := c.DecodeToUint64(""); !errors.Is(err, ErrInvalidLength) {
		t.Fatalf("empty error")
	}
}

func TestDecodeBodyOnly_IgnoresPadding(t *testing.T) {
	c := newCodecForTest(t, 0, 6, nil, nil)
	s := c.EncodeUint64(2025)
	body1, err := c.DecodeBodyOnly(s)
	if err != nil {
		t.Fatalf("DecodeBodyOnly: %v", err)
	}
	b := []byte(s)
	b[len(b)-1] = c.alphabet[(int(b[len(b)-1])%62+11)%62]
	body2, err := c.DecodeBodyOnly(string(b))
	if err != nil {
		t.Fatalf("DecodeBodyOnly 2: %v", err)
	}
	if body1 != body2 {
		t.Fatalf("body changed by padding tamper")
	}
}

func TestKindAffectsToken_AndValidation(t *testing.T) {
	k := byte('U')
	c := newCodecForTest(t, 0, 6, []byte("dom"), &k)
	s := c.EncodeUint64(321)
	if _, err := c.DecodeToUint64WithKind(s, &k); err != nil {
		t.Fatalf("decode with same kind failed")
	}
	k2 := byte('P')
	if _, err := c.DecodeToUint64WithKind(s, &k2); !errors.Is(err, ErrMACVerification) {
		t.Fatalf("decode with different kind should fail")
	}
	s2 := c.EncodeUint64WithKind(321, &k2)
	if s == s2 {
		t.Fatalf("tokens should differ for kinds")
	}
}

func TestDomainAffectsToken(t *testing.T) {
	secret := []byte("app-secret")
	c1 := MustNewCodecFromSecret(Config{Secret: secret, Version: 0, MacLen: 6, Domain: []byte("A")})
	c2 := MustNewCodecFromSecret(Config{Secret: secret, Version: 0, MacLen: 6, Domain: []byte("B")})
	id := uint64(9090)
	s1 := c1.EncodeUint64(id)
	s2 := c2.EncodeUint64(id)
	if s1 == s2 {
		t.Fatalf("tokens equal across domains")
	}
	if _, err := c2.DecodeToUint64(s1); !errors.Is(err, ErrMACVerification) {
		t.Fatalf("cross-domain decode should fail")
	}
}

func TestAlphabetOverride_Works(t *testing.T) {
	alp := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	c := MustNewCodecFromSecret(Config{Secret: []byte("k"), Version: 1, MacLen: 5, Alphabet: alp})
	id := uint64(424242)
	s := c.EncodeUint64(id)
	u, err := c.DecodeToUint64(s)
	if err != nil || u != id {
		t.Fatalf("round-trip failed")
	}
}

func TestMustMethods_PanicOnError(t *testing.T) {
	k1 := sha256.Sum256([]byte("k1"))
	k2 := sha256.Sum256([]byte("k2"))
	c1, _ := NewCodec(k1, 0, 6)
	c2, _ := NewCodec(k2, 0, 6)
	s := c1.EncodeUint64(999)
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	_ = c2.MustDecodeToUint64(s)
}

func TestTokenShapeAndReadmeStyle(t *testing.T) {
	var key [32]byte
	raw, _ := hex.DecodeString("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	copy(key[:], raw[:32])
	c, err := NewCodec(key, 61, 4)
	if err != nil {
		t.Fatalf("NewCodec: %v", err)
	}
	id := uint64(1234567890)
	s := c.EncodeUint64(id)
	if len(s) != 1+11+4 {
		t.Fatalf("length")
	}
	if s[0] != c.alphabet[61] {
		t.Fatalf("prefix")
	}
	u, err := c.DecodeToUint64(s)
	if err != nil || u != id {
		t.Fatalf("decode mismatch")
	}
}

func TestMultiCodec_DecodeOld(t *testing.T) {
	kOld := sha256.Sum256([]byte("old"))
	kCur := sha256.Sum256([]byte("cur"))
	oldC, _ := NewCodec(kOld, 0, 6)
	curC, _ := NewCodec(kCur, 1, 6)
	mc := MultiCodec{Cur: curC, Old: []*Codec{oldC}}
	sOld := oldC.EncodeUint64(2024)
	u, err := mc.DecodeToUint64(sOld)
	if err != nil || u != 2024 {
		t.Fatalf("multi decode failed")
	}
	sCur := mc.EncodeUint64(3030)
	u2, err := mc.DecodeToUint64(sCur)
	if err != nil || u2 != 3030 {
		t.Fatalf("multi decode current failed")
	}
}
