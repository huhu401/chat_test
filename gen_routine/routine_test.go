package gen_routine

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRootMgr(t *testing.T) {
	cleanEnv()
	assert.NotNil(t, RootMgr())
}

func TestNewMgr(t *testing.T) {
	cleanEnv()
	mgr, err := NewMgr(RootMgr(), "player mgr")
	assert.Nil(t, err)
	assert.NotNil(t, mgr)
	mgr1, err1 := NewMgr(RootMgr(), "player mgr")
	assert.Equal(t, mgr, mgr1)
	assert.NotNil(t, err1)
	mgr, err = NewMgr(mgr, "player bag mgr")
	assert.Nil(t, err)
	assert.NotNil(t, mgr)
}

func TestLookupMgr(t *testing.T) {
	initMgr(t)
	v, ok := LookupMgr(RootMgr(), "player mgr")
	assert.True(t, ok)
	assert.IsType(t, v, &Mgr{})

	subName := "bag mgr"

	pMgr := v.(*Mgr)

	mgr, err := NewMgr(pMgr, subName)
	assert.Nil(t, err)
	assert.NotNil(t, mgr)

	v, ok = LookupMgr(RootMgr(), subName)
	assert.Nil(t, v)
	assert.False(t, ok)

	v, ok = LookupMgr(pMgr, subName)
	assert.NotNil(t, v)
	assert.True(t, ok)
}

func TestMgr_StopMgr(t *testing.T) {
	svr, _ := initSvr(t)
	time.Sleep(time.Second)
	svr.mgr.StopMgr(&Error{Code: ErrorNormalStop})
	// terminate 正确调用了
	mod := svr.GetMod().(*svrBehavior)
	assert.True(t, mod.terminated)
}

func TestMgr_NewSvr(t *testing.T) {
	initMgr(t)
	v, _ := LookupMgr(RootMgr(), "player mgr")
	pMgr := v.(*Mgr)
	svr, err := pMgr.NewSvr(nil, &svrBehavior{})
	f :=
		func(s *Svr, e *Error, k interface{}) {
			assert.Nil(t, e)
			assert.NotNil(t, s.receive)
			assert.NotNil(t, s.mgr)
			assert.NotNil(t, s.mod)
			assert.Equal(t, k, s.key)
		}
	var a uint64 = 1
	f(svr, err, a)
	mod := &svrBehavior{}
	k := mod.Key()
	svr, err = pMgr.NewSvr(mod.Key(), mod)
	f(svr, err, k)
	svr1, err1 := pMgr.NewSvr(mod.Key(), mod)
	assert.Equal(t, svr, svr1)
	assert.Equal(t, err1.Code, ErrorAlreadyHad)
	assert.Equal(t, pMgr.countSvr, int32(2))

	svr, err = pMgr.NewSvr("init err", &svrBehavior{})
	assert.Nil(t, svr)
	assert.NotNil(t, err)
	//t.Logf("behavior init err ret %s", err.String())
	s, ok := pMgr.LookupSvr("init err")
	assert.Nil(t, s)
	assert.False(t, ok)
	svr, err = pMgr.NewSvr("init crash", &svrBehavior{})
	assert.Nil(t, svr)
	assert.NotNil(t, err)
	//t.Logf("behavior init crash ret %s", err.String())
	s, ok = pMgr.LookupSvr("init crash")
	assert.Nil(t, s)
	assert.False(t, ok)
}

func BenchmarkMgr_NewSvr(b *testing.B) {
	initRootMgr()
	NewMgr(RootMgr(), "player mgr")
	b.ResetTimer()
	v, _ := LookupMgr(RootMgr(), "player mgr")
	pMgr := v.(*Mgr)
	for i := 0; i < b.N; i++ {
		_, err := pMgr.NewSvr(i, &svrBehavior{})
		if err != nil {
			panic(err.String())
		}
	}
	pMgr.StopMgr(&Error{Code: ErrorNormalStop})
}

func TestSvr_Cast(t *testing.T) {
	svr, _ := initSvr(t)
	msg := "echo"
	svr.Cast(msg)
	time.Sleep(time.Second)
	s := svr.mod.(*svrBehavior)
	assert.Equal(t, s.echo, msg)
}

func TestSvr_Call(t *testing.T) {
	svr, _ := initSvr(t)
	msg := "call_echo"
	ret, err := svr.Call(msg, Infinity)
	assert.Nil(t, err)
	s := svr.mod.(*svrBehavior)
	assert.Equal(t, s.echo, msg)
	assert.Equal(t, s.echo, ret)

	msg = "call_wait"
	ret, err = svr.Call(msg, time.Nanosecond)
	assert.Equal(t, err.Code, ErrorTimeout)
	assert.NotEqual(t, s.echo, msg)
	assert.Nil(t, ret)
}

func TestSvr_StopSvr(t *testing.T) {
	svr, _ := initSvr(t)
	svr.StopSvr(&Error{Code: ErrorNormalStop})
	time.Sleep(time.Second)
	// 从管理器中删除了
	s, ok := svr.mgr.LookupSvr(svr.key)
	assert.False(t, ok)
	assert.Nil(t, s)

	// terminate 正确调用了
	mod := svr.mod.(*svrBehavior)
	assert.True(t, mod.terminated)
}

func TestSvr_CallInfinity(t *testing.T) {
	svr, _ := initSvr(t)
	msg := "call_echo"
	ret, err := svr.CallInfinity(msg)
	assert.Nil(t, err)
	s := svr.mod.(*svrBehavior)
	assert.Equal(t, s.echo, msg)
	assert.Equal(t, s.echo, ret)
}

func TestJustRun(t *testing.T) {
	b := 1
	JustRun(func() {
		b = 2
	})
	assert.NotEqual(t, b, 2)
	time.Sleep(time.Second)
	assert.Equal(t, b, 2)

	JustRun(func() {
		b = 3
		panic("crash just run")
	})
	time.Sleep(time.Second)
	assert.Equal(t, b, 3)

	l := &TestLogger{}
	SetLogger(l)
	JustRun(func() {
		b = 4
		panic("crash just run 3")
	})
	time.Sleep(time.Second)
	assert.Equal(t, b, 4)
	assert.True(t, l.in)
}

type TestLogger struct {
	in bool
}

func (l *TestLogger) Errorf(str string, args ...interface{}) {
	l.in = true
	log.Printf(str, args...)
}

func TestMgr_Foreach(t *testing.T) {
	svr, _ := initSvr(t)
	i := 0
	svr.mgr.Foreach(func(k interface{}, v interface{}) bool {
		assert.Equal(t, k, svr.key)
		assert.Equal(t, v, svr)
		i++
		return false
	})
	assert.Equal(t, i, 1)
}

func TestMgr_LookupSvr(t *testing.T) {
	svr, _ := initSvr(t)
	svr1, ok := svr.mgr.LookupSvr(svr.key)
	assert.True(t, ok)
	assert.Equal(t, svr, svr1)
	svr1, ok = svr.mgr.LookupSvr(1)
	assert.False(t, ok)
}

func TestError_String(t *testing.T) {
	svr, _ := initSvr(t)
	crash, err := svr.CallInfinity("crash")
	err.String()
	assert.Nil(t, crash)
	assert.NotNil(t, err)
	time.Sleep(time.Second)
	assert.Equal(t, svr.mgr.countSvr, int32(0))
	err = &Error{Code: ErrorClosed, Last: err}
	err.String()
}

// TestIntegration 协程管理的批量运行测试
func TestIntegration(t *testing.T) {
	initRootMgr()
	w := sync.WaitGroup{}
	const (
		makeMgr = 1 + iota
		makeSvr1
		makeSvr2
		makeSvr3
		makeSvr4
		svrCast1
		svrCast2
		svrCast3
		svrCast4
		svrCall1
		svrCall2
		svrCall3
		svrCall4
		max
	)
	statEchoCount := int32(0)
	statWaitEcho := int32(0)
	statCEchoCount := int32(0)
	statCWaitEcho := int32(0)
	funcGo := func(k int) {
		defer func() {
			w.Done()
		}()
		switch rand.Intn(max) {
		case makeMgr:
			NewMgr(RootMgr(), fmt.Sprintf("mgr %d", k))
		case makeSvr1, makeSvr2, makeSvr3, makeSvr4:
			var a []interface{}
			RootMgr().foreach(func(key, value interface{}) bool {
				a = append(a, value)
				return true
			})
			idx := rand.Intn(len(a))
			_, err := a[idx].(*Mgr).NewSvr(nil, &svrBehavior{})
			if err != nil {
				panic(err.String())
			}
		case svrCast1, svrCast2, svrCast3, svrCast4:
			for i := 0; i < 10; i++ {
				RootMgr().foreach(func(key, value interface{}) bool {
					value.(*Mgr).foreach(func(key1, value1 interface{}) bool {
						atomic.AddInt32(&statWaitEcho, 1)
						value1.(*Svr).Cast("echo")
						atomic.AddInt32(&statWaitEcho, -1)
						atomic.AddInt32(&statEchoCount, 1)
						return true
					})
					return true
				})
			}
		case svrCall1, svrCall2, svrCall3, svrCall4:
			for i := 0; i < 10; i++ {
				RootMgr().foreach(func(key, value interface{}) bool {
					value.(*Mgr).foreach(func(key1, value1 interface{}) bool {
						atomic.AddInt32(&statCWaitEcho, 1)
						value1.(*Svr).Call("call_echo", Infinity)
						atomic.AddInt32(&statCWaitEcho, -1)
						atomic.AddInt32(&statCEchoCount, 1)
						return true
					})
					return true
				})
			}
		}
	}
	NewMgr(RootMgr(), fmt.Sprintf("mgr %d", 0))
	for i := 1; i <= 1000; i++ {
		w.Add(1)
		go funcGo(i)
	}
	ticker := time.NewTicker(time.Second * 1)
	stopTicker := make(chan struct{})

	timeStart := time.Now()
	fStat := func() {
		c := int32(0)
		mgrG.foreach(func(k interface{}, v interface{}) bool {
			c = c + v.(*Mgr).countSvr
			return true
		})
		t.Logf("%s stat info 已完成cast echo次数：%d echo cast 卡着的数量：%d 已完成call echo次数：%d echo call 卡着的数量：%d 管理器数量：%d 总的svr数量：%d",
			time.Since(timeStart), statEchoCount, statWaitEcho, statCEchoCount, statCWaitEcho, mgrG.countMgr, c)
	}

	go func() {
		for {
			select {
			case <-stopTicker:
				return
			case <-ticker.C:
				fStat()
			}
		}
	}()
	w.Wait()
	close(stopTicker)
	fStat()
}

type svrBehavior struct {
	echo       string
	terminated bool

	execRecord map[string]string
}

const (
	testSvrKey = "svrKey"
)

func (s *svrBehavior) Key() interface{} {
	return testSvrKey
}

func (s *svrBehavior) HandleMsg(msg Msg) (interface{}, *Error) {
	switch v := msg.(type) {
	case string:
		switch v {
		case "echo":
			s.echo = v
			return v, nil
		case "call_echo":
			s.echo = v
			return v, nil
		case "call_wait":
			time.Sleep(time.Second)
			s.echo = v
			return v, nil
		case "crash":
			panic("test crash")
		}
	}
	fmt.Printf("received unknow msg %v\n", msg)
	return nil, nil
}

func (s *svrBehavior) Init(svr *Svr) *Error {
	if svr.key == "init err" {
		return &Error{Code: ErrorCrash}
	}
	if svr.key == "init crash" {
		panic("init crash")
	}
	return nil
}

func (s *svrBehavior) Terminate(reason *Error) {
	s.terminated = true
	if reason.Code == ErrorCrash {
		panic("test terminate crash")
	}
}

func (s *svrBehavior) ExecNoArgsNoRet() {
	s.execRecord["ExecNoArgsNoRet"] = "invoked"
}

func (s *svrBehavior) ExecOneArgsOneRet(in string) string {
	s.execRecord["ExecOneArgsOneRet"] = in
	return "ExecOneArgsOneRet " + in
}

func (s *svrBehavior) ExecSomeArgsSomeRet(in1 string, in2 []string) (string, error) {
	s.execRecord["ExecSomeArgsSomeRet"] = in1 + fmt.Sprintf("%v", in2)
	return "ExecSomeArgsSomeRet", errors.New("error ret")
}

var gExecRecord map[string]string

func execNoArgsNoRet() {
	gExecRecord["execNoArgsNoRet"] = "invoked"
}

func execSomeArgsSomeRet(in1 string, in2 int32, in3 []bool) (string, error, int) {
	gExecRecord["execSomeArgsSomeRet"] = fmt.Sprintf("%s-%d-%v", in1, in2, in3)
	return "execSomeArgsSomeRet", errors.New("error ret"), 1
}

func TestSvr_Grp(t *testing.T) {
	initGrp()
	// 直接拉空的出来
	assert.Equal(t, 0, len(GrpAll("")))

	svr, _ := initSvr(t)
	grpKey := "test grp"
	svr.GrpReg(grpKey)
	svr.GrpReg(grpKey)
	assert.Equal(t, grp.svrGrp[grpKey][svr.key], svr)
	assert.Equal(t, grp.svrKey[svr.key][grpKey], true)

	svr1, _ := svr.mgr.NewSvr(nil, &svrBehavior{})
	// 俩 svr 的 key 不一样
	assert.NotEqual(t, svr.key, svr1.key)

	svr1.GrpReg(grpKey)
	all := GrpAll(grpKey)
	sort.Slice(all, func(a, b int) bool { return all[a].key.(uint64) < all[b].key.(uint64) })
	// 同一组里，俩 svr 都在
	assert.Equal(t, []*Svr{svr, svr1}, all)

	svr.GrpByeBye()
	// 两个map里都清理掉了
	_, ok := grp.svrGrp[grpKey][svr.key]
	assert.False(t, ok)
	_, ok = grp.svrGrp[svr.key]
	assert.False(t, ok)

	// 全部里面只有1个
	all = GrpAll(grpKey)
	assert.Equal(t, []*Svr{svr1}, all)

	svr1.GrpUnReq(grpKey)
	// 都没了
	_, ok = grp.svrGrp[grpKey]
	assert.False(t, ok)
	_, ok = grp.svrKey[svr1.key]
	assert.False(t, ok)

	// 删除不存在的
	svr.GrpUnReq(grpKey)
	svr.GrpUnReq("")
	svr.GrpByeBye()

	// 走bye清理完所有的，清理grpKey
	svr.GrpReg(grpKey)
	svr.GrpByeBye()
}

func TestSvr_Exec(t *testing.T) {
	svr, _ := initSvr(t)
	mod := svr.mod.(*svrBehavior)
	// 调用 mod method
	ret, err := svr.SyncExec(mod.ExecNoArgsNoRet, Infinity)
	assert.Nil(t, err)
	assert.Equal(t, len(ret), 0)
	assert.Equal(t, mod.execRecord["ExecNoArgsNoRet"], "invoked")
	ret, err = svr.SyncExec(mod.ExecOneArgsOneRet, Infinity, "in1")
	assert.Nil(t, err)
	assert.Equal(t, len(ret), 1)
	assert.Equal(t, "ExecOneArgsOneRet in1", ret[0].(string))
	ret, err = svr.SyncExec(mod.ExecSomeArgsSomeRet, Infinity, "in1", []string{"string slice"})
	assert.Nil(t, err)
	assert.Equal(t, len(ret), 2)
	assert.Equal(t, "ExecSomeArgsSomeRet", ret[0].(string))
	assert.Equal(t, "error ret", ret[1].(error).Error())
	// 调用 package function
	gExecRecord = map[string]string{}
	ret, err = svr.SyncExec(execNoArgsNoRet, Infinity)
	assert.Nil(t, err)
	assert.Equal(t, len(ret), 0)
	assert.Equal(t, gExecRecord["execNoArgsNoRet"], "invoked")
	ret, err = svr.SyncExec(execSomeArgsSomeRet, Infinity, "in1", int32(1), []bool{true, false})
	assert.Nil(t, err)
	assert.Equal(t, len(ret), 3)
	assert.Equal(t, "execSomeArgsSomeRet", ret[0].(string))
	assert.Equal(t, "error ret", ret[1].(error).Error())
	assert.Equal(t, 1, ret[2].(int))
	// 非阻塞调用
	gExecRecord = map[string]string{}
	err = svr.ASyncExec(execNoArgsNoRet)
	assert.Nil(t, err)
	svr.ASyncExec(mod.ExecSomeArgsSomeRet, "in1", []string{"string slice"})
	time.Sleep(time.Second)
	assert.Equal(t, 1, len(gExecRecord))
	assert.Equal(t, "invoked", gExecRecord["execNoArgsNoRet"])
	assert.Equal(t, "in1[string slice]", mod.execRecord["ExecSomeArgsSomeRet"])
	// 错误参数数量调用
	err = svr.ASyncExec(mod.ExecSomeArgsSomeRet, []string{"string slice"})
	assert.NotNil(t, err)
	ret, err = svr.SyncExec(mod.ExecSomeArgsSomeRet, Infinity, []string{"string slice"})
	assert.NotNil(t, err)
	assert.Nil(t, ret)
}

func initMgr(t *testing.T) {
	cleanEnv()
	t.Cleanup(cleanEnv)
	NewMgr(RootMgr(), "player mgr")
}

func initSvr(t *testing.T) (*Svr, *Error) {
	initMgr(t)
	v, _ := LookupMgr(RootMgr(), "player mgr")
	pMgr := v.(*Mgr)
	return pMgr.NewSvr(nil, &svrBehavior{
		execRecord: map[string]string{},
	})
}

func cleanEnv() {
	BeforeMain()
}
