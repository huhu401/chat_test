package gen_routine

import (
	"context"
	"sync"
	"sync/atomic"
)

type Mgr struct {
	name      string
	m         sync.Map // 经测试，sync.Map 写比 map+mutex 慢一半不到的样子，读要快很多被
	countSvr  int32
	countMgr  int32
	ctx       context.Context
	ctxCancel context.CancelFunc
	wait      *sync.WaitGroup
	parent    *Mgr

	lock sync.RWMutex
}

type SvrImp interface {
}

// 单键管理器
var mgrG *Mgr
var maxRId uint64

func BeforeMain() {
	initRootMgr()
	initGrp()
}

func initRootMgr() {
	// 初始化全局管理器
	mgrG = &Mgr{name: "global"}
	mgrG.init(context.Background())
	maxRId = 0
}

func (mgr *Mgr) init(parent context.Context) {
	mgr.m = sync.Map{}
	mgr.wait = &sync.WaitGroup{}
	mgr.ctx, mgr.ctxCancel = context.WithCancel(parent)
	mgr.lock = sync.RWMutex{}
}

// 创建子管理器
func newMgr(parent *Mgr, name string) (*Mgr, *Error) {
	mgr := new(Mgr)
	parent.lock.Lock()
	defer func() {
		parent.lock.Unlock()
	}()
	if old, loaded := parent.reg(name, mgr); loaded {
		return old.(*Mgr), &Error{Code: ErrorAlreadyHad}
	}
	mgr.init(parent.ctx)
	mgr.name = name
	mgr.parent = parent
	atomic.AddInt32(&parent.countMgr, 1)
	return mgr, nil
}

// 分配一个新的协程编号
func svrKey(k interface{}) interface{} {
	if k == nil {
		return atomic.AddUint64(&maxRId, 1)
	}
	return k
}

// 全部协程退出
func (mgr *Mgr) stop(reason *Error) *Error {
	mgr.ctxCancel()
	mgr.wait.Wait()
	atomic.AddInt32(&mgr.parent.countMgr, -1)
	mgr.parent.unreg(mgr.name)
	return nil
}

// 启动一个没名字的svr
func (mgr *Mgr) newSvr(k interface{}, mod SvrBehavior) (*Svr, *Error) {
	mgr.lock.Lock()
	defer func() {
		mgr.lock.Unlock()
	}()
	svr := mgr.svrBase(k, mod)
	// 子管理器注册 先检查有没有老的
	if old, loaded := mgr.reg(svr.key, svr); loaded {
		return old.(*Svr), &Error{Code: ErrorAlreadyHad}
	}
	// 确定能注册成功则更新mod，以及启动协程
	svr.mod = mod
	err := svr.start(mgr)
	if err != nil {
		// 启动失败反注册
		mgr.unreg(svr.key)
		return nil, err
	}
	atomic.AddInt32(&mgr.countSvr, 1)
	return svr, nil
}

func (mgr *Mgr) svrBase(k interface{}, mod SvrBehavior) *Svr {
	key := svrKey(k)
	svr := &Svr{key: key}
	return svr
}

// 通知管理器，newSvr 结束了
func (mgr *Mgr) svrTerminate(svr *Svr) {
	atomic.AddInt32(&mgr.countSvr, -1)
	mgr.unreg(svr.key)
}

// 注册到管理器中
func (mgr *Mgr) reg(key interface{}, val interface{}) (interface{}, bool) {
	actual, ok := mgr.m.LoadOrStore(key, val)
	return actual, ok
}

// 删除注册信息
func (mgr *Mgr) unreg(key interface{}) {
	if key != nil {
		mgr.m.Delete(key)
	}
}

// 查询是否有这个协程
// Load returns the value stored in the map for a key, or nil if no
// value is present.
// The ok result indicates whether value was found in the map.
func (mgr *Mgr) lookup(key interface{}) (val interface{}, ok bool) {
	mgr.lock.RLock()
	defer func() {
		mgr.lock.RUnlock()
	}()
	return mgr.m.Load(key)
}

func (mgr *Mgr) foreach(f func(k interface{}, v interface{}) bool) {
	mgr.lock.RLock()
	defer func() {
		mgr.lock.RUnlock()
	}()
	mgr.m.Range(f)
}
