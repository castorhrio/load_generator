package lib

import (
	"errors"
	"fmt"
)

//goroutine票池接口	(ps:票池相当于POSIX标准中描述的多值信号量)
type Tickets interface {
	Take()
	Return()
	Active() bool
	Total() uint32
	Residue() uint32
}

//新建goroutine票池接口
func NewTickets(total uint32) (Tickets, error) {
	tick := tickets{}
	if !tick.init(total) {
		errMsg := fmt.Sprintf("The goroutine ticket pool can not be initialized! (total=%d)\n", total)
		fmt.Println(errMsg)
		return nil, errors.New(errMsg)
	}
	return &tick, nil
}

type tickets struct {
	total      uint32
	ticketChan chan struct{} //票池容器
	active     bool          //票池是否被激活
}

func (tick *tickets) init(total uint32) bool {
	if tick.active {
		return false
	}

	//通道缓冲的元素个数和通道容量相等，为了防止从票池里获取票时不会阻塞
	if total == 0 {
		return false
	}

	ch := make(chan struct{}, total)
	n := int(total)
	for i := 0; i < n; i++ {
		ch <- struct{}{}
	}
	tick.ticketChan = ch
	tick.total = total
	tick.active = true
	return true
}

func (tick *tickets) Take() {
	<-tick.ticketChan
}

func (tick *tickets) Return() {
	tick.ticketChan <- struct{}{}
}

func (tick *tickets) Active() bool {
	return tick.active
}

func (tick *tickets) Total() uint32 {
	return tick.total
}

func (tick *tickets) Residue() uint32 {
	return uint32(len(tick.ticketChan))
}
