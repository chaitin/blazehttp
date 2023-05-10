package http

import (
	"fmt"
	"strings"
)

%%{
	machine fsm_request_parser;
	include fsm_common "common.rl";

	action mark { mark = p }
	action found_http_method {
		r.Method = string(data[mark:p+1])
	}
	action found_meta {
		if p > mark {
			s := string(data[mark:p])
			s = strings.TrimSpace(s)
			if len(s) > 0 {
				r.Metadata = append(r.Metadata, s)
			}
		}
	}
	action found_key_start {
		key_start = p
	}
	action found_key_end {
		key_end = p+1
	}
	action found_value_start {
		value_start = p
	}
	action found_value_end {
		value_end = p+1
	}
	action found_http_header {
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
	}
	action first_line_start {
		start = p;
	}
	action first_line_end {
		r.RequestLine = data[start:p+1]
	}
	action only_requestline {
		r.RequestLine = append(data[start:p], []byte("\r\n")...)
	}
	action found_http_body{
		r.Body = data[mark:p+1]
	}
	
	# META = (any - CRLF)+ >mark;

	COMMENT_LINE = (
        '#'
		(
			SP*
			|
        	(any - CRLF)++ >mark 
		)
        CRLF >found_meta
    );

    HEADER_LINE = (
        HEADER_KEY >found_key_start @found_key_end
        SP*
        ':'
        SP*
        HEADER_VALUE >found_value_start @found_value_end
        CRLF
    );

    HTTP_BODY = (
        any+
    ) >mark @found_http_body;

    http_method = alpha+ ;
    http_uri = (any - CR - LF)+;

    request_line = (
        http_method >mark @found_http_method
        SP+
        http_uri
        SP+ 
        (any - CRLF)* %eof only_requestline
        CRLF 
    );

	main := 
		(EMPTY_LINE | COMMENT_LINE)*
		request_line >first_line_start @first_line_end 
		(HEADER_LINE @found_http_header)*
		(CRLF HTTP_BODY)?
	;

	write data;
}%%

// parse implements the Request parse method.
func (r *Request) parse() (int, error) {
	dataLen := r.inBuf.Len()
	data := r.inBuf.Bytes()

	cs, p, pe, eof := 0, 0, dataLen, dataLen
	mark, start := 0, 0
	key_start,key_end,value_start,value_end := 0, 0, 0, 0

	r.Headers = make([]*HeaderKV, 0)
	r.Metadata = make([]string, 0)
		
	%%{
		write init;
		write exec;
	}%%

	if p != pe {
		return p, fmt.Errorf("unexpected eof(p=%d, pe=%d)", p, pe)
	}

	_ = eof
	return p, nil
}
