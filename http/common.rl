%%{
    machine fsm_common;

    ## RFC2616
    NULL = 0;
    DEL = 0x7F;
    HT = '\t';
    LF = '\n';
    CR = '\r';
    SP = ' ';
    OCTET = any;
    CHAR = ascii;
    UPALPHA = upper;
    LOALPHA = lower;
    ALPHA = alpha;
    DIGIT = digit;
    HEX = xdigit;
    CTL = cntrl | DEL ;
    CRLF = CR? LF;

    ## invisiable_char
    invisible_char = (0x00..0x1f | DEL);

    EMPTY_LINE = (
        ( SP | HT )*
        CRLF
    );

    HTTP_VERSION = ( 
        'HTTP/'
        digit+
        (
            '.'
            digit+
        )?
    );

    HEADER_KEY = (any - ':' - CR - LF)* ;
    HEADER_VALUE = (any - CR - LF)* ;
}%%
