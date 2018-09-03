package load_generator

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"./lib"
)

type Parameter struct {
	Caller     lib.Caller
	Timeout    time.Duration
	LPS        uint32
	Duration   time.Duration
	ResultChan chan *lib.CallResult
}

func (para *Parameter) Check() error {
	var errMsg []string
	if para.Caller == nil {
		errMsg = append(errMsg, "Invalid caller!")
	}

	if para.Timeout == 0 {
		errMsg = append(errMsg, "Invalid timeout")
	}

	if para.LPS == 0 {
		errMsg = append(errMsg, "Invalid lps")
	}

	if para.Duration == 0 {
		errMsg = append(errMsg, "Invalid duration")
	}

	if para.ResultChan == nil {
		errMsg = append(errMsg, "Invalid result chan")
	}

	var buf bytes.Buffer
	buf.WriteString("Checking the parameter....")
	if errMsg != nil {
		errMsg := strings.Join(errMsg, " ")
		buf.WriteString(fmt.Sprintf("Check the parameter fail !(%s)", errMsg))
		return errors.New(errMsg)
	}

	buf.WriteString(fmt.Sprintf("Check the parameter pass. (timeout=%s,lps=%d,duration=%s)", para.Timeout, para.LPS, para.Duration))
	return nil
}
