package gen_routine

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"time"
)

const (
	Infinity = time.Second * 864000
)

// SvrBehavior 消息处理函数
type SvrBehavior interface {
	// HandleMsg 处理消息
	HandleMsg(Msg) (interface{}, *Error)
	// Terminate 处理结束
	Terminate(err *Error)
	// Init 协程初始化
	Init(*Svr) *Error
}

type Logger interface {
	Errorf(fmt string, args ...interface{})
}

// SetLogger 设置日志输出接口
func SetLogger(l Logger) {
	logger = l
}

func RootMgr() *Mgr {
	return mgrG
}

// NewMgr 开一个管理器
func NewMgr(parent *Mgr, name string) (*Mgr, *Error) {
	return newMgr(parent, name)
}

// NewSvr 开个协程
// k 为nil 则自动生成uint64的 rid
func (mgr *Mgr) NewSvr(k interface{}, mod SvrBehavior) (*Svr, *Error) {
	return mgr.newSvr(k, mod)
}

// LookupMgr 查询管理器
func LookupMgr(parent *Mgr, name string) (interface{}, bool) {
	return parent.lookup(name)
}

// LookupSvr 查询协程
func (mgr *Mgr) LookupSvr(k interface{}) (interface{}, bool) {
	return mgr.lookup(k)
}

// GetMod 获取逻辑模块
func (svr *Svr) GetMod() SvrBehavior {
	return svr.mod
}

// Foreach 遍历元素
func (mgr *Mgr) Foreach(f func(k interface{}, v interface{}) bool) {
	mgr.foreach(f)
}

// StopMgr 关某个管理器
func (mgr *Mgr) StopMgr(reason *Error) *Error {
	return mgr.stop(reason)
}

// StopSvr 停掉协程
func (svr *Svr) StopSvr(reason *Error) {
	svr.receive <- &MsgStop{reason: reason}
}

// Cast 不关心返回值的调用
func (svr *Svr) Cast(msg Msg) {
	svr.receive <- msg
}

// CallInfinity 不带超时的调用，其实是很长的一个时间
// Call 接口一定要注意，不要自己协程调到自己头上了！！！！！！
func (svr *Svr) CallInfinity(msg Msg) (interface{}, *Error) {
	return svr.call(msg, Infinity)
}

// Call 带超时的阻塞调用
func (svr *Svr) Call(msg Msg, timeout time.Duration) (interface{}, *Error) {
	return svr.call(msg, timeout)
}

// SyncExec 直接在协程中调用某个函数
func (svr *Svr) SyncExec(f interface{}, timeout time.Duration, args ...interface{}) ([]interface{}, *Error) {
	in, err := execIn(f, args...)
	if err != nil {
		return nil, err
	}
	msg := &MsgExec{
		f:      f,
		args:   in,
		isSync: true,
	}
	ret, err := svr.call(msg, timeout)
	retA := ret.([]reflect.Value)
	retV := make([]interface{}, len(retA))
	for i, v := range retA {
		retV[i] = v.Interface()
	}
	return retV, nil
}

// ASyncExec 在协程中调用某个函数，但不关心结果
func (svr *Svr) ASyncExec(f interface{}, args ...interface{}) *Error {
	in, err := execIn(f, args...)
	if err != nil {
		return err
	}
	//fv.Call(in)
	msg := &MsgExec{
		f:      f,
		args:   in,
		isSync: false,
	}
	svr.Cast(msg)
	return nil
}

// execIn 计算调用参数
func execIn(f interface{}, args ...interface{}) ([]reflect.Value, *Error) {
	ft := reflect.TypeOf(f)
	//fv := reflect.ValueOf(f)
	lenE := ft.NumIn()
	lenA := len(args)
	if lenE != lenA {
		return nil, &Error{Code: ErrorReflectParamsLen, Param: fmt.Sprintf("expected : %d, actual : %d", lenE, lenA)}
	}
	in := make([]reflect.Value, lenA)
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}
	return in, nil
}

// JustRun 不需要管理器，直接开一个协程跑个函数
func JustRun(work func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errorf("run error %v %s\n", r, debug.Stack())
			}
		}()
		work()
	}()
}

// =========== 分组相关接口 ===========

// GrpReg 在分组中注册
func (svr *Svr) GrpReg(grpKey interface{}) {
	reg(grpKey, svr)
}

// GrpUnReq 从指定分组中反注册
func (svr *Svr) GrpUnReq(grpKey interface{}) {
	unReq(grpKey, svr)
}

// GrpAll 获取某个分组中所有协程
func GrpAll(grpKey interface{}) []*Svr {
	return allSvr(grpKey)
}

// GrpByeBye 协程退出，通知清理
func (svr *Svr) GrpByeBye() {
	bye(svr)
}
