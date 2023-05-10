
//line response_parser.rl:1
package http

import (
	"fmt"
)


//line response_parser.go:11
var _fsm_response_parser_actions []byte = []byte{
	0, 1, 1, 1, 2, 1, 3, 1, 4, 
	1, 5, 1, 6, 1, 7, 1, 8, 
	1, 10, 1, 11, 2, 0, 2, 2, 
	0, 3, 2, 0, 11, 2, 4, 5, 
	2, 6, 7, 2, 6, 8, 2, 9, 
	0, 
}

var _fsm_response_parser_key_offsets []byte = []byte{
	0, 0, 1, 2, 3, 4, 5, 7, 
	11, 14, 17, 20, 22, 25, 28, 30, 
	31, 31, 32, 33, 36, 38, 41, 44, 
}

var _fsm_response_parser_trans_keys []byte = []byte{
	72, 84, 84, 80, 47, 48, 57, 32, 
	46, 48, 57, 32, 48, 57, 32, 48, 
	57, 10, 13, 32, 10, 13, 10, 13, 
	58, 10, 13, 32, 10, 13, 10, 10, 
	10, 10, 13, 32, 48, 57, 32, 48, 
	57, 10, 13, 58, 
}

var _fsm_response_parser_single_lengths []byte = []byte{
	0, 1, 1, 1, 1, 1, 0, 2, 
	1, 1, 3, 2, 3, 3, 2, 1, 
	0, 1, 1, 3, 0, 1, 3, 0, 
}

var _fsm_response_parser_range_lengths []byte = []byte{
	0, 0, 0, 0, 0, 0, 1, 1, 
	1, 1, 0, 0, 0, 0, 0, 0, 
	0, 0, 0, 0, 1, 1, 0, 0, 
}

var _fsm_response_parser_index_offsets []byte = []byte{
	0, 0, 2, 4, 6, 8, 10, 12, 
	16, 19, 22, 26, 29, 33, 37, 40, 
	42, 43, 45, 47, 51, 53, 56, 60, 
}

var _fsm_response_parser_trans_targs []byte = []byte{
	2, 0, 3, 0, 4, 0, 5, 0, 
	6, 0, 7, 0, 8, 20, 7, 0, 
	8, 9, 0, 10, 9, 0, 0, 0, 
	19, 11, 22, 18, 11, 0, 0, 13, 
	12, 22, 15, 13, 14, 22, 15, 14, 
	22, 0, 23, 16, 0, 22, 0, 22, 
	18, 19, 11, 21, 0, 8, 21, 0, 
	16, 17, 13, 12, 23, 
}

var _fsm_response_parser_trans_actions []byte = []byte{
	39, 0, 0, 0, 0, 0, 0, 0, 
	0, 0, 1, 0, 0, 0, 1, 0, 
	0, 21, 0, 0, 3, 0, 0, 0, 
	24, 24, 17, 0, 5, 0, 0, 0, 
	9, 36, 11, 33, 33, 15, 0, 13, 
	15, 0, 27, 0, 0, 17, 0, 17, 
	0, 24, 24, 1, 0, 0, 1, 0, 
	0, 0, 7, 30, 19, 
}

const fsm_response_parser_start int = 1
const fsm_response_parser_first_final int = 22
const fsm_response_parser_error int = 0

const fsm_response_parser_en_main int = 1


//line response_parser.rl:91


// parse implements the Response parse method.
func (r *Response) parse() (int, error) {
	dataLen := r.buf.Len()
	data := r.buf.Bytes()

	cs, p, pe, eof := 0, 0, dataLen, dataLen
	mark, start := 0, 0
	key_start,key_end,value_start,value_end := 0, 0, 0, 0

	r.Headers = make([]*HeaderKV, 0)
		
	
//line response_parser.go:98
	{
	cs = fsm_response_parser_start
	}

//line response_parser.go:103
	{
	var _klen int
	var _trans int
	var _acts int
	var _nacts uint
	var _keys int
	if p == pe {
		goto _test_eof
	}
	if cs == 0 {
		goto _out
	}
_resume:
	_keys = int(_fsm_response_parser_key_offsets[cs])
	_trans = int(_fsm_response_parser_index_offsets[cs])

	_klen = int(_fsm_response_parser_single_lengths[cs])
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
			case data[p] < _fsm_response_parser_trans_keys[_mid]:
				_upper = _mid - 1
			case data[p] > _fsm_response_parser_trans_keys[_mid]:
				_lower = _mid + 1
			default:
				_trans += int(_mid - int(_keys))
				goto _match
			}
		}
		_keys += _klen
		_trans += _klen
	}

	_klen = int(_fsm_response_parser_range_lengths[cs])
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
			case data[p] < _fsm_response_parser_trans_keys[_mid]:
				_upper = _mid - 2
			case data[p] > _fsm_response_parser_trans_keys[_mid + 1]:
				_lower = _mid + 2
			default:
				_trans += int((_mid - int(_keys)) >> 1)
				goto _match
			}
		}
		_trans += _klen
	}

_match:
	cs = int(_fsm_response_parser_trans_targs[_trans])

	if _fsm_response_parser_trans_actions[_trans] == 0 {
		goto _again
	}

	_acts = int(_fsm_response_parser_trans_actions[_trans])
	_nacts = uint(_fsm_response_parser_actions[_acts]); _acts++
	for ; _nacts > 0; _nacts-- {
		_acts++
		switch _fsm_response_parser_actions[_acts-1] {
		case 0:
//line response_parser.rl:11
 mark = p 
		case 1:
//line response_parser.rl:12

		r.Version = string(data[mark:p+1])
	
		case 2:
//line response_parser.rl:15

		r.StatusCode = data[mark:p+1]
	
		case 3:
//line response_parser.rl:18

		r.Reason = data[mark:p+1]
	
		case 4:
//line response_parser.rl:21

		key_start = p
	
		case 5:
//line response_parser.rl:24

		key_end = p+1
	
		case 6:
//line response_parser.rl:27

		value_start = p
	
		case 7:
//line response_parser.rl:30

		value_end = p+1
	
		case 8:
//line response_parser.rl:33

		kv := new(HeaderKV) 

		if key_start > key_end {
			kv.Key = []byte{}
		}else{
			kv.Key = data[key_start:key_end]
		}
		if value_start > value_end {
			kv.Value = []byte{}
		}else{
			kv.Value = data[value_start:value_end]
		}
		
		r.Headers = append(r.Headers, kv)
	
		case 9:
//line response_parser.rl:49

		start = p;
	
		case 10:
//line response_parser.rl:52

		r.StatusLine = data[start:p+1]
	
		case 11:
//line response_parser.rl:55

		r.Body = data[mark:p+1]
	
//line response_parser.go:252
		}
	}

_again:
	if cs == 0 {
		goto _out
	}
	p++
	if p != pe {
		goto _resume
	}
	_test_eof: {}
	_out: {}
	}

//line response_parser.rl:107


	if p != pe {
		return p, fmt.Errorf("unexpected eof(p=%d, pe=%d)", p, pe)
	}

	_ = eof
	return p, nil
}
