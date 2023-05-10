
//line request_parser.rl:1
package http

import (
	"fmt"
	"strings"
)


//line request_parser.go:12
var _fsm_request_parser_actions []byte = []byte{
	0, 1, 0, 1, 1, 1, 2, 1, 3, 
	1, 4, 1, 5, 1, 6, 1, 7, 
	1, 9, 1, 10, 1, 11, 2, 0, 
	2, 2, 0, 11, 2, 3, 4, 2, 
	5, 6, 2, 5, 7, 3, 8, 0, 
	1, 
}

var _fsm_request_parser_key_offsets []byte = []byte{
	0, 0, 9, 13, 14, 16, 18, 23, 
	25, 28, 29, 32, 35, 37, 38, 38, 
	39, 42, 
}

var _fsm_request_parser_trans_keys []byte = []byte{
	9, 10, 13, 32, 35, 65, 90, 97, 
	122, 9, 10, 13, 32, 10, 10, 13, 
	10, 13, 32, 65, 90, 97, 122, 10, 
	13, 10, 13, 32, 10, 10, 13, 58, 
	10, 13, 32, 10, 13, 10, 10, 10, 
	13, 58, 
}

var _fsm_request_parser_single_lengths []byte = []byte{
	0, 5, 4, 1, 2, 2, 1, 2, 
	3, 1, 3, 3, 2, 1, 0, 1, 
	3, 0, 
}

var _fsm_request_parser_range_lengths []byte = []byte{
	0, 2, 0, 0, 0, 0, 2, 0, 
	0, 0, 0, 0, 0, 0, 0, 0, 
	0, 0, 
}

var _fsm_request_parser_index_offsets []byte = []byte{
	0, 0, 8, 13, 15, 18, 21, 25, 
	28, 32, 34, 38, 42, 45, 47, 48, 
	50, 54, 
}

var _fsm_request_parser_trans_targs []byte = []byte{
	2, 1, 3, 2, 4, 6, 6, 0, 
	2, 1, 3, 2, 0, 1, 0, 1, 
	5, 5, 1, 5, 5, 7, 6, 6, 
	0, 0, 0, 8, 0, 0, 9, 8, 
	16, 9, 0, 0, 11, 10, 16, 13, 
	11, 12, 16, 13, 12, 16, 0, 17, 
	14, 0, 14, 15, 11, 10, 17, 
}

var _fsm_request_parser_trans_actions []byte = []byte{
	0, 0, 0, 0, 0, 38, 38, 0, 
	0, 0, 0, 0, 0, 0, 0, 5, 
	23, 1, 5, 5, 0, 0, 3, 3, 
	0, 0, 0, 0, 0, 0, 0, 0, 
	17, 0, 0, 0, 0, 9, 35, 11, 
	32, 32, 15, 0, 13, 15, 0, 26, 
	0, 0, 0, 0, 7, 29, 21, 
}

var _fsm_request_parser_eof_actions []byte = []byte{
	0, 0, 0, 0, 0, 0, 0, 0, 
	0, 19, 0, 0, 0, 0, 0, 0, 
	0, 0, 
}

const fsm_request_parser_start int = 1
const fsm_request_parser_first_final int = 16
const fsm_request_parser_error int = 0

const fsm_request_parser_en_main int = 1


//line request_parser.rl:111


// parse implements the Request parse method.
func (r *Request) parse() (int, error) {
	dataLen := r.inBuf.Len()
	data := r.inBuf.Bytes()

	cs, p, pe, eof := 0, 0, dataLen, dataLen
	mark, start := 0, 0
	key_start,key_end,value_start,value_end := 0, 0, 0, 0

	r.Headers = make([]*HeaderKV, 0)
	r.Metadata = make([]string, 0)
		
	
//line request_parser.go:104
	{
	cs = fsm_request_parser_start
	}

//line request_parser.go:109
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
	_keys = int(_fsm_request_parser_key_offsets[cs])
	_trans = int(_fsm_request_parser_index_offsets[cs])

	_klen = int(_fsm_request_parser_single_lengths[cs])
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
			case data[p] < _fsm_request_parser_trans_keys[_mid]:
				_upper = _mid - 1
			case data[p] > _fsm_request_parser_trans_keys[_mid]:
				_lower = _mid + 1
			default:
				_trans += int(_mid - int(_keys))
				goto _match
			}
		}
		_keys += _klen
		_trans += _klen
	}

	_klen = int(_fsm_request_parser_range_lengths[cs])
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
			case data[p] < _fsm_request_parser_trans_keys[_mid]:
				_upper = _mid - 2
			case data[p] > _fsm_request_parser_trans_keys[_mid + 1]:
				_lower = _mid + 2
			default:
				_trans += int((_mid - int(_keys)) >> 1)
				goto _match
			}
		}
		_trans += _klen
	}

_match:
	cs = int(_fsm_request_parser_trans_targs[_trans])

	if _fsm_request_parser_trans_actions[_trans] == 0 {
		goto _again
	}

	_acts = int(_fsm_request_parser_trans_actions[_trans])
	_nacts = uint(_fsm_request_parser_actions[_acts]); _acts++
	for ; _nacts > 0; _nacts-- {
		_acts++
		switch _fsm_request_parser_actions[_acts-1] {
		case 0:
//line request_parser.rl:12
 mark = p 
		case 1:
//line request_parser.rl:13

		r.Method = string(data[mark:p+1])
	
		case 2:
//line request_parser.rl:16

		if p > mark {
			s := string(data[mark:p])
			s = strings.TrimSpace(s)
			if len(s) > 0 {
				r.Metadata = append(r.Metadata, s)
			}
		}
	
		case 3:
//line request_parser.rl:25

		key_start = p
	
		case 4:
//line request_parser.rl:28

		key_end = p+1
	
		case 5:
//line request_parser.rl:31

		value_start = p
	
		case 6:
//line request_parser.rl:34

		value_end = p+1
	
		case 7:
//line request_parser.rl:37

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
	
		case 8:
//line request_parser.rl:53

		start = p;
	
		case 9:
//line request_parser.rl:56

		r.RequestLine = data[start:p+1]
	
		case 11:
//line request_parser.rl:62

		r.Body = data[mark:p+1]
	
//line request_parser.go:259
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
	if p == eof {
		__acts := _fsm_request_parser_eof_actions[cs]
		__nacts := uint(_fsm_request_parser_actions[__acts]); __acts++
		for ; __nacts > 0; __nacts-- {
			__acts++
			switch _fsm_request_parser_actions[__acts-1] {
			case 10:
//line request_parser.rl:59

		r.RequestLine = append(data[start:p], []byte("\r\n")...)
	
//line request_parser.go:283
			}
		}
	}

	_out: {}
	}

//line request_parser.rl:128


	if p != pe {
		return p, fmt.Errorf("unexpected eof(p=%d, pe=%d)", p, pe)
	}

	_ = eof
	return p, nil
}
