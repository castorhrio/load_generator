package load_generator

import (
	"time"
	"testing"

	"./helper"
	"./lib"
)

func TestStart(t *testing.T){
	server := helper.NewTCPServer()
	defer server.Close()
	serverAddr := "127.0.0.1:8043"
	t.Logf("Startup TCP server(%s)...\n",serverAddr)
	err:=server.Listen(serverAddr)
	if err!=nil{
		t.Fatalf("TCP Server startup failling! (addr=%s)",serverAddr)
		t.FailNow()
	}

	pset:=Parameter{
		Caller:helper.NewTCPCommunicator(serverAddr),
		Timeout:50*time.Millisecond,
		LPS:uint32(1000),
		Duration:10*time.Second,
		ResultChan:make(chan *lib.CallResult,50),
	}
	t.Logf("Initialize load generator (timeout=%v, lps=%d, duration=%v)...",pset.Timeout, pset.LPS, pset.Duration)

	gen,err:=NewGenerator(pset)
	if err!=nil{
		t.Fatalf("Load generator initialization failing: %s\n",err)
		t.FailNow()
	}

	t.Log("Start load generator...")
	gen.Start()

	countMap := make(map[int] int)
	for r:=range pset.ResultChan{
		countMap[r.Code] = countMap[r.Code] +1
		t.Logf("Result: ID=%d, Code=%d, Msg=%s, Elapse=%v.\n",r.ID, r.Code, r.Msg, r.Elapse)
	}

	var total int
	t.Log("RetCode Count:")
	for k,v:=range countMap{
		codePlan := lib.GetRetCodePlain(k)
		t.Logf("Code plain: %s (%d), Count: %d.\n",codePlan,k,v)
		total += v
	}

	t.Logf("Total:%d.\n",total)
	successCount := countMap[lib.RET_CODE_SUCCESS]
	tps:=float64(successCount)/float64(pset.Duration/1e9)
	t.Logf("Load per second:%d; Treatments per second:%f.\n",pset.LPS,tps)
}