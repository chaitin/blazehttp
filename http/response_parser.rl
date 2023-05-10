package http

import (
	"fmt"
)

%%{
	machine fsm_response_parser;
	include fsm_common "common.rl";

	action mark { mark = p }
	action found_http_version {
		r.Version = string(data[mark:p+1])
	}
	action found_http_status_code {
		r.StatusCode = data[mark:p+1]
	}
	action found_http_reason_phrase {
		r.Reason = data[mark:p+1]
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
		r.StatusLine = data[start:p+1]
	}
	action found_http_body{
		r.Body = data[mark:p+1]
	}

	
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

    reason_phrase = (any - CR - LF)+ ;

    status_line = (
        HTTP_VERSION >mark @found_http_version
		SP+
        digit+ >mark @found_http_status_code
		SP+
        reason_phrase >mark @found_http_reason_phrase
        CRLF
    );

    main := 
        status_line >first_line_start @first_line_end
        (HEADER_LINE @found_http_header)*
        (CRLF HTTP_BODY)?
    ;

	write data;
}%%

// parse implements the Response parse method.
func (r *Response) parse() (int, error) {
	dataLen := r.buf.Len()
	data := r.buf.Bytes()

	cs, p, pe, eof := 0, 0, dataLen, dataLen
	mark, start := 0, 0
	key_start,key_end,value_start,value_end := 0, 0, 0, 0

	r.Headers = make([]*HeaderKV, 0)
		
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
