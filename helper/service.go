package helper

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"bytes"
	"sync/atomic"
	"errors"
)

type ServerReq struct {
	ID       int64
	Data     []int
	Operator string
}

type ServerResp struct {
	ID      int64
	Formula string
	Result  int
	Err     error
}

type TCPServer struct{
	listener net.Listener
	active uint32
}

func NewTCPServer() *TCPServer{
	return &TCPServer{}
}

func(server *TCPServer)init(addr string)error{
	if !atomic.CompareAndSwapUint32(&server.active,0,1){
		return nil
	}
	lis,err:=net.Listen("tcp",addr)
	if err!=nil{
		atomic.StoreUint32(&server.active,0)
		return err
	}
	server.listener = lis
	return nil
}

func(server *TCPServer) Listen(addr string) error{
	err:=server.init(addr)
	if err!=nil{
		return err
	}

	go func(){
		for{
			if atomic.LoadUint32(&server.active) != 1{
				break
			}

			conn,err:=server.listener.Accept()
			if err!=nil{
				if atomic.LoadUint32(&server.active) == 1{
					fmt.Printf("Server:Request Acception Error:%s\n",err)
				}else{
					fmt.Println("Server:broken acception because of closed network connection")
				}
				continue
			}
			go reqHandler(conn)
		}
		}()
		return nil
	}


	func reqHandler(conn net.Conn){
	var errMsg string
	var sresp ServerResp
	req,err:=read(conn,'\n')
	if err!=nil{
		errMsg = fmt.Sprintf("Server:Request Read Error:%s",err)
	}else{
		var sreq ServerReq
		err := json.Unmarshal(req,&sreq)
		if err!=nil{
			errMsg = fmt.Sprintf("Server: Request Unmarshal Error: %s", err)
		}else{
			sresp.ID = sreq.ID
			sresp.Result = op(sreq.Data,sreq.Operator)
			sresp.Formula = genFormula(sreq.Data,sreq.Operator,sresp.Result,true)
		}
	}

	if errMsg != ""{
		sresp.Err = errors.New(errMsg)
	}

	bytes,err:=json.Marshal(sresp)
	if err!=nil{
		fmt.Printf("Server: Response Marshal Error: %s\n", err)
	}

	_,err=write(conn,bytes,'\n')
	if err!=nil{
		fmt.Printf("Server: Response write Error: %s\n", err)
	}
}

func(server *TCPServer) Close() bool{
	if atomic.CompareAndSwapUint32(&server.active,1,0){
		return false
	}

	server.listener.Close()
	return true
}

func op(data []int, oper string) int {
	var result = 0
	switch {
	case oper == "+":
		for _, item := range data {
			result += item
		}

	case oper == "-":
		for _, item := range data {
			result -= item
		}
	case oper == "*":
		for _, item := range data {
			result *= item
		}
	case oper == "/":
		for _, item := range data {
			result /= item
		}
	}
	return result
}


//生成运算表达式
func genFormula(data []int,oper string,result int,equal bool) string{
	var buff bytes.Buffer
	n:=len(data)
	for i:=0;i<n;i++{
		//保证第一位是操作数
		if i>0{
			buff.WriteString(" ")
			buff.WriteString(oper)
			buff.WriteString(" ")
		}
		buff.WriteString(strconv.Itoa(data[i]))
	}
	if equal{
		buff.WriteString(" = ")
	}else{
			buff.WriteString(" != ")
	}
	buff.WriteString(strconv.Itoa(result))
	return buff.String()

}