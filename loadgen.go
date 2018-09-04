package load_generator

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"./lib"
)

type generator struct {
	caller      lib.Caller
	timeout     time.Duration
	lps         uint32 //每秒载荷量
	duration    time.Duration
	concurrency uint32
	tickets     lib.Tickets
	ctx         context.Context
	cancelFunc  context.CancelFunc
	callCount   int64
	status      uint32
	resultChan  chan *lib.CallResult
}

func (ge *generator) init() error {
	var buf bytes.Buffer
	buf.WriteString("Initializing the load generator...")

	//并发量=单个载荷的响应超时时间/载荷的发送间隔时间
	//1e9代表1s对应的纳秒数
	//+1表示最开始发送的那个载荷
	var total = int64(ge.timeout)/int64(1e9/ge.lps) + 1

	if total > math.MaxInt32 {
		total = math.MaxInt32
	}

	ge.concurrency = uint32(total)
	tick, err := lib.NewTickets(ge.concurrency)
	if err != nil {
		return err
	}

	ge.tickets = tick
	buf.WriteString(fmt.Sprintf("Done. (concurrency=%d)", ge.concurrency))
	return nil
}

func NewGenerator(para Parameter) (lib.Generator, error) {
	fmt.Println("New a load generator...")
	if err := para.Check(); err != nil {
		return nil, err
	}
	gen := &generator{
		caller:     para.Caller,
		timeout:    para.Timeout,
		lps:        para.LPS,
		duration:   para.Duration,
		resultChan: para.ResultChan,
	}

	if err := gen.init(); err != nil {
		return nil, err
	}

	return gen, nil
}

func (gen *generator) Start() bool {
	if !atomic.CompareAndSwapUint32(&gen.status, lib.STATUS_ORIGINAL, lib.STATUS_STARTING) {
		if !atomic.CompareAndSwapUint32(&gen.status, lib.STATUS_STOPPED, lib.STATUS_STARTING) {
			return false
		}
	}

	//设置节流器
	var throttle <-chan time.Time
	if gen.lps > 0 {
		interval := time.Duration(1e9 / gen.lps)
		throttle = time.Tick(interval)
	}

	//初始化上下文和取消函数，让发生器能够在运行一段时间之后自己停下来
	gen.ctx, gen.cancelFunc = context.WithTimeout(context.Background(), gen.duration)

	//初始化调用计数
	gen.callCount = 0

	atomic.StoreUint32(&gen.status, lib.STATUS_STARTED)

	go func() {
		fmt.Println("Generating load...")
		gen.genLoad(throttle)
		fmt.Printf("Stopped. (call count: %d)", gen.callCount)
	}()

	return true
}

func (gen *generator) genLoad(throttle <-chan time.Time) {
	fmt.Println("gen load")
	for {
		select {
		case <-gen.ctx.Done():
			gen.prepareToStop(gen.ctx.Err())
			return
		default:
		}
		gen.asynCall()
		if gen.lps > 0 {
			select {
			case <-throttle:
			case <-gen.ctx.Done():
				gen.prepareToStop(gen.ctx.Err())
				return
			}
		}
	}
}

func (gen *generator) prepareToStop(ctxError error) {
	fmt.Printf("Prepare to stop load generator (cause:%s)...\n", ctxError)
	atomic.CompareAndSwapUint32(&gen.status, lib.STATUS_STARTED, lib.STATUS_STOPPING)
	fmt.Println("Closing result channel...")
	close(gen.resultChan)
	atomic.StoreUint32(&gen.status, lib.STATUS_STOPPED)
}

func (gen *generator) asynCall() {
	fmt.Println("asyn call")
	gen.tickets.Take()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				err, ok := interface{}(p).(error)
				var errMsg string
				if ok {
					errMsg = fmt.Sprintf("Async Call Panic! (error:%s)\n", err)
				} else {
					errMsg = fmt.Sprintf("Async Call Panic! (error:%#v)\n", p)
				}
				result := &lib.CallResult{
					ID:   -1,
					Code: lib.RET_CODE_FATAL_CALL,
					Msg:  errMsg,
				}
				gen.sendResult(result)
			}

			gen.tickets.Return()
		}()

		rawReq := gen.caller.BuildReq()
		var callStatus uint32
		timer := time.AfterFunc(gen.timeout, func() {
			if !atomic.CompareAndSwapUint32(&callStatus, 0, 2) {
				return
			}

			result := &lib.CallResult{
				ID:     rawReq.ID,
				Req:    rawReq,
				Code:   lib.RET_CODE_WARNING_CALL_TIMEOUT,
				Msg:    fmt.Sprintf("Timeout! (excepted: < %v)", gen.timeout),
				Elapse: gen.timeout,
			}
			gen.sendResult(result)
		})

		rawResp := gen.callOne(&rawReq)
		if !atomic.CompareAndSwapUint32(&callStatus, 0, 1) {
			return
		}
		timer.Stop()

		var result *lib.CallResult
		if rawResp.Err != nil {
			result = &lib.CallResult{
				ID:     rawResp.ID,
				Req:    rawReq,
				Code:   lib.RET_CODE_ERROR_CALL,
				Msg:    rawResp.Err.Error(),
				Elapse: rawResp.Elapse,
			}
		} else {
			result = gen.caller.CheckResp(rawReq, *rawResp)
			result.Elapse = rawResp.Elapse
		}
		gen.sendResult(result)
	}()
}

func (gen *generator) sendResult(result *lib.CallResult) bool {
	if atomic.LoadUint32(&gen.status) != lib.STATUS_STARTED {
		gen.printIgnoreResult(result, "stopped load generator")
		return false
	}
	select {
	case gen.resultChan <- result:
		return true
	default:
		gen.printIgnoreResult(result, "full result channel")
		return false
	}
}

func (gen *generator) printIgnoreResult(result *lib.CallResult, str string) {
	resultMsg := fmt.Sprintf(
		"ID=%d,Code=%d,Msg=%s,Elapse=%v", result.ID, result.Code, result.Msg, result.Elapse)
	fmt.Printf("Ignored result:%s. (cause:%s)\n", resultMsg, str)
}

func (gen *generator) callOne(rawReq *lib.RawReq) *lib.RawResp {
	atomic.AddInt64(&gen.callCount, 1)
	if rawReq == nil {
		return &lib.RawResp{ID: -1, Err: errors.New("Invalid raw request.")}
	}
	start := time.Now().UnixNano()
	resp, err := gen.caller.Call(rawReq.Req, gen.timeout)
	end := time.Now().UnixNano()
	elapsedTime := time.Duration(end - start)
	var rawResp lib.RawResp
	if err != nil {
		errMsg := fmt.Sprintf("Sync call error:%s", err)
		rawResp = lib.RawResp{
			ID:     rawReq.ID,
			Err:    errors.New(errMsg),
			Elapse: elapsedTime}
	} else {
		rawResp = lib.RawResp{
			ID:     rawReq.ID,
			Resp:   resp,
			Elapse: elapsedTime}
	}
	return &rawResp
}

func (gen *generator) Stop() bool {
	if !atomic.CompareAndSwapUint32(&gen.status, lib.STATUS_STARTED, lib.STATUS_STOPPING) {
		return false
	}
	gen.cancelFunc()
	for {
		if atomic.LoadUint32(&gen.status) == lib.STATUS_STOPPED {
			break
		}
		time.Sleep(time.Microsecond)
	}
	return true
}

func (gen *generator) Status() uint32 {
	return atomic.LoadUint32(&gen.status)
}

func (gen *generator) CallCount() int64 {
	return atomic.LoadInt64(&gen.callCount)
}
