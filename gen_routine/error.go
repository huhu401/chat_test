package gen_routine

import (
	"fmt"
	"log"
)

type Error struct {
	Code       int32
	Param      string
	ParamPanic interface{}
	Last       *Error
}

var logger Logger

func errorf(fmt string, args ...interface{}) {
	if logger != nil {
		logger.Errorf(fmt, args...)
		return
	}
	log.Printf(fmt, args...)
}

func (e *Error) String() string {
	ret := ""
	if e.Last != nil {
		ret += e.Last.String()
		ret += "\n"
	}
	ret += fmt.Sprintf("Code : %d", e.Code)
	if e.Param != "" {
		ret += " info : " + e.Param
	}
	if e.ParamPanic != nil {
		ret += fmt.Sprintf("panic err : %v", e.ParamPanic)
	}
	return ret
}

const (
	ErrorCodeOk           = int32(0) - iota
	ErrorNotFind          = int32(-1)
	ErrorCtxDone          = int32(-2)
	ErrorCrash            = int32(-3)
	ErrorNormalStop       = int32(-4)
	ErrorTimeout          = int32(-5)
	ErrorClosed           = int32(-6)
	ErrorAlreadyHad       = int32(-7)
	ErrorGateOffline      = int32(-8)
	ErrorReflectParamsLen = int32(-9)  // 反射调用函数时，参数长度不足
	ErrorRoutineInitFail  = int32(-10) // 协程初始化失败
	ErrorPbUnmarshal      = int32(-11) // Protocol buff 反序列化失败
	ErrorRpc              = int32(-12) // rpc 调用过程中出错了
)
