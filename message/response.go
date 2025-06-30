package message

// Resp rpc 响应信息
//
// | 	  head length 4  	| 									body length 4           		    |
// |      message id 	    |  version 1 | serializer 1 | compression algorithm 1 | message type 1  |
// |  	  error message	    |
// | 	  response body	    |
type Resp struct {
	Err  []byte
	Body []byte

	Meta map[string]string
}
