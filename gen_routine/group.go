package gen_routine

import "sync"

// 分组里面有哪些svr
type subMap map[interface{}]*Svr

// 分组map
type groupMap map[interface{}]subMap

// svr 在哪些分组里的 map, key : svr.key value : []group key
type svrMap map[interface{}]map[interface{}]bool

type group struct {
	svrGrp groupMap
	svrKey svrMap
	mux    sync.Mutex
}

var grp *group

func initGrp() {
	grp = &group{
		svrGrp: groupMap{},
		svrKey: svrMap{},
		mux:    sync.Mutex{},
	}
}

// 将协程主测到一个分组当中
func reg(key interface{}, svr *Svr) {
	defer func() {
		grp.mux.Unlock()
	}()
	grp.mux.Lock()
	svrM, ok := grp.svrGrp[key]
	if !ok {
		svrM = subMap{}
		grp.svrGrp[key] = svrM
	}
	svrM[svr.key] = svr
	a, ok := grp.svrKey[svr.key]
	if !ok {
		a = map[interface{}]bool{}
		grp.svrKey[svr.key] = a
	}
	a[key] = true
}

// 将协程从某个分组中删除
func unReq(key interface{}, svr *Svr) {
	defer func() {
		grp.mux.Unlock()
	}()
	grp.mux.Lock()

	// 从服务在哪些组里删除
	if subSvr, ok1 := grp.svrKey[svr.key]; ok1 {
		delete(subSvr, key)
		if len(subSvr) == 0 {
			delete(grp.svrKey, svr.key)
		}
	}

	// 从哪些组里有服务删除
	subM, ok := grp.svrGrp[key]
	if !ok {
		return
	}
	delete(subM, svr.key)
	if len(subM) == 0 {
		delete(grp.svrGrp, key)
	}
}

// 协程退出，清理所有相关记录
func bye(svr *Svr) {
	defer func() {
		grp.mux.Unlock()
	}()
	grp.mux.Lock()
	if svrM, ok1 := grp.svrKey[svr.key]; ok1 {
		for grpK, _ := range svrM {
			if m, ok := grp.svrGrp[grpK]; ok {
				delete(m, svr.key)
				if len(m) == 0 {
					delete(grp.svrGrp, grpK)
				}
			}
		}
	}
	delete(grp.svrKey, svr.key)
}

// allSvr 返回某个组里所有的svr
func allSvr(key interface{}) []*Svr {
	var a []*Svr
	if subM, ok := grp.svrGrp[key]; ok {
		for _, svr := range subM {
			a = append(a, svr)
		}
		return a
	}
	return a
}
