// core/slug/slug.go

package slug

import (
	"errors"
	"strconv"
	"strings"
)

const MaxLen = 12

var (
	ErrEmpty       = errors.New("slug: empty")
	ErrTooLong     = errors.New("slug: too long")
	ErrInvalidChar = errors.New("slug: invalid character")
	ErrLeadingZero = errors.New("slug: leading zero")
	ErrNonPositive = errors.New("slug: non-positive id")
	ErrOverflow    = errors.New("slug: overflow")
)

// Encode returns the uppercase base36 representation of id.
// id must be > 0.
func Encode(id int64) (string, error) {
	if id <= 0 {
		return "", ErrNonPositive
	}
	return strings.ToUpper(strconv.FormatInt(id, 36)), nil
}

// MustEncode panics on invalid id; use only when id is known-valid.
func MustEncode(id int64) string {
	s, err := Encode(id)
	if err != nil {
		panic(err)
	}
	return s
}

// Decode parses a strict canonical uppercase base36 slug into id > 0.
// Lowercase, leading zero, empty, overflow, or non-alnum input is rejected.
func Decode(s string) (int64, error) {
	if len(s) == 0 {
		return 0, ErrEmpty
	}
	if len(s) > MaxLen {
		return 0, ErrTooLong
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z')) {
			return 0, ErrInvalidChar
		}
	}
	if s == "0" {
		return 0, ErrNonPositive
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, ErrLeadingZero
	}
	id, err := strconv.ParseInt(s, 36, 64)
	if err != nil {
		return 0, ErrOverflow
	}
	if id <= 0 {
		return 0, ErrNonPositive
	}
	return id, nil
}

// Valid reports whether s is a syntactically valid canonical slug.
func Valid(s string) bool {
	_, err := Decode(s)
	return err == nil
}
