package gen_routine

import (
	"reflect"
	"runtime/debug"
	"time"
)

const receiveChanLen = 64

type Svr struct {
	key     interface{}
	receive chan Msg
	mgr     *Mgr
	mod     SvrBehavior
}

type MsgRet struct {
	ret interface{}
	err *Error
}

type MsgStop struct {
	reason *Error
}

type MsgCall struct {
	msg     Msg
	retChan chan *MsgRet
}

type MsgExec struct {
	f      interface{}
	args   []reflect.Value
	isSync bool // 是否为阻塞调用
}

type Msg interface{}

func (svr *Svr) start(mgr *Mgr) *Error {
	// 接收chan 加缓存是因为非阻塞式的自己给自己发消息能够写起来比较简单
	svr.receive = make(chan Msg, receiveChanLen)
	svr.mgr = mgr
	startOkChan := make(chan *Error)
	svr.mgr.wait.Add(1)
	go svr.loop(startOkChan)
	startRet := <-startOkChan
	return startRet
}

func (svr *Svr) loop(startOkChan chan *Error) {
	reason := &Error{Code: ErrorCodeOk}
	defer func() {
		// 截止 2022-03-16 loop 里面调用的下一层函数都自己recovery了的，也必须他们自己就recovery
		// 要保证协程的正常流程
		//if r := recover(); r != nil {
		//	reason = &Error{Code: ErrorCrash, ParamPanic: r, Param: string(debug.Stack())}
		//	errorf("svr loop crash %v \n%s", reason.ParamPanic, reason.Param)
		//}
		svr.mgr.wait.Done()
		svr.stop(reason)
	}()
	if e := svr.behaviorInit(startOkChan); e != nil {
		reason = e
		return
	}
LOOP:
	for {
		select {
		case <-svr.mgr.ctx.Done():
			reason = &Error{Code: ErrorCtxDone}
			break LOOP
		case msg := <-svr.receive: // 处理发进来的消息
			_, err := svr.handle(msg)
			// 返回错误，则退出
			if err != nil && err.Code != ErrorCodeOk {
				reason = err
				break LOOP
			}
		}
	}
}

func (svr *Svr) behaviorInit(startOkChan chan *Error) (reason *Error) {
	defer func() {
		if r := recover(); r != nil {
			reason = &Error{Code: ErrorCrash, ParamPanic: r, Param: string(debug.Stack())}
			errorf("svr behaviorInit crash %v \n%s", reason.ParamPanic, reason.Param)
		}
		startOkChan <- reason
	}()
	if err := svr.mod.Init(svr); err != nil {
		reason = &Error{}
		reason.Code = ErrorRoutineInitFail
		reason.Last = err
	}
	return
}

func (svr *Svr) handle(msg Msg) (ret interface{}, reason *Error) {
	// call 里面如果崩了，需要这里 catch 住，然后返回给调用者崩溃的结果
	defer func() {
		if r := recover(); r != nil {
			reason = &Error{Code: ErrorCrash, ParamPanic: r, Param: string(debug.Stack())}
			errorf("svr handle msg crash %v \n%s", reason.ParamPanic, reason.Param)
		}
	}()
	switch v := msg.(type) {
	case *MsgCall:
		ret, err := svr.handle(v.msg)
		v.retChan <- &MsgRet{ret: ret, err: err}
		return ret, err
	case *MsgStop:
		return nil, v.reason
	case *MsgExec:
		return handleExec(v)
	default:
		return svr.mod.HandleMsg(v)
	}
}

func (svr *Svr) call(msg Msg, timeout time.Duration) (interface{}, *Error) {
	retChan := make(chan *MsgRet)
	callMsg := &MsgCall{msg: msg, retChan: retChan}
	svr.receive <- callMsg
	select {
	case <-time.After(timeout):
		return nil, &Error{Code: ErrorTimeout}
	case ret := <-retChan:
		return ret.ret, ret.err
	}
}

// handleExec 处理直接走协程调用某个函数的情况
func handleExec(msg *MsgExec) (interface{}, *Error) {
	fv := reflect.ValueOf(msg.f)
	ret := fv.Call(msg.args)
	if !msg.isSync {
		return nil, nil
	}
	return ret, nil
}

func (svr *Svr) stop(reason *Error) (ret *Error) {
	defer func() {
		if r := recover(); r != nil {
			ret = &Error{Code: ErrorCrash, ParamPanic: r, Param: string(debug.Stack())}
			errorf("svr stop crash %v \n%s", ret.ParamPanic, ret.Param)
		}
		svr.mgr.svrTerminate(svr)
	}()
	svr.mod.Terminate(reason)
	return
}
