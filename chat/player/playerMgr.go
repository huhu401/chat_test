package player

import (
	"github.com/huhu401/chat_test/constant"
	"github.com/huhu401/chat_test/gen_routine"
	"github.com/huhu401/chat_test/msg"
	"sync"
)

type Mgr struct {
	*gen_routine.Mgr
	sync.Mutex
}

var playerMgr *Mgr

func BeforeMain() {
	// 建立基础的管理器
	newManager()
}

func newManager() (*gen_routine.Mgr, *gen_routine.Error) {
	mgr, err := gen_routine.NewMgr(gen_routine.RootMgr(), "player rMgr")
	playerMgr = &Mgr{Mgr: mgr}
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

// GetManager 获取全局玩家manager
func GetManager() *Mgr {
	return playerMgr
}

// GetPlayer 获取玩家
func (mgr *Mgr) GetPlayer(roleId int64) *Player {
	v, ok := mgr.LookupSvr(roleId)
	if !ok {
		return nil
	}
	svr := v.(*gen_routine.Svr)
	return svr.GetMod().(*Player)
}

// Login 玩家登陆
func (mgr *Mgr) Login(req *msg.ReqMsgLogin) *Player {
	rsp := &msg.RspMsgLogin{Status: constant.ErrorNo}
	p := NewPlayer(req)
	old, err := mgr.NewSvr(p.Key(), p)
	if err != nil && err.Code != gen_routine.ErrorAlreadyHad {
		rsp.Status = err.Code
	} else {
		if err != nil && err.Code == gen_routine.ErrorAlreadyHad {
			old.GetMod().(*Player).alreadyIn(req)
		}
	}
	return old.GetMod().(*Player)
}

func (mgr *Mgr) Logout(roleId int64) {
	p := mgr.GetPlayer(roleId)
	if p != nil {
		p.Stop(&gen_routine.Error{Code: gen_routine.ErrorGateOffline})
	}
}
