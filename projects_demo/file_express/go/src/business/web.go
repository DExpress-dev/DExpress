package business

import (
	"encoding/json"
	"express"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	log4plus "log4go"

	"github.com/pochard/goutils"
)

var (
	serverConfigName = flag.String(
		"server",      //参数
		"server.json", //参数默认值
		"直播房间配置信息")
)

type FileProgress struct {
	RemoteIp   string `json:"remoteIp"`
	RemotePort int    `json:"remotePort"`
	FilePath   string `json:"path"`
	Max        int    `json:"max"`
	Cur        int    `json:"cur"`
}

type ResponseProgress struct {
	Result   int          `json:"result"`
	Message  string       `json:"message"`
	Progress FileProgress `json:"progress"`
}

type SendFileRequest struct {
	RemoteIp         string `json:"remoteIp"`
	RemotePort       int    `json:"remotePort"`
	BindIp           string `json:"bindIp"`
	Encrypted        bool   `json:"encrypted"`
	FilePath         string `json:"path"`
	SaveRelativePath string `json:"relativePath"`
}

type Web struct {
	listen        string
	clientManager express.ExpressClientManager
}

func handlerWrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers",
				"Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		}

		h.ServeHTTP(w, r)
	})
}

func (we *Web) responseResult(errorCode int, w http.ResponseWriter, errorId int, errorString string) {

	// //返回结构
	// var response ResponseResult
	// response.Result = errorId
	// response.Message = errorString
	// jsonData, _ := json.Marshal(response)
	// w.Header().Set("Content-Type", "application/json")

	// w.WriteHeader(errorCode)
	// w.Write(jsonData)
}

func (we *Web) HttpError(w http.ResponseWriter, result int, msg string) {
	w.Write([]byte(fmt.Sprintf("{\"result\":%d,\"msg\":\"%s\"}", result, msg)))
}

func (we *Web) getLocalIps() []string {

	if addrs, err := net.InterfaceAddrs(); err == nil {

		var Ips []string
		for _, address := range addrs {
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					Ips = append(Ips, ipnet.IP.String())
				}
			}
		}
		return Ips
	}
	return nil
}

func (we *Web) HttpSendFile(w http.ResponseWriter, r *http.Request) {

	log4plus.Info("---->>>>HttpSendFile")

	//判断请求端的IP地址，保证使用的是本机进行的请求
	clientIp := goutils.GetClientIP(r, "x-real-ip", "x-forwarded-for")
	Ips := we.getLocalIps()
	if len(Ips) <= 0 {

		log4plus.Error("HttpSendFile Failed Client Ips <= 0")
		we.responseResult(400, w, -7, "HttpSendFile Failed Client Ips <= 0")
		return
	}

	//检测发送请求的IP是否是本机
	var exists bool = false
	for _, ip := range Ips {

		if clientIp == ip {
			exists = true
			break
		}
	}
	if !exists {
		log4plus.Error("HttpSendFile Failed Client Ip Not Is Local Ip")
		we.responseResult(400, w, -7, "HttpSendFile Failed Client Ip Not Is Local Ip")
		return
	}

	//解析请求
	var request SendFileRequest
	result, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(result, &request)

	//发送文件
	_ = we.clientManager.SendFile(request.BindIp, request.RemoteIp, request.RemotePort, request.Encrypted, request.FilePath, request.SaveRelativePath)

	//返回给客户端

}

func New(listen string) *Web {

	web := &Web{
		listen: listen,
	}

	return web
}
