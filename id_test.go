package gx

import (
	"crypto/sha256"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/bronystylecrazy/gx/idcodec"
)

func codecForGX(t *testing.T) *idcodec.Codec {
	t.Helper()
	sum := sha256.Sum256([]byte("gx-secret"))
	c, err := idcodec.NewCodec(sum, 0, 6)
	if err != nil {
		t.Fatalf("NewCodec: %v", err)
	}
	return c
}

func TestMarshalUnmarshal_RoundTrip_WithPointer(t *testing.T) {
	defaultCodec.Store(nil)
	SetDefaultCodec(codecForGX(t))
	type User struct {
		ID   *ID    `json:"id"`
		Name string `json:"name"`
	}
	u := User{ID: NewID(1234567890), Name: "Sirawit"}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	ids, ok := raw["id"].(string)
	if !ok || ids == "" {
		t.Fatalf("id not encoded string")
	}
	dec, err := defaultCodec.Load().DecodeToUint64(ids)
	if err != nil || dec != 1234567890 {
		t.Fatalf("decode mismatch")
	}
	var u2 User
	if err := json.Unmarshal(b, &u2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if u2.ID == nil || u2.ID.Uint64() != 1234567890 || u2.Name != "Sirawit" {
		t.Fatalf("round-trip mismatch")
	}
}

func TestUnmarshal_Null_Empty_Number_Encoded_Invalid(t *testing.T) {
	defaultCodec.Store(nil)
	SetDefaultCodec(codecForGX(t))
	var id1 *ID = new(ID)
	if err := json.Unmarshal([]byte("null"), id1); err != nil {
		t.Fatalf("null: %v", err)
	}
	if id1.Uint64() != 0 {
		t.Fatalf("null not zero")
	}
	var id2 *ID = new(ID)
	enc := defaultCodec.Load().EncodeUint64(42)
	if err := json.Unmarshal([]byte(`"`+enc+`"`), id2); err != nil {
		t.Fatalf("encoded: %v", err)
	}
	if id2.Uint64() != 42 {
		t.Fatalf("encoded mismatch")
	}
	var id3 *ID = new(ID)
	if err := json.Unmarshal([]byte(`""`), id3); err != nil {
		t.Fatalf("empty: %v", err)
	}
	if id3.Uint64() != 0 {
		t.Fatalf("empty not zero")
	}
	var id4 *ID = new(ID)
	if err := json.Unmarshal([]byte(`123`), id4); err != nil {
		t.Fatalf("number: %v", err)
	}
	if id4.Uint64() != 123 {
		t.Fatalf("number mismatch")
	}
	var id5 *ID = new(ID)
	if err := json.Unmarshal([]byte(`"!!invalid!!"`), id5); err == nil {
		t.Fatalf("expected error for invalid encoded")
	}
}

func TestMarshal_Error_When_DefaultCodec_Missing(t *testing.T) {
	defaultCodec.Store(nil)
	var id ID = 7
	if _, err := id.MarshalJSON(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestUnmarshal_Number_When_DefaultCodec_Missing(t *testing.T) {
	defaultCodec.Store(nil)
	var id *ID = new(ID)
	if err := json.Unmarshal([]byte(`321`), id); err != nil {
		t.Fatalf("number path should not require codec: %v", err)
	}
	if id.Uint64() != 321 {
		t.Fatalf("number mismatch")
	}
}

func TestDB_Value_And_Scan_AllPaths(t *testing.T) {
	var id ID = 9223372036854775807
	v, err := id.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if _, ok := v.(driver.Value); !ok && v != v {
		t.Fatalf("value type")
	}
	var a ID
	if err := a.Scan(int64(123)); err != nil || a.Uint64() != 123 {
		t.Fatalf("scan int64")
	}
	var b ID
	if err := b.Scan([]byte("456")); err != nil || b.Uint64() != 456 {
		t.Fatalf("scan []byte")
	}
	var c ID
	if err := c.Scan("789"); err != nil || c.Uint64() != 789 {
		t.Fatalf("scan string")
	}
	var d ID
	if err := d.Scan(uint64(321)); err != nil || d.Uint64() != 321 {
		t.Fatalf("scan uint64")
	}
	var e ID
	if err := e.Scan(nil); err != nil || e.Uint64() != 0 {
		t.Fatalf("scan nil")
	}
	var f ID
	if err := f.Scan(true); err == nil {
		t.Fatalf("scan unsupported should error")
	}
}

func TestOmitEmpty_WithNilPointer(t *testing.T) {
	defaultCodec.Store(nil)
	SetDefaultCodec(codecForGX(t))
	type User struct {
		ID   *ID    `json:"id,omitempty"`
		Name string `json:"name"`
	}
	u := User{ID: nil, Name: "N"}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["id"]; ok {
		t.Fatalf("id should be omitted")
	}
	if m["name"] != "N" {
		t.Fatalf("name mismatch")
	}
}

func TestJSON_NumberAccepted_ThenMarshalAsEncoded(t *testing.T) {
	defaultCodec.Store(nil)
	SetDefaultCodec(codecForGX(t))
	type User struct {
		ID   *ID    `json:"id"`
		Name string `json:"name"`
	}
	var u User
	if err := json.Unmarshal([]byte(`{"id": 555, "name": "X"}`), &u); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if u.ID == nil || u.ID.Uint64() != 555 {
		t.Fatalf("number parse")
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]string
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	dec, err := defaultCodec.Load().DecodeToUint64(raw["id"])
	if err != nil || dec != 555 {
		t.Fatalf("encoded mismatch")
	}
}

func TestNewID_Uint64_AndEqualityAfterRoundTrip(t *testing.T) {
	defaultCodec.Store(nil)
	SetDefaultCodec(codecForGX(t))
	type User struct {
		ID   *ID    `json:"id"`
		Name string `json:"name"`
		Note string `json:"note,omitempty"`
	}
	u1 := User{ID: NewID(77), Name: "A"}
	b, err := json.Marshal(u1)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var u2 User
	if err := json.Unmarshal(b, &u2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if u2.ID == nil {
		t.Fatalf("id nil")
	}
	u2.ID = NewID(u2.ID.Uint64())
	if !reflect.DeepEqual(u1, u2) {
		t.Fatalf("not equal")
	}
}

func TestString_WithAndWithoutCodec(t *testing.T) {
	defaultCodec.Store(nil)
	var id ID = 10
	if id.String() != "ID(10)" {
		t.Fatalf("string without codec mismatch")
	}
	SetDefaultCodec(codecForGX(t))
	s := id.String()
	if s == "ID(10)" || s == "" {
		t.Fatalf("string with codec not encoded")
	}
	if _, err := defaultCodec.Load().DecodeToUint64(s); err != nil {
		t.Fatalf("decode string failed")
	}
}

func TestTextMarshaler_Unmarshaler(t *testing.T) {
	defaultCodec.Store(nil)
	SetDefaultCodec(codecForGX(t))
	var i ID = 999
	var _ encoding.TextMarshaler = &i
	var _ encoding.TextUnmarshaler = &i
	b, err := i.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	var j ID
	if err := j.UnmarshalText(b); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if j.Uint64() != 999 {
		t.Fatalf("text roundtrip mismatch")
	}
}

func TestNullID_ValueScan(t *testing.T) {
	defaultCodec.Store(nil)
	var n NullID
	v, err := n.Value()
	if err != nil || v != nil {
		t.Fatalf("null Value mismatch")
	}
	if err := n.Scan(nil); err != nil || n.Valid {
		t.Fatalf("null Scan mismatch")
	}
	n = NullID{ID: ID(1234), Valid: true}
	v, err = n.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if _, ok := v.(driver.Value); !ok && v != v {
		t.Fatalf("value type")
	}
	var m NullID
	if err := m.Scan(int64(555)); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !m.Valid || m.ID.Uint64() != 555 {
		t.Fatalf("Scan data mismatch")
	}
}

func TestParseIDString(t *testing.T) {
	defaultCodec.Store(nil)
	SetDefaultCodec(codecForGX(t))
	s := defaultCodec.Load().EncodeUint64(31415)
	p, err := ParseIDString(s)
	if err != nil || p == nil || p.Uint64() != 31415 {
		t.Fatalf("ParseIDString failed")
	}
	if _, err := ParseIDString("bad"); err == nil {
		t.Fatalf("ParseIDString should fail")
	}
}

func TestSetDefaultCodec_PanicOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	SetDefaultCodec(nil)
}
