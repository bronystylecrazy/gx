package gx

import (
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/bronystylecrazy/gx/idcodec"
)

type ID uint64

var defaultCodec atomic.Pointer[idcodec.Codec]

func SetDefaultCodec(c *idcodec.Codec) {
	if c == nil {
		panic("gx: codec cannot be nil")
	}
	defaultCodec.Store(c)
}

func getCodec() (*idcodec.Codec, error) {
	c := defaultCodec.Load()
	if c == nil {
		return nil, errors.New("gx: DefaultCodec is nil (call gx.SetDefaultCodec at boot)")
	}
	return c, nil
}

func NewID(u uint64) *ID    { x := ID(u); return &x }
func (i ID) Uint64() uint64 { return uint64(i) }

func (i ID) MarshalJSON() ([]byte, error) {
	c, err := getCodec()
	if err != nil {
		return nil, err
	}
	s := c.EncodeUint64(uint64(i))
	return json.Marshal(s)
}

func (i *ID) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*i = 0
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			*i = 0
			return nil
		}
		c, err := getCodec()
		if err != nil {
			return err
		}
		u, err := c.DecodeToUint64(s)
		if err != nil {
			return fmt.Errorf("gx: invalid encoded ID: %w", err)
		}
		*i = ID(u)
		return nil
	}
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		*i = ID(n)
		return nil
	}
	return errors.New("gx: id must be string (encoded) or number")
}

func (i ID) Value() (driver.Value, error) {
	return int64(i), nil
}

func (i *ID) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*i = ID(uint64(v))
		return nil
	case []byte:
		u, err := strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("gx: scan []byte: %w", err)
		}
		*i = ID(u)
		return nil
	case string:
		u, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return fmt.Errorf("gx: scan string: %w", err)
		}
		*i = ID(u)
		return nil
	case uint64:
		*i = ID(v)
		return nil
	case nil:
		*i = 0
		return nil
	default:
		return fmt.Errorf("gx: unsupported scan type %T", src)
	}
}

func (i ID) String() string {
	c := defaultCodec.Load()
	if c == nil {
		return fmt.Sprintf("ID(%d)", uint64(i))
	}
	return c.EncodeUint64(uint64(i))
}

var _ encoding.TextMarshaler = (*ID)(nil)
var _ encoding.TextUnmarshaler = (*ID)(nil)

func (i ID) MarshalText() ([]byte, error) {
	c, err := getCodec()
	if err != nil {
		return nil, err
	}
	return []byte(c.EncodeUint64(uint64(i))), nil
}

func (i *ID) UnmarshalText(b []byte) error {
	c, err := getCodec()
	if err != nil {
		return err
	}
	u, err := c.DecodeToUint64(strings.TrimSpace(string(b)))
	if err != nil {
		return err
	}
	*i = ID(u)
	return nil
}

type NullID struct {
	ID    ID
	Valid bool
}

func (n NullID) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.ID.Value()
}

func (n *NullID) Scan(src any) error {
	if src == nil {
		n.Valid = false
		n.ID = 0
		return nil
	}
	n.Valid = true
	return n.ID.Scan(src)
}

func ParseIDString(s string) (*ID, error) {
	c, err := getCodec()
	if err != nil {
		return nil, err
	}
	u, err := c.DecodeToUint64(strings.TrimSpace(s))
	if err != nil {
		return nil, err
	}
	return NewID(u), nil
}
