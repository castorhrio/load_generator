package helper

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	"../lib"
)

//操作符
var operators = []string{"+", "-", "*", "/"}

//数据流分隔符
const flag = '\n'

type TCPCommunicator struct {
	addr string
}

func NewTCPCommunicator(addr string) lib.Caller {
	return &TCPCommunicator{addr: addr}
}

//构建一个请求
func (comm *TCPCommunicator) BuildReq() lib.RawReq {
	id := time.Now().UnixNano()
	req := ServerReq{
		ID: id,
		Data: []int{
			int(rand.Int31n(1000) + 1),
			int(rand.Int31n(1000) + 1)},
		Operator: func() string {
			return operators[rand.Int31n(100)%4]
		}(),
	}
	bytes, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	rawReq := lib.RawReq{ID: id, Req: bytes}
	return rawReq
}

func (comm *TCPCommunicator) Call(req []byte, timeout time.Duration) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", comm.addr, timeout)
	if err != nil {
		return nil, err
	}

	_, err = write(conn, req, flag)
	if err != nil {
		return nil, err
	}

	return read(conn, flag)
}

func (comm *TCPCommunicator) CheckResp(req lib.RawReq, resp lib.RawResp) *lib.CallResult {
	var result lib.CallResult
	result.ID = resp.ID
	result.Req = req
	result.Resp = resp
	var sreq ServerReq
	err := json.Unmarshal(req.Req, &sreq)
	if err != nil {
		result.Code = lib.RET_CODE_FATAL_CALL
		result.Msg = fmt.Sprintf("Incorrectly fromatted Request:%s!\n", string(req.Req))
		fmt.Println(result.Msg)
		return &result
	}

	var sresp ServerResp
	err = json.Unmarshal(resp.Resp, &sresp)
	if err != nil {
		result.Code = lib.RET_CODE_ERROR_RESPONSE
		result.Msg = fmt.Sprintf("Incorrectly fromatted Response:%s!\n", string(resp.Resp))
		fmt.Println(result.Msg)
		return &result
	}

	if sresp.ID != sreq.ID {
		result.Code = lib.RET_CODE_ERROR_RESPONSE
		result.Msg = fmt.Sprintf("Incorrectly raw id (%d!=%d)!\n", req.ID, resp.ID)
		fmt.Println(result.Msg)
		return &result
	}

	if sresp.Err != nil {
		result.Code = lib.RET_CODE_ERROR_CALLER
		result.Msg = fmt.Sprintf("Abnormal server: %s!\n", sresp.Err)
		fmt.Println(result.Msg)
		return &result
	}

	if sresp.Result != op(sreq.Data, sreq.Operator) {
		result.Code = lib.RET_CODE_ERROR_RESPONSE
		result.Msg =
			fmt.Sprintf("Incorrect result: %s!\n", genFormula(sreq.Data, sreq.Operator, sresp.Result, false))
		fmt.Println(result.Msg)
		return &result
	}

	result.Code = lib.RET_CODE_SUCCESS
	result.Msg = fmt.Sprintf("Success.(%s)", sresp.Formula)
	return &result
}

//从连接中读取数据直到遇到flag参数
func read(conn net.Conn, deline byte) ([]byte, error) {
	readBytes := make([]byte, 1)
	var buffer bytes.Buffer
	for {
		_, err := conn.Read(readBytes)
		if err != nil {
			return nil, err
		}

		readByte := readBytes[0]
		if readByte == deline {
			break
		}
		buffer.WriteByte(readByte)
	}
	return buffer.Bytes(), nil
}

//向链接中写数据，并在最后追加flag
func write(conn net.Conn, content []byte, deline byte) (int, error) {
	write := bufio.NewWriter(conn)
	n, err := write.Write(content)
	if err == nil {
		write.WriteByte(deline)
	}

	if err == nil {
		err = write.Flush()
	}
	return n, err
}
