package express

/*
#include "express_c_interface.h"
#cgo LDFLAGS: -ldl

*/
import "C"
import (
	"header"
	"os"
	"path/filepath"
	"sync"
	"unsafe"
)

//版本号
var (
	harqVersion string = "1.0.1"
)

/* Express 接口函数 */

//export OnLogin
func OnLogin(express_handle int, remote_ip string, remote_port int, session string) {

	if gExpressClient != nil {
		gExpressClient.LoginEvent(express_handle, remote_ip, remote_port, session)
	}
}

//export OnProgress
func OnProgress(express_handle int, file_path string, max int, cur int) bool {

	if gExpressClient != nil {
		return gExpressClient.ProgressEvent(express_handle, file_path, max, cur)
	}

	return true
}

//export OnFinish
func OnFinish(express_handle int, file_path string, size int64) {

	if gExpressClient != nil {
		gExpressClient.FinishEvent(express_handle, file_path, size)
	}
}

//export OnDisconnect
func OnDisconnect(express_handle int, remote_ip string, remote_port int) {

	if gExpressClient != nil {
		gExpressClient.DisconnectEvent(express_handle, remote_ip, remote_port)
	}
}

//export OnClientError
func OnClientError(express_handle int, errorId int, remote_ip string, remote_port int) {

	if gExpressClient != nil {
		gExpressClient.ErrorEvent(express_handle, errorId, remote_ip, remote_port)
	}
}

//定义回调函数
type LoginCallBack func(express_handle int, remote_ip string, remote_port int, session string)
type ProgressCallBack func(express_handle int, file_path string, max int, cur int) bool
type FinishCallBack func(express_handle int, file_path string, size int64)
type DisconnectCallBack func(express_handle int, remote_ip string, remote_port int)
type ErrorCallBack func(express_handle int, err int, remote_ip string, remote_port int)

type Object interface{}

type ExpressClient struct {

	//参数信息
	bindIp      string
	remoteIp    string
	remotePort  int
	logPath     string
	harqPath    string
	libraryPath string
	session     string
	encrypted   bool

	//返回信息
	expressHandle int

	//回调函数
	HandleLogin      LoginCallBack
	HandleProgress   ProgressCallBack
	HandleFinish     FinishCallBack
	HandleDisconnect DisconnectCallBack
	HandleError      ErrorCallBack
}

var gExpressClient *ExpressClient

func NewClient(bindIp string, remoteIp string, remotePort int, logPath string, harqPath string, libraryPath string, session string, encrypted bool) *ExpressClient {

	expressClient := &ExpressClient{
		bindIp:      bindIp,
		remoteIp:    remoteIp,
		remotePort:  remotePort,
		logPath:     logPath,
		harqPath:    harqPath,
		libraryPath: libraryPath,
		session:     session,
		encrypted:   encrypted,
	}
	gExpressClient = expressClient
	return gExpressClient
}

func (c *ExpressClient) SetCallBack(handleLogin LoginCallBack,
	handleProgress ProgressCallBack,
	handleFinish FinishCallBack,
	handleDisconnect DisconnectCallBack,
	handleError ErrorCallBack) {

	c.HandleLogin = handleLogin
	c.HandleProgress = handleProgress
	c.HandleFinish = handleFinish
	c.HandleDisconnect = handleDisconnect
	c.HandleError = handleError
}

func (c *ExpressClient) StartClient() (expressHandle int) {

	var absPath string
	if filepath.IsAbs(c.libraryPath) {
		absPath = c.libraryPath
	} else {
		dir, _ := os.Getwd()
		absPath = dir + "/" + c.libraryPath
	}

	//初始化
	initRet := C.init_client(C.CString(absPath))
	inited := (*bool)(unsafe.Pointer(&initRet))
	if !(*inited) {
		return -1
	}

	//启动客户端
	startRet := C.start_client(C.CString(c.bindIp), C.CString(c.remoteIp), C.int(c.remotePort), C.CString(c.logPath), C.CString(c.harqPath), C.CString(c.session), C.bool(c.encrypted))
	handle := (*int)(unsafe.Pointer(&startRet))
	if *handle <= 0 {
		return *handle
	} else {
		c.expressHandle = *handle
		return c.expressHandle
	}
}

func (c *ExpressClient) SendFile(file_path string, save_relative_path string) int {

	//检测句柄是否存在
	if c.expressHandle <= 0 {
		return -1
	}

	//检测文件是否存在
	if exist, err := header.FileExists(file_path); err == nil {

		if !exist {
			return -2
		}

		ret := C.send_file(C.int(c.expressHandle), C.CString(file_path), C.CString(save_relative_path))
		resultInt := (*int)(unsafe.Pointer(&ret))
		return *resultInt
	}
	return -3
}

func (c *ExpressClient) SendDir(dir_path string, save_relative_path string) int {

	//检测句柄是否存在
	if c.expressHandle <= 0 {
		return -1
	}

	//检测文件是否存在
	if exist, err := header.DirExists(dir_path); err == nil {

		if !exist {
			return -2
		}

		ret := C.send_dir(C.int(c.expressHandle), C.CString(dir_path), C.CString(save_relative_path))
		resultInt := (*int)(unsafe.Pointer(&ret))
		return *resultInt
	}
	return -3
}

func (c *ExpressClient) StopSendFile() int {

	return 1
}

func (c *ExpressClient) CloseClient(linker_handle int) {

}

func (c *ExpressClient) Version() string {

	return C.GoString(C.version())
}

func (c *ExpressClient) LoginEvent(express_handle int, remote_ip string, remote_port int, session string) {

	if c.HandleLogin != nil {
		c.HandleLogin(express_handle, remote_ip, remote_port, session)
	}
}

func (c *ExpressClient) ProgressEvent(express_handle int, file_path string, max int, cur int) bool {

	if c.HandleProgress != nil {
		return c.HandleProgress(express_handle, file_path, max, cur)
	}
	return true
}

func (c *ExpressClient) FinishEvent(express_handle int, file_path string, size int64) {

	if c.HandleFinish != nil {
		c.HandleFinish(express_handle, file_path, size)
	}
}

func (c *ExpressClient) DisconnectEvent(express_handle int, remote_ip string, remote_port int) {

	if c.HandleDisconnect != nil {
		c.HandleDisconnect(express_handle, remote_ip, remote_port)
	}
}

func (c *ExpressClient) ErrorEvent(express_handle int, errorId int, remote_ip string, remote_port int) {

	if c.HandleError != nil {
		c.HandleError(express_handle, errorId, remote_ip, remote_port)
	}
}

//******************
type ExpressClientManager struct {
	logPath     string
	harqPath    string
	libraryPath string

	clientLock sync.Mutex
	ClientMap  map[int]*ExpressClient
}

func New(logPath string, harqPath string, libraryPath string) *ExpressClientManager {

	manager := &ExpressClientManager{
		logPath:     logPath,
		harqPath:    harqPath,
		libraryPath: libraryPath,
		ClientMap:   make(map[int]*ExpressClient),
	}
	return manager
}

func (e *ExpressClientManager) addClient(bindIp string, remoteIp string, remotePort int, encrypted bool) *ExpressClient {

	e.clientLock.Lock()
	defer e.clientLock.Unlock()

	client := NewClient(bindIp, remoteIp, remotePort, e.logPath, e.harqPath, e.libraryPath, "123456789", encrypted)
	client.SetCallBack(OnLogin, OnProgress, OnFinish, OnDisconnect, OnClientError)
	handle := client.StartClient()
	if handle <= 0 {
		return nil
	}
	e.ClientMap[handle] = client
	return client
}

func (e *ExpressClientManager) findClientFromHandle(expressHandle int) *ExpressClient {

	e.clientLock.Lock()
	defer e.clientLock.Unlock()

	if client, Ok := e.ClientMap[expressHandle]; Ok {
		return client
	} else {
		return nil
	}
}

func (e *ExpressClientManager) findClientFromAddr(remoteIp string, remotePort int) *ExpressClient {

	e.clientLock.Lock()
	defer e.clientLock.Unlock()

	for _, v := range e.ClientMap {

		if v.remoteIp == remoteIp && v.remotePort == remotePort {
			return v
		}
	}
	return nil
}

func (e *ExpressClientManager) SendFile(bindIp string, remoteIp string, remotePort int, encrypted bool, filePath string, saveRelativePath string) int {

	//判断文件是否存在
	if exist, err := header.FileExists(filePath); err == nil {

		if !exist {
			return -2
		}

		client := e.findClientFromAddr(remoteIp, remotePort)
		if nil == client {
			client = e.addClient(bindIp, remoteIp, remotePort, encrypted)
		}
		return client.SendFile(filePath, saveRelativePath)
	}
	return -1
}

func (e *ExpressClientManager) SendDir(bindIp string, remoteIp string, remotePort int, encrypted bool, dirPath string, saveRelativePath string) int {

	//判断文件是否存在
	if exist, err := header.DirExists(dirPath); err == nil {

		if !exist {
			return -2
		}

		client := e.findClientFromAddr(remoteIp, remotePort)
		if nil == client {
			client = e.addClient(bindIp, remoteIp, remotePort, encrypted)
		}
		return client.SendDir(dirPath, saveRelativePath)
	}
	return -1
}
