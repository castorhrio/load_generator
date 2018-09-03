package lib

import "time"

//调用器
type Caller interface {
	BuildReq() RawReq
	Call(req []byte, timeout time.Duration) ([]byte, error)
	CheckResp(req RawReq, resp RawResp) *CallResult
}
