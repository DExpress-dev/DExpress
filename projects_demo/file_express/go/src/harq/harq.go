package harq

/*
#include "harq_c_interface.h"
#cgo LDFLAGS: -ldl

*/
import "C"
import (
	"os"
	"path/filepath"
	"unsafe"
)

//版本号
var (
	harqVersion string = "1.0.1"
)

/* Harq接口函数 */

//export OnConnect
func OnConnect(remote_ip string, remote_port int, linker_handle int, time_stamp int64) {

	if gHarq != nil {
		gHarq.ConnectEvent(remote_ip, remote_port, linker_handle, time_stamp)
	}
}

//export OnReceive
func OnReceive(data []byte, size int, linker_handle int, remote_ip string, remote_port int, consume_timer int) bool {

	if gHarq != nil {
		return gHarq.ReceiveEvent(data, size, linker_handle, remote_ip, remote_port, consume_timer)
	}

	return true
}

//export OnDisconnect
func OnDisconnect(linker_handle int, remote_ip string, remote_port int) {

	if gHarq != nil {
		gHarq.DisconnectEvent(linker_handle, remote_ip, remote_port)
	}
}

//export OnError
func OnError(errorId int, linker_handle int, remote_ip string, remote_port int) {

	if gHarq != nil {
		gHarq.ErrorEvent(errorId, linker_handle, remote_ip, remote_port)
	}
}

//export OnRto
func OnRto(remote_ip string, remote_port int, local_rto int, remote_rto int) {

	if gHarq != nil {
		gHarq.RtoEvent(remote_ip, remote_port, local_rto, remote_rto)
	}
}

//export OnRate
func OnRate(remote_ip string, remote_port int, send_rate int, recv_rate int) {

	if gHarq != nil {
		gHarq.RateEvent(remote_ip, remote_port, send_rate, recv_rate)
	}
}

//定义回调函数
type ConnectCallBack func(remote_ip string, remote_port int, linker_handle int, time_stamp int64)
type ReceiveCallBack func(data []byte, size int, linker_handle int, remote_ip string, remote_port int, consume_timer int) bool
type DisconnectCallBack func(linker_handle int, remote_ip string, remote_port int)
type ErrorCallBack func(errorId int, linker_handle int, remote_ip string, remote_port int)
type RtoCallBack func(remote_ip string, remote_port int, local_rto int, remote_rto int)
type RateCallBack func(remote_ip string, remote_port int, send_rate int, recv_rate int)

type Harq struct {
	remoteIp    string
	remotePort  int
	libraryPath string
	logPath     string
	handle      int

	HandleConnect    ConnectCallBack
	HandleRecv       ReceiveCallBack
	HandleDisconnect DisconnectCallBack
	HandleError      ErrorCallBack
	HandleRto        RtoCallBack
	HandleRate       RateCallBack
}

var gHarq *Harq

func New(libraryPath string, remoteIp string, remotePort int, logPath string) *Harq {

	harq := &Harq{
		remoteIp:    remoteIp,
		remotePort:  remotePort,
		libraryPath: libraryPath,
		logPath:     logPath,
		handle:      -1,
	}
	gHarq = harq
	return harq
}

func (h *Harq) SetCallBack(handleConnect ConnectCallBack,
	handleRecv ReceiveCallBack,
	handleDisconnect DisconnectCallBack,
	handleError ErrorCallBack,
	handleRto RtoCallBack,
	handleRate RateCallBack) {

	h.HandleConnect = handleConnect
	h.HandleRecv = handleRecv
	h.HandleDisconnect = handleDisconnect
	h.HandleError = handleError
	h.HandleRto = handleRto
	h.HandleRate = handleRate
}

func (h *Harq) StartClient(encrypted bool) (handle int) {

	var absPath string
	if filepath.IsAbs(h.libraryPath) {
		absPath = h.libraryPath
	} else {
		dir, _ := os.Getwd()
		absPath = dir + "/" + h.libraryPath
	}

	//初始化
	initRet := C.init_client(C.CString(absPath))
	inited := (*bool)(unsafe.Pointer(&initRet))
	if !(*inited) {
		return -1
	}

	//启动客户端
	startRet := C.start_client(C.CString(h.logPath), C.CString(h.remoteIp), C.int(h.remotePort), C.bool(encrypted))
	startHandle := (*int)(unsafe.Pointer(&startRet))
	if *startHandle <= 0 {
		return *startHandle
	} else {
		h.handle = *startHandle
		return h.handle
	}
}

func (h *Harq) SendBuffer(data []byte, size int) int {

	if h.handle <= 0 {
		return -1
	}

	ret := C.send_buffer((*C.char)(unsafe.Pointer(&data[0])), C.int(size), C.int(h.handle))
	resultInt := (*int)(unsafe.Pointer(&ret))
	return *resultInt
}

func (h *Harq) CloseClient(linker_handle int) {

	if h.handle <= 0 {
		return
	}
	C.close_client(C.int(linker_handle))
}

func (h *Harq) Version() string {

	return C.GoString(C.version())
}

func (h *Harq) Version2() string {

	return C.GoString(C.version())
}

func (h *Harq) ConnectEvent(remote_ip string, remote_port int, linker_handle int, time_stamp int64) {

	if h.HandleConnect != nil {
		h.HandleConnect(remote_ip, remote_port, linker_handle, time_stamp)
	}
}

func (h *Harq) ReceiveEvent(data []byte, size int, linker_handle int, remote_ip string, remote_port int, consume_timer int) bool {

	if h.HandleRecv != nil {
		return h.HandleRecv(data, size, linker_handle, remote_ip, remote_port, consume_timer)
	}
	return true
}

func (h *Harq) DisconnectEvent(linker_handle int, remote_ip string, remote_port int) {

	if h.HandleDisconnect != nil {
		h.HandleDisconnect(linker_handle, remote_ip, remote_port)
	}
}

func (h *Harq) ErrorEvent(errorId int, linker_handle int, remote_ip string, remote_port int) {

	if h.HandleError != nil {
		h.HandleError(errorId, linker_handle, remote_ip, remote_port)
	}
}

func (h *Harq) RtoEvent(remote_ip string, remote_port int, local_rto int, remote_rto int) {

	if h.HandleRto != nil {
		h.HandleRto(remote_ip, remote_port, local_rto, remote_rto)
	}
}

func (h *Harq) RateEvent(remote_ip string, remote_port int, send_rate int, recv_rate int) {

	if h.HandleRate != nil {
		h.HandleRate(remote_ip, remote_port, send_rate, recv_rate)
	}
}
