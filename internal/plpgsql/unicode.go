package plpgsql

import (
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func (s *scanner) unicodeToken(start int, quote byte) token {
	t := s.quotedToken(start, start+2, quote, false)
	escape := byte('\\')
	probe := scanner{source: s.source, pos: t.end}
	if keyword := probe.next(); keyword.text == "uescape" && !keyword.quoted {
		value := probe.next()
		if value.kind != STRING || len(value.text) != 1 || strings.ContainsRune("0123456789abcdefABCDEF+'\" \t\r\n\f\v", rune(value.text[0])) {
			failAt(s.source, value.start, "invalid Unicode escape character")
		}
		escape = value.text[0]
		t.end = value.end
	}
	var decoded strings.Builder
	var high rune
	for i := 0; i < len(t.text); {
		if t.text[i] != escape {
			if high != 0 {
				failAt(s.source, start, "invalid Unicode surrogate pair")
			}
			decoded.WriteByte(t.text[i])
			i++
			continue
		}
		i++
		if i < len(t.text) && t.text[i] == escape {
			if high != 0 {
				failAt(s.source, start, "invalid Unicode surrogate pair")
			}
			decoded.WriteByte(escape)
			i++
			continue
		}
		digits := 4
		if i < len(t.text) && t.text[i] == '+' {
			digits = 6
			i++
		}
		if i+digits > len(t.text) {
			failAt(s.source, start, "invalid Unicode escape")
		}
		value, err := strconv.ParseUint(t.text[i:i+digits], 16, 32)
		if err != nil || value == 0 || value > utf8.MaxRune {
			failAt(s.source, start, "invalid Unicode escape value")
		}
		i += digits
		r := rune(value)
		if high != 0 {
			if r < 0xdc00 || r > 0xdfff {
				failAt(s.source, start, "invalid Unicode surrogate pair")
			}
			r, high = utf16.DecodeRune(high, r), 0
		} else if r >= 0xd800 && r <= 0xdbff {
			high = r
			continue
		}
		if r >= 0xdc00 && r <= 0xdfff {
			failAt(s.source, start, "invalid Unicode surrogate pair")
		}
		decoded.WriteRune(r)
	}
	if high != 0 {
		failAt(s.source, start, "invalid Unicode surrogate pair")
	}
	t.text = decoded.String()
	if quote == '"' {
		t.text = truncateIdentifier(t.text)
	}
	return t
}

func (s *scanner) escapeRune(pos int, kind byte) (rune, int) {
	digits := 4
	if kind == 'U' {
		digits = 8
	}
	if pos+digits > len(s.source) {
		failAt(s.source, pos, "invalid Unicode escape")
	}
	value, err := strconv.ParseUint(s.source[pos:pos+digits], 16, 32)
	if err != nil || value == 0 || value > utf8.MaxRune {
		failAt(s.source, pos, "invalid Unicode escape value")
	}
	end, r := pos+digits, rune(value)
	if r >= 0xd800 && r <= 0xdbff {
		if end+2 > len(s.source) || s.source[end] != '\\' || s.source[end+1] != 'u' && s.source[end+1] != 'U' {
			failAt(s.source, pos, "invalid Unicode surrogate pair")
		}
		n := 4
		if s.source[end+1] == 'U' {
			n = 8
		}
		if end+2+n > len(s.source) {
			failAt(s.source, end, "invalid Unicode surrogate pair")
		}
		low, err := strconv.ParseUint(s.source[end+2:end+2+n], 16, 32)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			failAt(s.source, end, "invalid Unicode surrogate pair")
		}
		r = utf16.DecodeRune(r, rune(low))
		end += 2 + n
	} else if r >= 0xdc00 && r <= 0xdfff {
		failAt(s.source, pos, "invalid Unicode surrogate pair")
	}
	return r, end
}
