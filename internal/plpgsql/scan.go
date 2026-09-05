//line scan.rl:1
package plpgsql

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

type token struct {
	kind             int
	text             string
	start, end, line int
	quoted           bool
}
type scanner struct {
	source    string
	pos, line int
}

//line scan.rl:55

//line scan.go:20
var _plpgsql_scanner_actions []byte = []byte{
	0, 1, 0, 1, 1, 1, 2, 1, 3,
	1, 4, 1, 5, 1, 6, 1, 7,
	1, 8, 1, 9, 1, 10, 1, 11,
	1, 12, 1, 13, 1, 14, 1, 15,
	1, 16, 1, 17,
}

var _plpgsql_scanner_key_offsets []byte = []byte{
	0, 2, 33, 36, 37, 39, 40, 41,
	48, 50, 51, 52, 63, 75,
}

var _plpgsql_scanner_trans_keys []byte = []byte{
	34, 39, 32, 34, 36, 39, 45, 46,
	47, 58, 59, 60, 69, 85, 96, 101,
	117, 0, 8, 9, 13, 14, 44, 48,
	57, 61, 62, 63, 64, 91, 94, 123,
	127, 32, 9, 13, 45, 10, 13, 46,
	42, 95, 48, 57, 65, 90, 97, 122,
	58, 61, 60, 62, 96, 0, 35, 37,
	47, 58, 64, 91, 94, 123, 127, 39,
	96, 0, 35, 37, 47, 58, 64, 91,
	94, 123, 127, 38, 96, 0, 35, 37,
	47, 58, 64, 91, 94, 123, 127,
}

var _plpgsql_scanner_single_lengths []byte = []byte{
	2, 15, 1, 1, 2, 1, 1, 1,
	2, 1, 1, 1, 2, 2,
}

var _plpgsql_scanner_range_lengths []byte = []byte{
	0, 8, 1, 0, 0, 0, 0, 3,
	0, 0, 0, 5, 5, 5,
}

var _plpgsql_scanner_index_offsets []byte = []byte{
	0, 3, 27, 30, 32, 35, 37, 39,
	44, 47, 49, 51, 58, 66,
}

var _plpgsql_scanner_indicies []byte = []byte{
	1, 2, 0, 4, 5, 6, 7, 8,
	9, 10, 12, 3, 13, 16, 17, 3,
	16, 17, 3, 4, 3, 11, 14, 3,
	3, 3, 15, 4, 4, 18, 20, 19,
	21, 21, 20, 22, 19, 23, 19, 11,
	11, 11, 11, 24, 22, 22, 19, 22,
	19, 22, 19, 25, 25, 25, 25, 25,
	25, 15, 26, 25, 25, 25, 25, 25,
	25, 15, 27, 25, 25, 25, 25, 25,
	25, 15,
}

var _plpgsql_scanner_trans_targs []byte = []byte{
	1, 1, 1, 1, 2, 1, 1, 1,
	3, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 1, 1, 4, 1, 1, 1,
	1, 1, 1, 0,
}

var _plpgsql_scanner_trans_actions []byte = []byte{
	35, 11, 9, 23, 0, 17, 19, 15,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 5, 25, 33, 0, 27, 21, 7,
	31, 29, 13, 0,
}

var _plpgsql_scanner_to_state_actions []byte = []byte{
	0, 1, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
}

var _plpgsql_scanner_from_state_actions []byte = []byte{
	0, 3, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0,
}

var _plpgsql_scanner_eof_trans []byte = []byte{
	1, 0, 19, 20, 22, 20, 20, 25,
	20, 20, 20, 26, 26, 26,
}

const plpgsql_scanner_start int = 1
const plpgsql_scanner_first_final int = 1
const plpgsql_scanner_error int = -1

const plpgsql_scanner_en_main int = 1

//line scan.rl:58

func (s *scanner) next() token {
	data := s.source
	p, pe, eof := s.pos, len(data), len(data)
	cs, ts, te, act := 0, 0, 0, 0
	tok := token{start: len(data), end: len(data)}

//line scan.go:120
	{
		cs = plpgsql_scanner_start
		ts = 0
		te = 0
		act = 0
	}

//line scan.rl:65

//line scan.go:130
	{
		var _klen int
		var _trans int
		var _acts int
		var _nacts uint
		var _keys int
		if p == pe {
			goto _test_eof
		}
	_resume:
		_acts = int(_plpgsql_scanner_from_state_actions[cs])
		_nacts = uint(_plpgsql_scanner_actions[_acts])
		_acts++
		for ; _nacts > 0; _nacts-- {
			_acts++
			switch _plpgsql_scanner_actions[_acts-1] {
			case 1:
//line NONE:1
				ts = p

//line scan.go:150
			}
		}

		_keys = int(_plpgsql_scanner_key_offsets[cs])
		_trans = int(_plpgsql_scanner_index_offsets[cs])

		_klen = int(_plpgsql_scanner_single_lengths[cs])
		if _klen > 0 {
			_lower := int(_keys)
			var _mid int
			_upper := int(_keys + _klen - 1)
			for {
				if _upper < _lower {
					break
				}

				_mid = _lower + ((_upper - _lower) >> 1)
				switch {
				case data[p] < _plpgsql_scanner_trans_keys[_mid]:
					_upper = _mid - 1
				case data[p] > _plpgsql_scanner_trans_keys[_mid]:
					_lower = _mid + 1
				default:
					_trans += int(_mid - int(_keys))
					goto _match
				}
			}
			_keys += _klen
			_trans += _klen
		}

		_klen = int(_plpgsql_scanner_range_lengths[cs])
		if _klen > 0 {
			_lower := int(_keys)
			var _mid int
			_upper := int(_keys + (_klen << 1) - 2)
			for {
				if _upper < _lower {
					break
				}

				_mid = _lower + (((_upper - _lower) >> 1) & ^1)
				switch {
				case data[p] < _plpgsql_scanner_trans_keys[_mid]:
					_upper = _mid - 2
				case data[p] > _plpgsql_scanner_trans_keys[_mid+1]:
					_lower = _mid + 2
				default:
					_trans += int((_mid - int(_keys)) >> 1)
					goto _match
				}
			}
			_trans += _klen
		}

	_match:
		_trans = int(_plpgsql_scanner_indicies[_trans])
	_eof_trans:
		cs = int(_plpgsql_scanner_trans_targs[_trans])

		if _plpgsql_scanner_trans_actions[_trans] == 0 {
			goto _again
		}

		_acts = int(_plpgsql_scanner_trans_actions[_trans])
		_nacts = uint(_plpgsql_scanner_actions[_acts])
		_acts++
		for ; _nacts > 0; _nacts-- {
			_acts++
			switch _plpgsql_scanner_actions[_acts-1] {
			case 2:
//line NONE:1
				te = p + 1

			case 3:
//line scan.rl:21
				te = p + 1
				{
					depth, i := 1, ts+2
					for i < len(data) && depth > 0 {
						if strings.HasPrefix(data[i:], "/*") {
							depth++
							i += 2
						} else if strings.HasPrefix(data[i:], "*/") {
							depth--
							i += 2
						} else {
							i++
						}
					}
					if depth != 0 {
						failAt(data, ts, "unterminated block comment")
					}
					p = i - 1
				}
			case 4:
//line scan.rl:29
				te = p + 1
				{
					tok = s.unicodeToken(ts, '\'')
					p = tok.end - 1
					p++
					goto _out
				}
			case 5:
//line scan.rl:30
				te = p + 1
				{
					tok = s.unicodeToken(ts, '"')
					p = tok.end - 1
					p++
					goto _out
				}
			case 6:
//line scan.rl:31
				te = p + 1
				{
					tok = s.quotedToken(ts, ts+1, '\'', true)
					p = tok.end - 1
					p++
					goto _out
				}
			case 7:
//line scan.rl:32
				te = p + 1
				{
					tok = s.quotedToken(ts, ts, '\'', false)
					p = tok.end - 1
					p++
					goto _out
				}
			case 8:
//line scan.rl:33
				te = p + 1
				{
					tok = s.quotedToken(ts, ts, '"', false)
					p = tok.end - 1
					p++
					goto _out
				}
			case 9:
//line scan.rl:34
				te = p + 1
				{
					i := ts + 1
					for i < len(data) && (data[i] == '_' || data[i] >= 'a' && data[i] <= 'z' || data[i] >= 'A' && data[i] <= 'Z' || i > ts+1 && data[i] >= '0' && data[i] <= '9' || data[i] >= 128) {
						i++
					}
					if i < len(data) && data[i] == '$' {
						delim := data[ts : i+1]
						end := strings.Index(data[i+1:], delim)
						if end < 0 {
							failAt(data, ts, "unterminated dollar-quoted string")
						}
						tok = token{kind: STRING, text: data[i+1 : i+1+end], start: ts, end: i + 1 + end + len(delim)}
					} else {
						i = ts + 1
						for i < len(data) && data[i] >= '0' && data[i] <= '9' {
							i++
						}
						tok = token{kind: WORD, text: data[ts:i], start: ts, end: i}
					}
					p = tok.end - 1
					p++
					goto _out

				}
			case 10:
//line scan.rl:52
				te = p + 1
				{
					tok = token{kind: SYMBOL, text: data[ts:te], start: ts, end: te}
					p++
					goto _out
				}
			case 11:
//line scan.rl:53
				te = p + 1
				{
					tok = token{kind: int(data[ts]), text: data[ts:te], start: ts, end: te}
					p++
					goto _out
				}
			case 12:
//line scan.rl:19
				te = p
				p--

			case 13:
//line scan.rl:20
				te = p
				p--

			case 14:
//line scan.rl:50
				te = p
				p--
				{
					tok = token{kind: WORD, text: foldIdentifier(data[ts:te]), start: ts, end: te}
					p++
					goto _out
				}
			case 15:
//line scan.rl:51
				te = p
				p--
				{
					tok = token{kind: NUMBER, text: data[ts:te], start: ts, end: te}
					p++
					goto _out
				}
			case 16:
//line scan.rl:53
				te = p
				p--
				{
					tok = token{kind: int(data[ts]), text: data[ts:te], start: ts, end: te}
					p++
					goto _out
				}
			case 17:
//line scan.rl:50
				p = (te) - 1
				{
					tok = token{kind: WORD, text: foldIdentifier(data[ts:te]), start: ts, end: te}
					p++
					goto _out
				}
//line scan.go:323
			}
		}

	_again:
		_acts = int(_plpgsql_scanner_to_state_actions[cs])
		_nacts = uint(_plpgsql_scanner_actions[_acts])
		_acts++
		for ; _nacts > 0; _nacts-- {
			_acts++
			switch _plpgsql_scanner_actions[_acts-1] {
			case 0:
//line NONE:1
				ts = 0

//line scan.go:337
			}
		}

		p++
		if p != pe {
			goto _resume
		}
	_test_eof:
		{
		}
		if p == eof {
			if _plpgsql_scanner_eof_trans[cs] > 0 {
				_trans = int(_plpgsql_scanner_eof_trans[cs] - 1)
				goto _eof_trans
			}
		}

	_out:
		{
		}
	}

//line scan.rl:66
	_ = cs
	_ = ts
	_ = te
	_ = act
	_ = eof
	tok.line = s.line + 1 + strings.Count(data[s.pos:tok.start], "\n")
	s.line += strings.Count(data[s.pos:p], "\n")
	s.pos = p
	if tok.kind == WORD && !tok.quoted {
		if k, ok := keywords[tok.text]; ok {
			tok.kind = k
		}
	}
	return tok
}

func (s *scanner) quotedToken(start, quoteAt int, quote byte, escape bool) token {
	var b strings.Builder
	data := s.source
	for i := quoteAt + 1; i < len(data); {
		c := data[i]
		i++
		if c == quote {
			if i < len(data) && data[i] == quote {
				b.WriteByte(quote)
				i++
				continue
			}
			if quote == '\'' {
				if next := continuedQuote(data, i); next >= 0 {
					i = next + 1
					continue
				}
			}
			kind, text := STRING, b.String()
			if !utf8.ValidString(text) {
				failAt(data, start, "invalid UTF-8 in string")
			}
			if quote == '"' {
				kind = WORD
				if b.Len() == 0 {
					failAt(data, start, "zero-length delimited identifier")
				}
				if quoteAt == start {
					text = truncateIdentifier(text)
				}
			}
			return token{kind: kind, text: text, start: start, end: i, quoted: quote == '"'}
		}
		if c == '\\' && escape {
			if i == len(data) {
				break
			}
			c = data[i]
			i++
			switch c {
			case 'b':
				c = '\b'
			case 'f':
				c = '\f'
			case 'n':
				c = '\n'
			case 'r':
				c = '\r'
			case 't':
				c = '\t'
			case 'u', 'U':
				value, end := s.escapeRune(i, c)
				b.WriteRune(value)
				i = end
				continue
			case 'x':
				j := i
				for i < len(data) && i < j+2 && strings.ContainsRune("0123456789abcdefABCDEF", rune(data[i])) {
					i++
				}
				if i > j {
					v, _ := strconv.ParseUint(data[j:i], 16, 8)
					c = byte(v)
				}
			default:
				if c >= '0' && c <= '7' {
					j := i - 1
					for i < len(data) && i < j+3 && data[i] >= '0' && data[i] <= '7' {
						i++
					}
					v, _ := strconv.ParseUint(data[j:i], 8, 16)
					c = byte(v)
				}
			}
		}
		if c == 0 {
			failAt(data, start, "invalid zero byte in string")
		}
		b.WriteByte(c)
	}
	failAt(data, start, "unterminated quoted string")
	return token{}
}

func foldIdentifier(value string) string {
	var result []byte
	for i := range len(value) {
		if value[i] >= 'A' && value[i] <= 'Z' {
			if result == nil {
				result = []byte(value)
			}
			result[i] += 'a' - 'A'
		}
	}
	if result != nil {
		value = string(result)
	}
	return truncateIdentifier(value)
}
func truncateIdentifier(value string) string {
	if len(value) <= 63 {
		return value
	}
	end := 63
	for !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}
func continuedQuote(data string, pos int) int {
	newline := false
	for pos < len(data) {
		switch data[pos] {
		case ' ', '\t', '\f', '\v':
			pos++
		case '\n', '\r':
			newline = true
			pos++
		case '-':
			if !strings.HasPrefix(data[pos:], "--") {
				return -1
			}
			for pos < len(data) && data[pos] != '\n' && data[pos] != '\r' {
				pos++
			}
		default:
			if newline && data[pos] == '\'' {
				return pos
			}
			return -1
		}
	}
	return -1
}
