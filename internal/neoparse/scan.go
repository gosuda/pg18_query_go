//line scan.rl:1
package neoparse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	pg_query "github.com/gosuda/pg18_query_go/internal/pgproto"
)

//line scan.go:18
var _pg_scanner_actions []byte = []byte{
	0, 1, 0, 1, 1, 1, 16, 1, 17,
	1, 18, 1, 19, 1, 20, 1, 21,
	1, 22, 1, 23, 1, 24, 1, 25,
	1, 26, 1, 27, 1, 28, 1, 29,
	1, 30, 1, 31, 1, 32, 1, 33,
	1, 34, 1, 35, 1, 36, 1, 37,
	1, 38, 1, 39, 1, 40, 1, 41,
	1, 42, 2, 2, 3, 2, 2, 4,
	2, 2, 5, 2, 2, 6, 2, 2,
	7, 2, 2, 8, 2, 2, 9, 2,
	2, 10, 2, 2, 11, 2, 2, 12,
	2, 2, 13, 2, 2, 14, 2, 2,
	15,
}

var _pg_scanner_key_offsets []int16 = []int16{
	0, 1, 4, 6, 7, 56, 59, 74,
	88, 89, 90, 95, 109, 125, 127, 130,
	143, 154, 169, 171, 182, 195, 208, 221,
	236, 250, 270, 284, 298, 311, 327, 340,
	353, 366, 379, 392, 405, 422, 439, 456,
	458, 474, 489, 504, 515, 527, 539, 551,
	563,
}

var _pg_scanner_trans_keys []byte = []byte{
	34, 36, 48, 122, 34, 39, 34, 32,
	33, 34, 36, 39, 44, 45, 46, 47,
	48, 58, 59, 60, 61, 62, 66, 69,
	78, 85, 88, 94, 96, 98, 101, 110,
	117, 120, 124, 126, 0, 8, 9, 13,
	14, 31, 35, 38, 40, 41, 42, 43,
	49, 57, 63, 64, 91, 93, 123, 127,
	32, 9, 13, 33, 35, 45, 47, 61,
	94, 96, 124, 126, 37, 38, 42, 43,
	60, 64, 33, 35, 45, 47, 94, 96,
	124, 126, 37, 38, 42, 43, 60, 64,
	34, 34, 36, 48, 57, 58, 122, 33,
	35, 45, 47, 94, 96, 124, 126, 37,
	38, 42, 43, 60, 64, 10, 13, 33,
	35, 45, 47, 94, 96, 124, 126, 37,
	38, 42, 43, 60, 64, 10, 13, 46,
	48, 57, 69, 95, 101, 0, 47, 48,
	57, 58, 64, 91, 96, 123, 127, 96,
	0, 35, 37, 47, 58, 64, 91, 94,
	123, 127, 43, 45, 96, 0, 35, 37,
	47, 48, 57, 58, 64, 91, 94, 123,
	127, 48, 57, 95, 0, 47, 48, 57,
	58, 64, 91, 96, 123, 127, 96, 0,
	35, 37, 47, 48, 57, 58, 64, 91,
	94, 123, 127, 95, 0, 35, 37, 47,
	48, 57, 58, 64, 91, 96, 123, 127,
	96, 0, 35, 37, 47, 48, 57, 58,
	64, 91, 94, 123, 127, 69, 95, 101,
	0, 35, 37, 47, 48, 57, 58, 64,
	91, 96, 123, 127, 33, 35, 42, 43,
	45, 47, 94, 96, 124, 126, 37, 38,
	60, 64, 46, 66, 69, 79, 88, 95,
	98, 101, 111, 120, 0, 47, 48, 57,
	58, 64, 91, 96, 123, 127, 46, 69,
	96, 101, 0, 47, 48, 57, 58, 64,
	91, 94, 123, 127, 46, 69, 95, 101,
	0, 47, 48, 57, 58, 64, 91, 96,
	123, 127, 96, 0, 35, 37, 47, 48,
	57, 58, 64, 91, 94, 123, 127, 46,
	69, 95, 101, 0, 35, 37, 47, 48,
	57, 58, 64, 91, 96, 123, 127, 95,
	0, 35, 37, 47, 48, 49, 58, 64,
	91, 96, 123, 127, 95, 0, 35, 37,
	47, 48, 49, 58, 64, 91, 96, 123,
	127, 96, 0, 35, 37, 47, 48, 49,
	58, 64, 91, 94, 123, 127, 95, 0,
	35, 37, 47, 48, 55, 58, 64, 91,
	96, 123, 127, 95, 0, 35, 37, 47,
	48, 55, 58, 64, 91, 96, 123, 127,
	96, 0, 35, 37, 47, 48, 55, 58,
	64, 91, 94, 123, 127, 95, 0, 35,
	37, 47, 48, 57, 58, 64, 65, 70,
	91, 96, 97, 102, 123, 127, 95, 0,
	35, 37, 47, 48, 57, 58, 64, 65,
	70, 91, 96, 97, 102, 123, 127, 96,
	0, 35, 37, 47, 48, 57, 58, 64,
	65, 70, 91, 94, 97, 102, 123, 127,
	58, 61, 33, 35, 45, 47, 61, 62,
	94, 96, 124, 126, 37, 38, 42, 43,
	60, 64, 33, 35, 45, 47, 62, 94,
	96, 124, 126, 37, 38, 42, 43, 60,
	64, 33, 35, 45, 47, 61, 94, 96,
	124, 126, 37, 38, 42, 43, 60, 64,
	96, 0, 35, 37, 47, 58, 64, 91,
	94, 123, 127, 39, 96, 0, 35, 37,
	47, 58, 64, 91, 94, 123, 127, 39,
	96, 0, 35, 37, 47, 58, 64, 91,
	94, 123, 127, 39, 96, 0, 35, 37,
	47, 58, 64, 91, 94, 123, 127, 38,
	96, 0, 35, 37, 47, 58, 64, 91,
	94, 123, 127, 34,
}

var _pg_scanner_single_lengths []byte = []byte{
	1, 1, 2, 1, 29, 1, 9, 8,
	1, 1, 1, 8, 10, 2, 1, 3,
	1, 3, 0, 1, 1, 1, 1, 3,
	10, 10, 4, 4, 1, 4, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 2,
	10, 9, 9, 1, 2, 2, 2, 2,
	1,
}

var _pg_scanner_range_lengths []byte = []byte{
	0, 1, 0, 0, 10, 1, 3, 3,
	0, 0, 2, 3, 3, 0, 1, 5,
	5, 6, 1, 5, 6, 6, 6, 6,
	2, 5, 5, 5, 6, 6, 6, 6,
	6, 6, 6, 6, 8, 8, 8, 0,
	3, 3, 3, 5, 5, 5, 5, 5,
	0,
}

var _pg_scanner_index_offsets []int16 = []int16{
	0, 2, 5, 8, 10, 50, 53, 66,
	78, 80, 82, 86, 98, 112, 115, 118,
	127, 134, 144, 146, 153, 161, 169, 177,
	187, 200, 216, 226, 236, 244, 255, 263,
	271, 279, 287, 295, 303, 313, 323, 333,
	336, 350, 363, 376, 383, 391, 399, 407,
	415,
}

var _pg_scanner_indicies []byte = []byte{
	2, 1, 3, 4, 0, 6, 7, 5,
	8, 6, 10, 11, 12, 14, 15, 9,
	16, 17, 18, 19, 21, 9, 22, 23,
	24, 26, 27, 28, 29, 26, 13, 13,
	26, 27, 28, 29, 26, 13, 13, 9,
	10, 9, 13, 9, 13, 20, 13, 9,
	9, 25, 10, 10, 30, 13, 13, 13,
	13, 32, 13, 13, 13, 13, 13, 13,
	13, 31, 13, 13, 13, 13, 13, 13,
	13, 13, 13, 13, 13, 0, 2, 1,
	1, 34, 3, 35, 4, 0, 13, 13,
	36, 13, 13, 13, 13, 13, 13, 13,
	13, 31, 37, 37, 36, 36, 36, 36,
	36, 36, 36, 36, 36, 36, 36, 38,
	37, 37, 38, 39, 40, 33, 43, 44,
	43, 41, 40, 41, 41, 41, 42, 45,
	45, 45, 45, 45, 45, 42, 46, 46,
	45, 45, 45, 47, 45, 45, 45, 42,
	48, 45, 50, 49, 48, 49, 49, 49,
	42, 45, 45, 45, 47, 45, 45, 45,
	42, 50, 49, 49, 47, 49, 49, 49,
	42, 45, 45, 45, 51, 45, 45, 45,
	42, 43, 44, 43, 41, 41, 51, 41,
	41, 41, 42, 13, 13, 52, 13, 13,
	13, 13, 13, 13, 13, 13, 13, 31,
	54, 55, 43, 56, 57, 58, 55, 43,
	56, 57, 53, 20, 53, 53, 53, 42,
	59, 43, 41, 43, 41, 40, 41, 41,
	41, 42, 54, 43, 58, 43, 53, 20,
	53, 53, 53, 42, 45, 45, 45, 60,
	45, 45, 45, 42, 54, 43, 58, 43,
	53, 53, 60, 53, 53, 53, 42, 63,
	61, 61, 62, 61, 61, 61, 42, 65,
	64, 64, 62, 64, 64, 64, 42, 0,
	0, 0, 62, 0, 0, 0, 42, 67,
	61, 61, 66, 61, 61, 61, 42, 69,
	68, 68, 66, 68, 68, 68, 42, 0,
	0, 0, 66, 0, 0, 0, 42, 71,
	61, 61, 70, 61, 70, 61, 70, 61,
	42, 73, 72, 72, 70, 72, 70, 72,
	70, 72, 42, 0, 0, 0, 70, 0,
	70, 0, 70, 0, 42, 74, 75, 33,
	13, 13, 13, 13, 76, 32, 13, 13,
	13, 13, 13, 13, 13, 31, 13, 13,
	13, 13, 77, 13, 13, 13, 13, 13,
	13, 13, 31, 13, 13, 13, 13, 78,
	13, 13, 13, 13, 13, 13, 13, 31,
	79, 79, 79, 79, 79, 79, 25, 80,
	79, 79, 79, 79, 79, 79, 25, 15,
	79, 79, 79, 79, 79, 79, 25, 81,
	79, 79, 79, 79, 79, 79, 25, 82,
	79, 79, 79, 79, 79, 79, 25, 6,
	83,
}

var _pg_scanner_trans_targs []byte = []byte{
	4, 0, 9, 4, 1, 4, 3, 4,
	48, 4, 5, 6, 8, 7, 10, 4,
	11, 14, 24, 25, 27, 39, 40, 41,
	42, 43, 44, 45, 46, 47, 4, 4,
	7, 4, 4, 10, 12, 4, 13, 4,
	15, 4, 16, 17, 22, 4, 18, 21,
	19, 4, 20, 23, 7, 4, 26, 30,
	33, 36, 28, 4, 29, 4, 31, 32,
	4, 32, 34, 35, 4, 35, 37, 38,
	4, 38, 4, 4, 7, 7, 7, 4,
	4, 4, 2, 4,
}

var _pg_scanner_trans_actions []byte = []byte{
	57, 0, 86, 19, 0, 55, 0, 17,
	83, 23, 0, 0, 95, 92, 95, 21,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 89, 25, 51,
	71, 53, 47, 74, 0, 27, 0, 7,
	0, 31, 0, 0, 0, 43, 0, 0,
	0, 29, 0, 0, 59, 39, 0, 0,
	0, 0, 0, 11, 0, 41, 0, 77,
	37, 80, 0, 77, 35, 80, 0, 77,
	33, 80, 5, 9, 65, 62, 68, 49,
	15, 13, 0, 45,
}

var _pg_scanner_to_state_actions []byte = []byte{
	0, 0, 0, 0, 1, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0,
}

var _pg_scanner_from_state_actions []byte = []byte{
	0, 0, 0, 0, 3, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0,
}

var _pg_scanner_eof_trans []int16 = []int16{
	1, 1, 6, 1, 0, 31, 32, 1,
	34, 35, 1, 32, 38, 38, 34, 42,
	46, 46, 46, 50, 46, 50, 46, 42,
	32, 54, 42, 54, 46, 54, 62, 65,
	1, 62, 69, 1, 62, 73, 1, 34,
	32, 32, 32, 80, 80, 80, 80, 80,
	84,
}

const pg_scanner_start int = 4
const pg_scanner_first_final int = 4
const pg_scanner_error int = -1

const pg_scanner_en_main int = 4

//line scan.rl:559

type Lexer struct {
	data               string
	p                  int
	pe                 int
	eof                int
	cs                 int
	ts                 int
	te                 int
	act                int
	statements         []Statement
	statementLocations []statementLocation
	tokenStart         int
	tokenEnd           int
	initialToken       int
	err                error
	scanMode           bool

	haveLookahead bool
	lookaheadTok  int
	lookaheadVal  yySymType
}

func NewLexer(input string) *Lexer {
	l := &Lexer{
		data: input,
		pe:   len(input),
		eof:  len(input),
	}
	var cs, ts, te, act int

//line scan.go:303
	{
		cs = pg_scanner_start
		ts = 0
		te = 0
		act = 0
	}

//line scan.rl:591
	l.cs = cs
	l.ts = ts
	l.te = te
	l.act = act
	return l
}

var newLexer = NewLexer

type lexer = Lexer

func (l *Lexer) rawLex(lval *yySymType) int {
	if l.err != nil {
		return 0
	}
	data := l.data
	p := l.p
	pe := l.pe
	eof := l.eof
	cs := l.cs
	ts := l.ts
	te := l.te
	act := l.act

	for p < pe {
		tok := 0
		var tokStart, tokEnd int32
		var keywordKind KeywordKind
		_ = tokStart
		_ = tokEnd
		_ = keywordKind

		oldP := p

//line scan.go:345
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
			_acts = int(_pg_scanner_from_state_actions[cs])
			_nacts = uint(_pg_scanner_actions[_acts])
			_acts++
			for ; _nacts > 0; _nacts-- {
				_acts++
				switch _pg_scanner_actions[_acts-1] {
				case 1:
//line NONE:1
					ts = p

//line scan.go:365
				}
			}

			_keys = int(_pg_scanner_key_offsets[cs])
			_trans = int(_pg_scanner_index_offsets[cs])

			_klen = int(_pg_scanner_single_lengths[cs])
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
					case data[p] < _pg_scanner_trans_keys[_mid]:
						_upper = _mid - 1
					case data[p] > _pg_scanner_trans_keys[_mid]:
						_lower = _mid + 1
					default:
						_trans += int(_mid - int(_keys))
						goto _match
					}
				}
				_keys += _klen
				_trans += _klen
			}

			_klen = int(_pg_scanner_range_lengths[cs])
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
					case data[p] < _pg_scanner_trans_keys[_mid]:
						_upper = _mid - 2
					case data[p] > _pg_scanner_trans_keys[_mid+1]:
						_lower = _mid + 2
					default:
						_trans += int((_mid - int(_keys)) >> 1)
						goto _match
					}
				}
				_trans += _klen
			}

		_match:
			_trans = int(_pg_scanner_indicies[_trans])
		_eof_trans:
			cs = int(_pg_scanner_trans_targs[_trans])

			if _pg_scanner_trans_actions[_trans] == 0 {
				goto _again
			}

			_acts = int(_pg_scanner_trans_actions[_trans])
			_nacts = uint(_pg_scanner_actions[_acts])
			_acts++
			for ; _nacts > 0; _nacts-- {
				_acts++
				switch _pg_scanner_actions[_acts-1] {
				case 2:
//line NONE:1
					te = p + 1

				case 3:
//line scan.rl:67
					act = 3
				case 4:
//line scan.rl:115
					act = 7
				case 5:
//line scan.rl:122
					act = 8
				case 6:
//line scan.rl:129
					act = 9
				case 7:
//line scan.rl:136
					act = 10
				case 8:
//line scan.rl:143
					act = 11
				case 9:
//line scan.rl:273
					act = 19
				case 10:
//line scan.rl:278
					act = 20
				case 11:
//line scan.rl:377
					act = 26
				case 12:
//line scan.rl:387
					act = 27
				case 13:
//line scan.rl:406
					act = 28
				case 14:
//line scan.rl:430
					act = 29
				case 15:
//line scan.rl:546
					act = 30
				case 16:
//line scan.rl:94
					te = p + 1
					{
						tok = TYPECAST
						tokStart = int32(ts)
						tokEnd = int32(te)
						p++
						goto _out

					}
				case 17:
//line scan.rl:101
					te = p + 1
					{
						tok = DOT_DOT
						tokStart = int32(ts)
						tokEnd = int32(te)
						p++
						goto _out

					}
				case 18:
//line scan.rl:108
					te = p + 1
					{
						tok = COLON_EQUALS
						tokStart = int32(ts)
						tokEnd = int32(te)
						p++
						goto _out

					}
				case 19:
//line scan.rl:159
					te = p + 1
					{
						// Integer followed by '..'; emit integer and rewind 2 bytes
						tokStart = int32(ts)
						tokEnd = int32(te - 2)
						p = te - 3
						raw := strings.ReplaceAll(data[ts:te-2], "_", "")
						if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts : te-2])
							}
						}
						p++
						goto _out

					}
				case 20:
//line scan.rl:284
					te = p + 1
					{
						tok = NCHAR
						tokStart = int32(ts)
						tokEnd = int32(ts + 1)
						keywordKind = KeywordKind_COL_NAME_KEYWORD
						p = ts
						if lval != nil {
							lval.setstr("nchar")
						}
						p++
						goto _out

					}
				case 21:
//line scan.rl:296
					te = p + 1
					{
						tok = BCONST
						prefix := "b"
						if data[ts] == 'x' || data[ts] == 'X' {
							tok = XCONST
							prefix = "x"
						}
						value, end, err := scanQuotedLiteral(data, te-1, literalBits)
						if err != nil {
							l.errorAt(err.Error(), ts)
							p++
							goto _out

						}
						tokStart = int32(ts)
						tokEnd = int32(end)
						p = end - 1
						if lval != nil {
							lval.setstr(prefix + value)
						}
						p++
						goto _out

					}
				case 22:
//line scan.rl:317
					te = p + 1
					{
						value, end, err := scanQuotedLiteral(data, te-1, literalStandard)
						if err != nil {
							l.errorAt(err.Error(), ts)
							p++
							goto _out

						}
						tok = USCONST
						tokStart = int32(ts)
						tokEnd = int32(end)
						p = end - 1
						if lval != nil {
							lval.setstr(value)
						}
						p++
						goto _out

					}
				case 23:
//line scan.rl:334
					te = p + 1
					{
						tag := data[ts:te]
						// search for closing tag
						endIdx := strings.Index(data[te:], tag)
						if endIdx >= 0 {
							fullEnd := te + endIdx + len(tag)
							if !utf8.ValidString(data[te : te+endIdx]) {
								l.errorAt("invalid UTF-8 string", ts)
								p++
								goto _out

							}
							tok = SCONST
							tokStart = int32(ts)
							tokEnd = int32(fullEnd)
							p = fullEnd - 1
							if lval != nil {
								lval.setstr(data[te : te+endIdx])
							}
							p++
							goto _out

						}
						l.errorAt("unterminated dollar-quoted string", ts)
						p++
						goto _out

					}
				case 24:
//line scan.rl:357
					te = p + 1
					{
						mode := literalStandard
						if data[ts] != '\'' {
							mode = literalEscape
						}
						value, end, err := scanQuotedLiteral(data, te-1, mode)
						if err != nil {
							l.errorAt(err.Error(), ts)
							p++
							goto _out

						}
						tok = SCONST
						tokStart = int32(ts)
						tokEnd = int32(end)
						p = end - 1
						if lval != nil {
							lval.setstr(value)
						}
						p++
						goto _out

					}
				case 25:
//line scan.rl:546
					te = p + 1
					{
						if data[ts] == '"' {
							l.errorAt("unterminated quoted identifier", ts)
							p++
							goto _out

						}
						tok = int(data[ts])
						tokStart = int32(ts)
						tokEnd = int32(ts + 1)
						p++
						goto _out

					}
				case 26:
//line scan.rl:55
					te = p
					p--
					{
						// whitespace ignored
					}
				case 27:
//line scan.rl:59
					te = p
					p--
					{
						tok = SQL_COMMENT
						tokStart = int32(ts)
						tokEnd = int32(te)
						keywordKind = KeywordKind_NO_KEYWORD
						p++
						goto _out

					}
				case 28:
//line scan.rl:179
					te = p
					p--
					{
						tok = FCONST
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := data[ts:te]
						if lval != nil {
							lval.setstr(raw)
						}
						p++
						goto _out

					}
				case 29:
//line scan.rl:190
					te = p
					p--
					{
						tok = FCONST
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := data[ts:te]
						if lval != nil {
							lval.setstr(raw)
						}
						p++
						goto _out

					}
				case 30:
//line scan.rl:201
					te = p
					p--
					{
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := strings.ReplaceAll(data[ts+2:te], "_", "")
						if n, err := strconv.ParseInt(raw, 16, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts:te])
							}
						}
						p++
						goto _out

					}
				case 31:
//line scan.rl:219
					te = p
					p--
					{
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := strings.ReplaceAll(data[ts+2:te], "_", "")
						if n, err := strconv.ParseInt(raw, 8, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts:te])
							}
						}
						p++
						goto _out

					}
				case 32:
//line scan.rl:237
					te = p
					p--
					{
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := strings.ReplaceAll(data[ts+2:te], "_", "")
						if n, err := strconv.ParseInt(raw, 2, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts:te])
							}
						}
						p++
						goto _out

					}
				case 33:
//line scan.rl:255
					te = p
					p--
					{
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := strings.ReplaceAll(data[ts:te], "_", "")
						if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts:te])
							}
						}
						p++
						goto _out

					}
				case 34:
//line scan.rl:273
					te = p
					p--
					{
						l.errorAt("invalid non-decimal integer", ts)
						p++
						goto _out

					}
				case 35:
//line scan.rl:278
					te = p
					p--
					{
						l.errorAt("trailing junk after numeric literal", ts)
						p++
						goto _out

					}
				case 36:
//line scan.rl:377
					te = p
					p--
					{
						tok = UIDENT
						tokStart = int32(ts)
						tokEnd = int32(te)
						if lval != nil {
							lval.setstr(unquoteIdent(data[ts:te]))
						}
						p++
						goto _out

					}
				case 37:
//line scan.rl:387
					te = p
					p--
					{
						value := unquoteIdent(data[ts:te])
						if value == "" {
							l.errorAt("zero-length delimited identifier", ts)
							p++
							goto _out

						}
						if !utf8.ValidString(value) {
							l.errorAt("invalid UTF-8 identifier", ts)
							p++
							goto _out

						}
						tok = IDENT
						tokStart = int32(ts)
						tokEnd = int32(te)
						if lval != nil {
							lval.setstr(truncateIdentifier(value))
						}
						p++
						goto _out

					}
				case 38:
//line scan.rl:406
					te = p
					p--
					{
						word := data[ts:te]
						if !utf8.ValidString(word) {
							l.errorAt("invalid UTF-8 identifier", ts)
							p++
							goto _out

						}
						tokStart = int32(ts)
						tokEnd = int32(te)
						if kw, ok := LookupKeyword(word); ok {
							tok = kw.Token
							keywordKind = kw.KeywordKind
							if lval != nil {
								lval.setstr(strings.ToLower(word))
							}
						} else {
							tok = IDENT
							keywordKind = KeywordKind_NO_KEYWORD
							if lval != nil {
								lval.setstr(truncateIdentifier(asciiDowncase(word)))
							}
						}
						p++
						goto _out

					}
				case 39:
//line scan.rl:430
					te = p
					p--
					{
						// Check for embedded comment starts: /* or --
						nchars := te - ts
						rawOp := data[ts:te]

						if strings.HasPrefix(rawOp, "/*") {
							depth := 1
							i := ts + 2
							for i < len(data) && depth > 0 {
								if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
									depth++
									i += 2
								} else if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
									depth--
									i += 2
								} else {
									i++
								}
							}
							if depth != 0 {
								l.errorAt("unterminated /* comment", ts)
								p++
								goto _out

							}
							tok = C_COMMENT
							tokStart = int32(ts)
							tokEnd = int32(i)
							p = i - 1
							keywordKind = KeywordKind_NO_KEYWORD
							p++
							goto _out

						}
						if strings.HasPrefix(rawOp, "--") {
							i := ts + 2
							for i < len(data) && data[i] != '\n' && data[i] != '\r' {
								i++
							}
							tok = SQL_COMMENT
							tokStart = int32(ts)
							tokEnd = int32(i)
							p = i - 1
							keywordKind = KeywordKind_NO_KEYWORD
							p++
							goto _out

						}

						slashstar := strings.Index(rawOp, "/*")
						dashdash := strings.Index(rawOp, "--")
						cutIdx := -1
						if slashstar >= 0 && dashdash >= 0 {
							if slashstar < dashdash {
								cutIdx = slashstar
							} else {
								cutIdx = dashdash
							}
						} else if slashstar >= 0 {
							cutIdx = slashstar
						} else if dashdash >= 0 {
							cutIdx = dashdash
						}
						if cutIdx > 0 {
							nchars = cutIdx
							rawOp = rawOp[:nchars]
						}

						// If nchars > 1 and ends with + or - preceded only by SQL ops, strip trailing [+-]
						if nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
							hasSpecial := false
							for ic := nchars - 2; ic >= 0; ic-- {
								c := rawOp[ic]
								if c == '~' || c == '!' || c == '@' || c == '#' || c == '^' ||
									c == '&' || c == '|' || c == '`' || c == '?' || c == '%' {
									hasSpecial = true
									break
								}
							}
							if !hasSpecial {
								for nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
									nchars--
								}
								rawOp = rawOp[:nchars]
							}
						}

						if nchars >= 64 {
							l.Error("operator too long")
							p++
							goto _out

						}
						tokStart = int32(ts)
						tokEnd = int32(ts + nchars)
						p = ts + nchars - 1

						if nchars == 1 && strings.ContainsRune(",()[].;:+-*/%^<>=", rune(rawOp[0])) {
							tok = int(rawOp[0])
						} else if nchars == 2 {
							switch rawOp {
							case "=>":
								tok = EQUALS_GREATER
							case ">=":
								tok = GREATER_EQUALS
							case "<=":
								tok = LESS_EQUALS
							case "<>", "!=":
								tok = NOT_EQUALS
							default:
								tok = Op
								if lval != nil {
									lval.setstr(rawOp)
								}
							}
						} else {
							tok = Op
							if lval != nil {
								lval.setstr(rawOp)
							}
						}
						p++
						goto _out

					}
				case 40:
//line scan.rl:546
					te = p
					p--
					{
						if data[ts] == '"' {
							l.errorAt("unterminated quoted identifier", ts)
							p++
							goto _out

						}
						tok = int(data[ts])
						tokStart = int32(ts)
						tokEnd = int32(ts + 1)
						p++
						goto _out

					}
				case 41:
//line scan.rl:406
					p = (te) - 1
					{
						word := data[ts:te]
						if !utf8.ValidString(word) {
							l.errorAt("invalid UTF-8 identifier", ts)
							p++
							goto _out

						}
						tokStart = int32(ts)
						tokEnd = int32(te)
						if kw, ok := LookupKeyword(word); ok {
							tok = kw.Token
							keywordKind = kw.KeywordKind
							if lval != nil {
								lval.setstr(strings.ToLower(word))
							}
						} else {
							tok = IDENT
							keywordKind = KeywordKind_NO_KEYWORD
							if lval != nil {
								lval.setstr(truncateIdentifier(asciiDowncase(word)))
							}
						}
						p++
						goto _out

					}
				case 42:
//line NONE:1
					switch act {
					case 3:
						{
							p = (te) - 1

							// Handle potentially nested C-style block comments
							depth := 1
							i := ts + 2
							for i < len(data) && depth > 0 {
								if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
									depth++
									i += 2
								} else if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
									depth--
									i += 2
								} else {
									i++
								}
							}
							if depth != 0 {
								l.errorAt("unterminated /* comment", ts)
								p++
								goto _out

							}
							tok = C_COMMENT
							tokStart = int32(ts)
							tokEnd = int32(i)
							p = i - 1
							keywordKind = KeywordKind_NO_KEYWORD
							p++
							goto _out

						}
					case 7:
						{
							p = (te) - 1

							tok = EQUALS_GREATER
							tokStart = int32(ts)
							tokEnd = int32(te)
							p++
							goto _out

						}
					case 8:
						{
							p = (te) - 1

							tok = LESS_EQUALS
							tokStart = int32(ts)
							tokEnd = int32(te)
							p++
							goto _out

						}
					case 9:
						{
							p = (te) - 1

							tok = GREATER_EQUALS
							tokStart = int32(ts)
							tokEnd = int32(te)
							p++
							goto _out

						}
					case 10:
						{
							p = (te) - 1

							tok = NOT_EQUALS
							tokStart = int32(ts)
							tokEnd = int32(te)
							p++
							goto _out

						}
					case 11:
						{
							p = (te) - 1

							tok = PARAM
							tokStart = int32(ts)
							tokEnd = int32(te)
							numStr := data[ts+1 : te]
							n, err := strconv.ParseInt(numStr, 10, 32)
							if err != nil {
								l.errorAt("parameter number too large", ts)
								p++
								goto _out

							}
							if lval != nil {
								lval.setint(n)
							}
							p++
							goto _out

						}
					case 19:
						{
							p = (te) - 1

							l.errorAt("invalid non-decimal integer", ts)
							p++
							goto _out

						}
					case 20:
						{
							p = (te) - 1

							l.errorAt("trailing junk after numeric literal", ts)
							p++
							goto _out

						}
					case 26:
						{
							p = (te) - 1

							tok = UIDENT
							tokStart = int32(ts)
							tokEnd = int32(te)
							if lval != nil {
								lval.setstr(unquoteIdent(data[ts:te]))
							}
							p++
							goto _out

						}
					case 27:
						{
							p = (te) - 1

							value := unquoteIdent(data[ts:te])
							if value == "" {
								l.errorAt("zero-length delimited identifier", ts)
								p++
								goto _out

							}
							if !utf8.ValidString(value) {
								l.errorAt("invalid UTF-8 identifier", ts)
								p++
								goto _out

							}
							tok = IDENT
							tokStart = int32(ts)
							tokEnd = int32(te)
							if lval != nil {
								lval.setstr(truncateIdentifier(value))
							}
							p++
							goto _out

						}
					case 28:
						{
							p = (te) - 1

							word := data[ts:te]
							if !utf8.ValidString(word) {
								l.errorAt("invalid UTF-8 identifier", ts)
								p++
								goto _out

							}
							tokStart = int32(ts)
							tokEnd = int32(te)
							if kw, ok := LookupKeyword(word); ok {
								tok = kw.Token
								keywordKind = kw.KeywordKind
								if lval != nil {
									lval.setstr(strings.ToLower(word))
								}
							} else {
								tok = IDENT
								keywordKind = KeywordKind_NO_KEYWORD
								if lval != nil {
									lval.setstr(truncateIdentifier(asciiDowncase(word)))
								}
							}
							p++
							goto _out

						}
					case 29:
						{
							p = (te) - 1

							// Check for embedded comment starts: /* or --
							nchars := te - ts
							rawOp := data[ts:te]

							if strings.HasPrefix(rawOp, "/*") {
								depth := 1
								i := ts + 2
								for i < len(data) && depth > 0 {
									if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
										depth++
										i += 2
									} else if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
										depth--
										i += 2
									} else {
										i++
									}
								}
								if depth != 0 {
									l.errorAt("unterminated /* comment", ts)
									p++
									goto _out

								}
								tok = C_COMMENT
								tokStart = int32(ts)
								tokEnd = int32(i)
								p = i - 1
								keywordKind = KeywordKind_NO_KEYWORD
								p++
								goto _out

							}
							if strings.HasPrefix(rawOp, "--") {
								i := ts + 2
								for i < len(data) && data[i] != '\n' && data[i] != '\r' {
									i++
								}
								tok = SQL_COMMENT
								tokStart = int32(ts)
								tokEnd = int32(i)
								p = i - 1
								keywordKind = KeywordKind_NO_KEYWORD
								p++
								goto _out

							}

							slashstar := strings.Index(rawOp, "/*")
							dashdash := strings.Index(rawOp, "--")
							cutIdx := -1
							if slashstar >= 0 && dashdash >= 0 {
								if slashstar < dashdash {
									cutIdx = slashstar
								} else {
									cutIdx = dashdash
								}
							} else if slashstar >= 0 {
								cutIdx = slashstar
							} else if dashdash >= 0 {
								cutIdx = dashdash
							}
							if cutIdx > 0 {
								nchars = cutIdx
								rawOp = rawOp[:nchars]
							}

							// If nchars > 1 and ends with + or - preceded only by SQL ops, strip trailing [+-]
							if nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
								hasSpecial := false
								for ic := nchars - 2; ic >= 0; ic-- {
									c := rawOp[ic]
									if c == '~' || c == '!' || c == '@' || c == '#' || c == '^' ||
										c == '&' || c == '|' || c == '`' || c == '?' || c == '%' {
										hasSpecial = true
										break
									}
								}
								if !hasSpecial {
									for nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
										nchars--
									}
									rawOp = rawOp[:nchars]
								}
							}

							if nchars >= 64 {
								l.Error("operator too long")
								p++
								goto _out

							}
							tokStart = int32(ts)
							tokEnd = int32(ts + nchars)
							p = ts + nchars - 1

							if nchars == 1 && strings.ContainsRune(",()[].;:+-*/%^<>=", rune(rawOp[0])) {
								tok = int(rawOp[0])
							} else if nchars == 2 {
								switch rawOp {
								case "=>":
									tok = EQUALS_GREATER
								case ">=":
									tok = GREATER_EQUALS
								case "<=":
									tok = LESS_EQUALS
								case "<>", "!=":
									tok = NOT_EQUALS
								default:
									tok = Op
									if lval != nil {
										lval.setstr(rawOp)
									}
								}
							} else {
								tok = Op
								if lval != nil {
									lval.setstr(rawOp)
								}
							}
							p++
							goto _out

						}
					case 30:
						{
							p = (te) - 1

							if data[ts] == '"' {
								l.errorAt("unterminated quoted identifier", ts)
								p++
								goto _out

							}
							tok = int(data[ts])
							tokStart = int32(ts)
							tokEnd = int32(ts + 1)
							p++
							goto _out

						}
					}

//line scan.go:1351
				}
			}

		_again:
			_acts = int(_pg_scanner_to_state_actions[cs])
			_nacts = uint(_pg_scanner_actions[_acts])
			_acts++
			for ; _nacts > 0; _nacts-- {
				_acts++
				switch _pg_scanner_actions[_acts-1] {
				case 0:
//line NONE:1
					ts = 0

//line scan.go:1365
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
				if _pg_scanner_eof_trans[cs] > 0 {
					_trans = int(_pg_scanner_eof_trans[cs] - 1)
					goto _eof_trans
				}
			}

		_out:
			{
			}
		}

//line scan.rl:624

		l.p = p
		l.cs = cs
		l.ts = ts
		l.te = te
		l.act = act

		if l.err != nil {
			return 0
		}
		if tok != 0 {
			lval.yyloc = int(tokStart)
			// In parser mode, ignore SQL comments and C comments
			if !l.scanMode && (tok == SQL_COMMENT || tok == C_COMMENT) {
				continue
			}
			return tok
		}

		if p <= oldP {
			p = oldP + 1
			l.p = p
		}
	}

	l.p = p
	lval.yyloc = len(data)
	l.cs = cs
	return 0
}

func (l *Lexer) Lex(lval *yySymType) int {
	if l.initialToken != 0 {
		token := l.initialToken
		l.initialToken = 0
		lval.yyloc = 0
		return token
	}
	var curTok int
	if l.haveLookahead {
		curTok = l.lookaheadTok
		if lval != nil {
			*lval = l.lookaheadVal
		}
		l.haveLookahead = false
	} else {
		curTok = l.rawLex(lval)
	}
	l.tokenStart = lval.yyloc
	l.tokenEnd = l.te

	if curTok == 0 {
		return 0
	}

	if l.scanMode {
		return curTok
	}

	switch curTok {
	case NOT:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case BETWEEN, IN_P, LIKE, ILIKE, SIMILAR:
			return NOT_LA
		}

	case NULLS_P:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case FIRST_P, LAST_P:
			return NULLS_LA
		}

	case WITH:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case TIME, ORDINALITY:
			return WITH_LA
		}

	case WITHOUT:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case TIME:
			return WITHOUT_LA
		}

	case FORMAT:
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		l.haveLookahead = true
		l.lookaheadTok = nextTok
		l.lookaheadVal = nextVal

		switch nextTok {
		case JSON:
			return FORMAT_LA
		}

	case USCONST, UIDENT:
		escape := byte('\\')
		var nextVal yySymType
		nextTok := l.rawLex(&nextVal)
		if nextTok == UESCAPE {
			var escVal yySymType
			if l.rawLex(&escVal) != SCONST {
				l.Error("UESCAPE must be followed by a simple string literal")
				return 0
			}
			esc := escVal.str()
			if len(esc) != 1 || isUnicodeHexDigit(esc[0]) || strings.ContainsRune("+\"' \t\n\r\f\v", rune(esc[0])) {
				l.Error("invalid Unicode escape character")
				return 0
			}
			escape = esc[0]
		} else {
			l.haveLookahead = true
			l.lookaheadTok = nextTok
			l.lookaheadVal = nextVal
		}
		decoded, err := decodeUnicodeEscapes(lval.str(), escape)
		if err != nil {
			l.Error(err.Error())
			return 0
		}
		if curTok == UIDENT && decoded == "" {
			l.Error("zero-length delimited identifier")
			return 0
		}
		if curTok == UIDENT {
			decoded = truncateIdentifier(decoded)
		}
		lval.setstr(decoded)
		if curTok == UIDENT {
			return IDENT
		}
		return SCONST
	}

	return curTok
}

func (l *Lexer) Error(s string) {
	l.errorAt(s, l.tokenStart)
}

func (l *Lexer) errorAt(message string, location int) {
	if l.err != nil {
		return
	}
	location = max(0, min(location, len(l.data)))
	filename, funcname := "gram.y", "parser_yyerror"
	if message == "syntax error" || strings.HasPrefix(message, "syntax error:") {
		filename, funcname = "scan.l", "scanner_yyerror"
		if location == len(l.data) {
			message = "syntax error at end of input"
		} else {
			end := max(location+1, min(l.tokenEnd, len(l.data)))
			message = fmt.Sprintf("syntax error at or near %q", l.data[location:end])
		}
	}
	l.err = &Error{
		Message:   message,
		Funcname:  funcname,
		Filename:  filename,
		Cursorpos: utf8.RuneCountInString(l.data[:location]) + 1,
		Location:  location,
	}
}

// Scan parses the SQL string into tokens matching PostgreSQL's scan.l and pg_query.Scan.
func Scan(input string) (*ScanResult, error) {
	l := NewLexer(input)
	l.scanMode = true

	data := l.data
	p := l.p
	pe := l.pe
	eof := l.eof
	cs := l.cs
	ts := l.ts
	te := l.te
	act := l.act

	var tokens []*ScanToken
	var lval *yySymType

	for p < pe {
		tok := 0
		var tokStart, tokEnd int32
		var keywordKind KeywordKind

//line scan.go:1599
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
			_acts = int(_pg_scanner_from_state_actions[cs])
			_nacts = uint(_pg_scanner_actions[_acts])
			_acts++
			for ; _nacts > 0; _nacts-- {
				_acts++
				switch _pg_scanner_actions[_acts-1] {
				case 1:
//line NONE:1
					ts = p

//line scan.go:1619
				}
			}

			_keys = int(_pg_scanner_key_offsets[cs])
			_trans = int(_pg_scanner_index_offsets[cs])

			_klen = int(_pg_scanner_single_lengths[cs])
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
					case data[p] < _pg_scanner_trans_keys[_mid]:
						_upper = _mid - 1
					case data[p] > _pg_scanner_trans_keys[_mid]:
						_lower = _mid + 1
					default:
						_trans += int(_mid - int(_keys))
						goto _match
					}
				}
				_keys += _klen
				_trans += _klen
			}

			_klen = int(_pg_scanner_range_lengths[cs])
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
					case data[p] < _pg_scanner_trans_keys[_mid]:
						_upper = _mid - 2
					case data[p] > _pg_scanner_trans_keys[_mid+1]:
						_lower = _mid + 2
					default:
						_trans += int((_mid - int(_keys)) >> 1)
						goto _match
					}
				}
				_trans += _klen
			}

		_match:
			_trans = int(_pg_scanner_indicies[_trans])
		_eof_trans:
			cs = int(_pg_scanner_trans_targs[_trans])

			if _pg_scanner_trans_actions[_trans] == 0 {
				goto _again
			}

			_acts = int(_pg_scanner_trans_actions[_trans])
			_nacts = uint(_pg_scanner_actions[_acts])
			_acts++
			for ; _nacts > 0; _nacts-- {
				_acts++
				switch _pg_scanner_actions[_acts-1] {
				case 2:
//line NONE:1
					te = p + 1

				case 3:
//line scan.rl:67
					act = 3
				case 4:
//line scan.rl:115
					act = 7
				case 5:
//line scan.rl:122
					act = 8
				case 6:
//line scan.rl:129
					act = 9
				case 7:
//line scan.rl:136
					act = 10
				case 8:
//line scan.rl:143
					act = 11
				case 9:
//line scan.rl:273
					act = 19
				case 10:
//line scan.rl:278
					act = 20
				case 11:
//line scan.rl:377
					act = 26
				case 12:
//line scan.rl:387
					act = 27
				case 13:
//line scan.rl:406
					act = 28
				case 14:
//line scan.rl:430
					act = 29
				case 15:
//line scan.rl:546
					act = 30
				case 16:
//line scan.rl:94
					te = p + 1
					{
						tok = TYPECAST
						tokStart = int32(ts)
						tokEnd = int32(te)
						p++
						goto _out

					}
				case 17:
//line scan.rl:101
					te = p + 1
					{
						tok = DOT_DOT
						tokStart = int32(ts)
						tokEnd = int32(te)
						p++
						goto _out

					}
				case 18:
//line scan.rl:108
					te = p + 1
					{
						tok = COLON_EQUALS
						tokStart = int32(ts)
						tokEnd = int32(te)
						p++
						goto _out

					}
				case 19:
//line scan.rl:159
					te = p + 1
					{
						// Integer followed by '..'; emit integer and rewind 2 bytes
						tokStart = int32(ts)
						tokEnd = int32(te - 2)
						p = te - 3
						raw := strings.ReplaceAll(data[ts:te-2], "_", "")
						if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts : te-2])
							}
						}
						p++
						goto _out

					}
				case 20:
//line scan.rl:284
					te = p + 1
					{
						tok = NCHAR
						tokStart = int32(ts)
						tokEnd = int32(ts + 1)
						keywordKind = KeywordKind_COL_NAME_KEYWORD
						p = ts
						if lval != nil {
							lval.setstr("nchar")
						}
						p++
						goto _out

					}
				case 21:
//line scan.rl:296
					te = p + 1
					{
						tok = BCONST
						prefix := "b"
						if data[ts] == 'x' || data[ts] == 'X' {
							tok = XCONST
							prefix = "x"
						}
						value, end, err := scanQuotedLiteral(data, te-1, literalBits)
						if err != nil {
							l.errorAt(err.Error(), ts)
							p++
							goto _out

						}
						tokStart = int32(ts)
						tokEnd = int32(end)
						p = end - 1
						if lval != nil {
							lval.setstr(prefix + value)
						}
						p++
						goto _out

					}
				case 22:
//line scan.rl:317
					te = p + 1
					{
						value, end, err := scanQuotedLiteral(data, te-1, literalStandard)
						if err != nil {
							l.errorAt(err.Error(), ts)
							p++
							goto _out

						}
						tok = USCONST
						tokStart = int32(ts)
						tokEnd = int32(end)
						p = end - 1
						if lval != nil {
							lval.setstr(value)
						}
						p++
						goto _out

					}
				case 23:
//line scan.rl:334
					te = p + 1
					{
						tag := data[ts:te]
						// search for closing tag
						endIdx := strings.Index(data[te:], tag)
						if endIdx >= 0 {
							fullEnd := te + endIdx + len(tag)
							if !utf8.ValidString(data[te : te+endIdx]) {
								l.errorAt("invalid UTF-8 string", ts)
								p++
								goto _out

							}
							tok = SCONST
							tokStart = int32(ts)
							tokEnd = int32(fullEnd)
							p = fullEnd - 1
							if lval != nil {
								lval.setstr(data[te : te+endIdx])
							}
							p++
							goto _out

						}
						l.errorAt("unterminated dollar-quoted string", ts)
						p++
						goto _out

					}
				case 24:
//line scan.rl:357
					te = p + 1
					{
						mode := literalStandard
						if data[ts] != '\'' {
							mode = literalEscape
						}
						value, end, err := scanQuotedLiteral(data, te-1, mode)
						if err != nil {
							l.errorAt(err.Error(), ts)
							p++
							goto _out

						}
						tok = SCONST
						tokStart = int32(ts)
						tokEnd = int32(end)
						p = end - 1
						if lval != nil {
							lval.setstr(value)
						}
						p++
						goto _out

					}
				case 25:
//line scan.rl:546
					te = p + 1
					{
						if data[ts] == '"' {
							l.errorAt("unterminated quoted identifier", ts)
							p++
							goto _out

						}
						tok = int(data[ts])
						tokStart = int32(ts)
						tokEnd = int32(ts + 1)
						p++
						goto _out

					}
				case 26:
//line scan.rl:55
					te = p
					p--
					{
						// whitespace ignored
					}
				case 27:
//line scan.rl:59
					te = p
					p--
					{
						tok = SQL_COMMENT
						tokStart = int32(ts)
						tokEnd = int32(te)
						keywordKind = KeywordKind_NO_KEYWORD
						p++
						goto _out

					}
				case 28:
//line scan.rl:179
					te = p
					p--
					{
						tok = FCONST
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := data[ts:te]
						if lval != nil {
							lval.setstr(raw)
						}
						p++
						goto _out

					}
				case 29:
//line scan.rl:190
					te = p
					p--
					{
						tok = FCONST
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := data[ts:te]
						if lval != nil {
							lval.setstr(raw)
						}
						p++
						goto _out

					}
				case 30:
//line scan.rl:201
					te = p
					p--
					{
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := strings.ReplaceAll(data[ts+2:te], "_", "")
						if n, err := strconv.ParseInt(raw, 16, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts:te])
							}
						}
						p++
						goto _out

					}
				case 31:
//line scan.rl:219
					te = p
					p--
					{
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := strings.ReplaceAll(data[ts+2:te], "_", "")
						if n, err := strconv.ParseInt(raw, 8, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts:te])
							}
						}
						p++
						goto _out

					}
				case 32:
//line scan.rl:237
					te = p
					p--
					{
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := strings.ReplaceAll(data[ts+2:te], "_", "")
						if n, err := strconv.ParseInt(raw, 2, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts:te])
							}
						}
						p++
						goto _out

					}
				case 33:
//line scan.rl:255
					te = p
					p--
					{
						tokStart = int32(ts)
						tokEnd = int32(te)
						raw := strings.ReplaceAll(data[ts:te], "_", "")
						if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
							tok = ICONST
							if lval != nil {
								lval.setint(n)
							}
						} else {
							tok = FCONST
							if lval != nil {
								lval.setstr(data[ts:te])
							}
						}
						p++
						goto _out

					}
				case 34:
//line scan.rl:273
					te = p
					p--
					{
						l.errorAt("invalid non-decimal integer", ts)
						p++
						goto _out

					}
				case 35:
//line scan.rl:278
					te = p
					p--
					{
						l.errorAt("trailing junk after numeric literal", ts)
						p++
						goto _out

					}
				case 36:
//line scan.rl:377
					te = p
					p--
					{
						tok = UIDENT
						tokStart = int32(ts)
						tokEnd = int32(te)
						if lval != nil {
							lval.setstr(unquoteIdent(data[ts:te]))
						}
						p++
						goto _out

					}
				case 37:
//line scan.rl:387
					te = p
					p--
					{
						value := unquoteIdent(data[ts:te])
						if value == "" {
							l.errorAt("zero-length delimited identifier", ts)
							p++
							goto _out

						}
						if !utf8.ValidString(value) {
							l.errorAt("invalid UTF-8 identifier", ts)
							p++
							goto _out

						}
						tok = IDENT
						tokStart = int32(ts)
						tokEnd = int32(te)
						if lval != nil {
							lval.setstr(truncateIdentifier(value))
						}
						p++
						goto _out

					}
				case 38:
//line scan.rl:406
					te = p
					p--
					{
						word := data[ts:te]
						if !utf8.ValidString(word) {
							l.errorAt("invalid UTF-8 identifier", ts)
							p++
							goto _out

						}
						tokStart = int32(ts)
						tokEnd = int32(te)
						if kw, ok := LookupKeyword(word); ok {
							tok = kw.Token
							keywordKind = kw.KeywordKind
							if lval != nil {
								lval.setstr(strings.ToLower(word))
							}
						} else {
							tok = IDENT
							keywordKind = KeywordKind_NO_KEYWORD
							if lval != nil {
								lval.setstr(truncateIdentifier(asciiDowncase(word)))
							}
						}
						p++
						goto _out

					}
				case 39:
//line scan.rl:430
					te = p
					p--
					{
						// Check for embedded comment starts: /* or --
						nchars := te - ts
						rawOp := data[ts:te]

						if strings.HasPrefix(rawOp, "/*") {
							depth := 1
							i := ts + 2
							for i < len(data) && depth > 0 {
								if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
									depth++
									i += 2
								} else if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
									depth--
									i += 2
								} else {
									i++
								}
							}
							if depth != 0 {
								l.errorAt("unterminated /* comment", ts)
								p++
								goto _out

							}
							tok = C_COMMENT
							tokStart = int32(ts)
							tokEnd = int32(i)
							p = i - 1
							keywordKind = KeywordKind_NO_KEYWORD
							p++
							goto _out

						}
						if strings.HasPrefix(rawOp, "--") {
							i := ts + 2
							for i < len(data) && data[i] != '\n' && data[i] != '\r' {
								i++
							}
							tok = SQL_COMMENT
							tokStart = int32(ts)
							tokEnd = int32(i)
							p = i - 1
							keywordKind = KeywordKind_NO_KEYWORD
							p++
							goto _out

						}

						slashstar := strings.Index(rawOp, "/*")
						dashdash := strings.Index(rawOp, "--")
						cutIdx := -1
						if slashstar >= 0 && dashdash >= 0 {
							if slashstar < dashdash {
								cutIdx = slashstar
							} else {
								cutIdx = dashdash
							}
						} else if slashstar >= 0 {
							cutIdx = slashstar
						} else if dashdash >= 0 {
							cutIdx = dashdash
						}
						if cutIdx > 0 {
							nchars = cutIdx
							rawOp = rawOp[:nchars]
						}

						// If nchars > 1 and ends with + or - preceded only by SQL ops, strip trailing [+-]
						if nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
							hasSpecial := false
							for ic := nchars - 2; ic >= 0; ic-- {
								c := rawOp[ic]
								if c == '~' || c == '!' || c == '@' || c == '#' || c == '^' ||
									c == '&' || c == '|' || c == '`' || c == '?' || c == '%' {
									hasSpecial = true
									break
								}
							}
							if !hasSpecial {
								for nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
									nchars--
								}
								rawOp = rawOp[:nchars]
							}
						}

						if nchars >= 64 {
							l.Error("operator too long")
							p++
							goto _out

						}
						tokStart = int32(ts)
						tokEnd = int32(ts + nchars)
						p = ts + nchars - 1

						if nchars == 1 && strings.ContainsRune(",()[].;:+-*/%^<>=", rune(rawOp[0])) {
							tok = int(rawOp[0])
						} else if nchars == 2 {
							switch rawOp {
							case "=>":
								tok = EQUALS_GREATER
							case ">=":
								tok = GREATER_EQUALS
							case "<=":
								tok = LESS_EQUALS
							case "<>", "!=":
								tok = NOT_EQUALS
							default:
								tok = Op
								if lval != nil {
									lval.setstr(rawOp)
								}
							}
						} else {
							tok = Op
							if lval != nil {
								lval.setstr(rawOp)
							}
						}
						p++
						goto _out

					}
				case 40:
//line scan.rl:546
					te = p
					p--
					{
						if data[ts] == '"' {
							l.errorAt("unterminated quoted identifier", ts)
							p++
							goto _out

						}
						tok = int(data[ts])
						tokStart = int32(ts)
						tokEnd = int32(ts + 1)
						p++
						goto _out

					}
				case 41:
//line scan.rl:406
					p = (te) - 1
					{
						word := data[ts:te]
						if !utf8.ValidString(word) {
							l.errorAt("invalid UTF-8 identifier", ts)
							p++
							goto _out

						}
						tokStart = int32(ts)
						tokEnd = int32(te)
						if kw, ok := LookupKeyword(word); ok {
							tok = kw.Token
							keywordKind = kw.KeywordKind
							if lval != nil {
								lval.setstr(strings.ToLower(word))
							}
						} else {
							tok = IDENT
							keywordKind = KeywordKind_NO_KEYWORD
							if lval != nil {
								lval.setstr(truncateIdentifier(asciiDowncase(word)))
							}
						}
						p++
						goto _out

					}
				case 42:
//line NONE:1
					switch act {
					case 3:
						{
							p = (te) - 1

							// Handle potentially nested C-style block comments
							depth := 1
							i := ts + 2
							for i < len(data) && depth > 0 {
								if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
									depth++
									i += 2
								} else if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
									depth--
									i += 2
								} else {
									i++
								}
							}
							if depth != 0 {
								l.errorAt("unterminated /* comment", ts)
								p++
								goto _out

							}
							tok = C_COMMENT
							tokStart = int32(ts)
							tokEnd = int32(i)
							p = i - 1
							keywordKind = KeywordKind_NO_KEYWORD
							p++
							goto _out

						}
					case 7:
						{
							p = (te) - 1

							tok = EQUALS_GREATER
							tokStart = int32(ts)
							tokEnd = int32(te)
							p++
							goto _out

						}
					case 8:
						{
							p = (te) - 1

							tok = LESS_EQUALS
							tokStart = int32(ts)
							tokEnd = int32(te)
							p++
							goto _out

						}
					case 9:
						{
							p = (te) - 1

							tok = GREATER_EQUALS
							tokStart = int32(ts)
							tokEnd = int32(te)
							p++
							goto _out

						}
					case 10:
						{
							p = (te) - 1

							tok = NOT_EQUALS
							tokStart = int32(ts)
							tokEnd = int32(te)
							p++
							goto _out

						}
					case 11:
						{
							p = (te) - 1

							tok = PARAM
							tokStart = int32(ts)
							tokEnd = int32(te)
							numStr := data[ts+1 : te]
							n, err := strconv.ParseInt(numStr, 10, 32)
							if err != nil {
								l.errorAt("parameter number too large", ts)
								p++
								goto _out

							}
							if lval != nil {
								lval.setint(n)
							}
							p++
							goto _out

						}
					case 19:
						{
							p = (te) - 1

							l.errorAt("invalid non-decimal integer", ts)
							p++
							goto _out

						}
					case 20:
						{
							p = (te) - 1

							l.errorAt("trailing junk after numeric literal", ts)
							p++
							goto _out

						}
					case 26:
						{
							p = (te) - 1

							tok = UIDENT
							tokStart = int32(ts)
							tokEnd = int32(te)
							if lval != nil {
								lval.setstr(unquoteIdent(data[ts:te]))
							}
							p++
							goto _out

						}
					case 27:
						{
							p = (te) - 1

							value := unquoteIdent(data[ts:te])
							if value == "" {
								l.errorAt("zero-length delimited identifier", ts)
								p++
								goto _out

							}
							if !utf8.ValidString(value) {
								l.errorAt("invalid UTF-8 identifier", ts)
								p++
								goto _out

							}
							tok = IDENT
							tokStart = int32(ts)
							tokEnd = int32(te)
							if lval != nil {
								lval.setstr(truncateIdentifier(value))
							}
							p++
							goto _out

						}
					case 28:
						{
							p = (te) - 1

							word := data[ts:te]
							if !utf8.ValidString(word) {
								l.errorAt("invalid UTF-8 identifier", ts)
								p++
								goto _out

							}
							tokStart = int32(ts)
							tokEnd = int32(te)
							if kw, ok := LookupKeyword(word); ok {
								tok = kw.Token
								keywordKind = kw.KeywordKind
								if lval != nil {
									lval.setstr(strings.ToLower(word))
								}
							} else {
								tok = IDENT
								keywordKind = KeywordKind_NO_KEYWORD
								if lval != nil {
									lval.setstr(truncateIdentifier(asciiDowncase(word)))
								}
							}
							p++
							goto _out

						}
					case 29:
						{
							p = (te) - 1

							// Check for embedded comment starts: /* or --
							nchars := te - ts
							rawOp := data[ts:te]

							if strings.HasPrefix(rawOp, "/*") {
								depth := 1
								i := ts + 2
								for i < len(data) && depth > 0 {
									if i+1 < len(data) && data[i] == '/' && data[i+1] == '*' {
										depth++
										i += 2
									} else if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
										depth--
										i += 2
									} else {
										i++
									}
								}
								if depth != 0 {
									l.errorAt("unterminated /* comment", ts)
									p++
									goto _out

								}
								tok = C_COMMENT
								tokStart = int32(ts)
								tokEnd = int32(i)
								p = i - 1
								keywordKind = KeywordKind_NO_KEYWORD
								p++
								goto _out

							}
							if strings.HasPrefix(rawOp, "--") {
								i := ts + 2
								for i < len(data) && data[i] != '\n' && data[i] != '\r' {
									i++
								}
								tok = SQL_COMMENT
								tokStart = int32(ts)
								tokEnd = int32(i)
								p = i - 1
								keywordKind = KeywordKind_NO_KEYWORD
								p++
								goto _out

							}

							slashstar := strings.Index(rawOp, "/*")
							dashdash := strings.Index(rawOp, "--")
							cutIdx := -1
							if slashstar >= 0 && dashdash >= 0 {
								if slashstar < dashdash {
									cutIdx = slashstar
								} else {
									cutIdx = dashdash
								}
							} else if slashstar >= 0 {
								cutIdx = slashstar
							} else if dashdash >= 0 {
								cutIdx = dashdash
							}
							if cutIdx > 0 {
								nchars = cutIdx
								rawOp = rawOp[:nchars]
							}

							// If nchars > 1 and ends with + or - preceded only by SQL ops, strip trailing [+-]
							if nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
								hasSpecial := false
								for ic := nchars - 2; ic >= 0; ic-- {
									c := rawOp[ic]
									if c == '~' || c == '!' || c == '@' || c == '#' || c == '^' ||
										c == '&' || c == '|' || c == '`' || c == '?' || c == '%' {
										hasSpecial = true
										break
									}
								}
								if !hasSpecial {
									for nchars > 1 && (rawOp[nchars-1] == '+' || rawOp[nchars-1] == '-') {
										nchars--
									}
									rawOp = rawOp[:nchars]
								}
							}

							if nchars >= 64 {
								l.Error("operator too long")
								p++
								goto _out

							}
							tokStart = int32(ts)
							tokEnd = int32(ts + nchars)
							p = ts + nchars - 1

							if nchars == 1 && strings.ContainsRune(",()[].;:+-*/%^<>=", rune(rawOp[0])) {
								tok = int(rawOp[0])
							} else if nchars == 2 {
								switch rawOp {
								case "=>":
									tok = EQUALS_GREATER
								case ">=":
									tok = GREATER_EQUALS
								case "<=":
									tok = LESS_EQUALS
								case "<>", "!=":
									tok = NOT_EQUALS
								default:
									tok = Op
									if lval != nil {
										lval.setstr(rawOp)
									}
								}
							} else {
								tok = Op
								if lval != nil {
									lval.setstr(rawOp)
								}
							}
							p++
							goto _out

						}
					case 30:
						{
							p = (te) - 1

							if data[ts] == '"' {
								l.errorAt("unterminated quoted identifier", ts)
								p++
								goto _out

							}
							tok = int(data[ts])
							tokStart = int32(ts)
							tokEnd = int32(ts + 1)
							p++
							goto _out

						}
					}

//line scan.go:2605
				}
			}

		_again:
			_acts = int(_pg_scanner_to_state_actions[cs])
			_nacts = uint(_pg_scanner_actions[_acts])
			_acts++
			for ; _nacts > 0; _nacts-- {
				_acts++
				switch _pg_scanner_actions[_acts-1] {
				case 0:
//line NONE:1
					ts = 0

//line scan.go:2619
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
				if _pg_scanner_eof_trans[cs] > 0 {
					_trans = int(_pg_scanner_eof_trans[cs] - 1)
					goto _eof_trans
				}
			}

		_out:
			{
			}
		}

//line scan.rl:838

		l.p = p
		l.cs = cs
		l.ts = ts
		l.te = te
		l.act = act

		if l.err != nil {
			return nil, l.err
		}
		if tok != 0 {
			tokens = append(tokens, &ScanToken{
				Start:       tokStart,
				End:         tokEnd,
				Token:       pg_query.Token(tok),
				KeywordKind: pg_query.KeywordKind(keywordKind),
			})
		}
	}

	return &ScanResult{
		Version: 180004,
		Tokens:  tokens,
	}, nil
}

func unquoteIdent(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return strings.ReplaceAll(s[1:len(s)-1], `""`, `"`)
	}
	if len(s) >= 4 && (s[0] == 'u' || s[0] == 'U') && s[1] == '&' && s[2] == '"' && s[len(s)-1] == '"' {
		return strings.ReplaceAll(s[3:len(s)-1], `""`, `"`)
	}
	return s
}

var (
	errInvalidUnicodeEscape = errors.New("invalid Unicode escape")
	errInvalidUnicodeValue  = errors.New("invalid Unicode escape value")
	errInvalidUnicodePair   = errors.New("invalid Unicode surrogate pair")
)

func isUnicodeHexDigit(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func decodeUnicodeEscapes(s string, escape byte) (string, error) {
	if !utf8.ValidString(s) || strings.IndexByte(s, 0) >= 0 {
		return "", errInvalidUnicodeValue
	}
	if strings.IndexByte(s, escape) < 0 {
		return s, nil
	}
	var decoded strings.Builder
	decoded.Grow(len(s))
	var highSurrogate rune
	for i := 0; i < len(s); {
		if s[i] != escape {
			if highSurrogate != 0 {
				return "", errInvalidUnicodePair
			}
			decoded.WriteByte(s[i])
			i++
			continue
		}
		i++
		if i < len(s) && s[i] == escape {
			if highSurrogate != 0 {
				return "", errInvalidUnicodePair
			}
			decoded.WriteByte(escape)
			i++
			continue
		}
		digits := 4
		if i < len(s) && s[i] == '+' {
			digits = 6
			i++
		}
		if len(s)-i < digits {
			return "", errInvalidUnicodeEscape
		}
		var value rune
		for j := range digits {
			c := s[i+j]
			if !isUnicodeHexDigit(c) {
				return "", errInvalidUnicodeEscape
			}
			value <<= 4
			switch {
			case c <= '9':
				value += rune(c - '0')
			case c <= 'F':
				value += rune(c - 'A' + 10)
			default:
				value += rune(c - 'a' + 10)
			}
		}
		i += digits
		if value == 0 || value > utf8.MaxRune {
			return "", errInvalidUnicodeValue
		}
		if highSurrogate != 0 {
			if value < 0xDC00 || value > 0xDFFF {
				return "", errInvalidUnicodePair
			}
			value = utf16.DecodeRune(highSurrogate, value)
			highSurrogate = 0
		} else if value >= 0xD800 && value <= 0xDBFF {
			highSurrogate = value
			continue
		}
		if value >= 0xDC00 && value <= 0xDFFF {
			return "", errInvalidUnicodePair
		}
		decoded.WriteRune(value)
	}
	if highSurrogate != 0 {
		return "", errInvalidUnicodePair
	}
	return decoded.String(), nil
}

type literalMode uint8

const (
	literalStandard literalMode = iota
	literalEscape
	literalBits
)

var (
	errUnterminatedString = errors.New("unterminated quoted string")
	errInvalidString      = errors.New("invalid UTF-8 or zero byte in string")
)

func scanQuotedLiteral(data string, quote int, mode literalMode) (string, int, error) {
	var value strings.Builder
	segment := quote + 1
	for i := segment; i < len(data); {
		if data[i] == '\'' {
			if mode != literalBits && i+1 < len(data) && data[i+1] == '\'' {
				value.WriteString(data[segment : i+1])
				i += 2
				segment = i
				continue
			}
			if next := continuedQuote(data, i+1); next >= 0 {
				value.WriteString(data[segment:i])
				i = next + 1
				segment = i
				continue
			}
			text := data[segment:i]
			if value.Len() != 0 {
				value.WriteString(text)
				text = value.String()
			}
			if !utf8.ValidString(text) || strings.IndexByte(text, 0) >= 0 {
				return "", i + 1, errInvalidString
			}
			return text, i + 1, nil
		}
		if mode == literalEscape && data[i] == '\\' {
			value.WriteString(data[segment:i])
			next, err := appendStringEscape(&value, data, i)
			if err != nil {
				return "", next, err
			}
			i = next
			segment = i
			continue
		}
		i++
	}
	return "", len(data), errUnterminatedString
}

func continuedQuote(data string, pos int) int {
	hasNewline := false
	for pos < len(data) {
		switch data[pos] {
		case ' ', '\t', '\f', '\v':
			pos++
		case '\n', '\r':
			hasNewline = true
			pos++
		case '-':
			if pos+1 >= len(data) || data[pos+1] != '-' {
				return -1
			}
			pos += 2
			for pos < len(data) && data[pos] != '\n' && data[pos] != '\r' {
				pos++
			}
		default:
			if hasNewline && data[pos] == '\'' {
				return pos
			}
			return -1
		}
	}
	return -1
}

func appendStringEscape(dst *strings.Builder, data string, slash int) (int, error) {
	pos := slash + 1
	if pos >= len(data) {
		return pos, errUnterminatedString
	}
	c := data[pos]
	pos++
	if c == 'u' || c == 'U' {
		value, next, err := stringUnicodeEscape(data, slash)
		if err != nil {
			return next, err
		}
		if value >= 0xD800 && value <= 0xDBFF {
			if next+1 >= len(data) || data[next] != '\\' || data[next+1] != 'u' && data[next+1] != 'U' {
				return next, errInvalidUnicodePair
			}
			second, end, err := stringUnicodeEscape(data, next)
			if err != nil {
				return end, err
			}
			if second < 0xDC00 || second > 0xDFFF {
				return end, errInvalidUnicodePair
			}
			value = utf16.DecodeRune(value, second)
			next = end
		}
		if value >= 0xDC00 && value <= 0xDFFF {
			return next, errInvalidUnicodePair
		}
		dst.WriteRune(value)
		return next, nil
	}
	if c >= '0' && c <= '7' {
		value := c - '0'
		for count := 1; count < 3 && pos < len(data) && data[pos] >= '0' && data[pos] <= '7'; count++ {
			value = value*8 + data[pos] - '0'
			pos++
		}
		dst.WriteByte(value)
		return pos, nil
	}
	if c == 'x' && pos < len(data) && isUnicodeHexDigit(data[pos]) {
		start := pos
		pos++
		if pos < len(data) && isUnicodeHexDigit(data[pos]) {
			pos++
		}
		value, err := strconv.ParseUint(data[start:pos], 16, 8)
		if err != nil {
			return pos, err
		}
		dst.WriteByte(byte(value))
		return pos, nil
	}
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
	}
	dst.WriteByte(c)
	return pos, nil
}

func stringUnicodeEscape(data string, slash int) (rune, int, error) {
	digits := 4
	if data[slash+1] == 'U' {
		digits = 8
	}
	end := slash + 2 + digits
	if end > len(data) {
		return 0, len(data), errInvalidUnicodeEscape
	}
	for i := slash + 2; i < end; i++ {
		if !isUnicodeHexDigit(data[i]) {
			return 0, i, errInvalidUnicodeEscape
		}
	}
	value, err := strconv.ParseUint(data[slash+2:end], 16, 32)
	if err != nil || value == 0 || value > utf8.MaxRune {
		return 0, end, errInvalidUnicodeValue
	}
	return rune(value), end, nil
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

func asciiDowncase(value string) string {
	first := strings.IndexFunc(value, func(c rune) bool { return c >= 'A' && c <= 'Z' })
	if first < 0 {
		return value
	}
	lower := []byte(value)
	for i := first; i < len(lower); i++ {
		if lower[i] >= 'A' && lower[i] <= 'Z' {
			lower[i] += 'a' - 'A'
		}
	}
	return string(lower)
}
