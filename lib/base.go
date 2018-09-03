package lib

import (
	"time"
)

//荷载发生器状态
const (
	STATUS_ORIGINAL uint32 = 0 //初始化
	STATUS_STARTING uint32 = 1 //正在启动
	STATUS_STARTED  uint32 = 2 //已启动
	STATUS_STOPPING uint32 = 3 //正在停止
	STATUS_STOPPED  uint32 = 4 //已停止
)

const (
	RET_CODE_SUCCESS              int = 0
	RET_CODE_WARNING_CALL_TIMEOUT     = 1001
	RET_CODE_ERROR_CALL               = 1002 //调用错误
	RET_CODE_ERROR_RESPONSE           = 1003
	RET_CODE_ERROR_CALLER             = 1004 //调用方内部错误
	RET_CODE_FATAL_CALL               = 1005
)

//调用结果
type CallResult struct {
	ID     int64
	Req    RawReq
	Resp   RawResp
	Code   int
	Msg    string
	Elapse time.Duration
}

//原生请求
type RawReq struct {
	ID  int64
	Req []byte
}

//原生响应
type RawResp struct {
	ID     int64
	Resp   []byte
	Err    error
	Elapse time.Duration
}

type Generator interface {
	Start() bool
	Stop() bool
	Status() uint32
	CallCount() int64
}
