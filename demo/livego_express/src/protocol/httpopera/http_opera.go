package httpopera

import (
	"av"
	cmap "concurrent-map"
	"configure"
	"encoding/json"
	"errors"
	"fmt"
	_ "io/ioutil"
	log "logging"
	"net"
	"net/http"
	"protocol/rtmp"
	"protocol/rtmp/rtmprelay"
	"strconv"
	"sync"
	"time"
)

type Response struct {
	w       http.ResponseWriter
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (r *Response) SendJson() (int, error) {
	resp, _ := json.Marshal(r)
	r.w.Header().Set("Content-Type", "application/json")
	return r.w.Write(resp)
}

type Operation struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Stop   bool   `json:"stop"`
}

type OperationChange struct {
	Method    string `json:"method"`
	SourceURL string `json:"source_url"`
	TargetURL string `json:"target_url"`
	Stop      bool   `json:"stop"`
}

type ClientInfo struct {
	url              string
	rtmpRemoteClient *rtmp.Client
	rtmpLocalClient  *rtmp.Client
}

type ResponseResult struct {
	Result  int    `json:"result"`
	Message string `json:"message"`
}

type Server struct {
	handler       av.Handler
	session       map[string]*rtmprelay.RtmpRelay
	sessionFlv    map[string]*rtmprelay.FlvPull
	sessionMRelay cmap.ConcurrentMap
	mrelayMutex   sync.RWMutex
	rtmpAddr      string
}

func NewServer(h av.Handler, rtmpAddr string) *Server {
	return &Server{
		handler:       h,
		session:       make(map[string]*rtmprelay.RtmpRelay),
		sessionFlv:    make(map[string]*rtmprelay.FlvPull),
		sessionMRelay: cmap.New(),
		rtmpAddr:      rtmpAddr,
	}
}

type ReportStat struct {
	serverList  []string
	isStart     bool
	localServer *Server
}

type MRelayStart struct {
	Instancename string
	Dsturl       string
	Srcurlset    []rtmprelay.SrcUrlItem
	Buffertime   int
}

type MRelayAdd struct {
	Instanceid int64
	Srcurlset  []rtmprelay.SrcUrlItem
	Buffertime int
}
type MRelayReponse struct {
	Retcode      int
	Instanceid   int64
	Instancename string
	Dscr         string
}

var reportStatObj *ReportStat

func (s *Server) responseResult(w http.ResponseWriter, errorId int, errorString string) {

	//返回结构
	var response ResponseResult
	response.Result = errorId
	response.Message = errorString
	jsonData, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (s *Server) Serve(l net.Listener) error {

	mux := http.NewServeMux()

	//得到推流点
	mux.HandleFunc("/getPush", func(w http.ResponseWriter, r *http.Request) { s.handleGetPush(w, r) })
	//得到指定的回看地址信息
	mux.HandleFunc("/getReplay", func(w http.ResponseWriter, r *http.Request) { s.handleGetReplay(w, r) })
	//项目结束
	mux.HandleFunc("/stopProject", func(w http.ResponseWriter, r *http.Request) { s.handleStopProject(w, r) })
	//结束推流点
	mux.HandleFunc("/stopPush", func(w http.ResponseWriter, r *http.Request) { s.handleStopPush(w, r) })
	//获得列表信息
	mux.HandleFunc("/getCurrentList", func(w http.ResponseWriter, r *http.Request) { s.handleGetCurrentList(w, r) })
	//控制声音
	mux.HandleFunc("/setPushIdAudio", func(w http.ResponseWriter, r *http.Request) { s.handleSetAudioFromPushId(w, r) })

	reportStatObj = NewReportStat(configure.GetReportList(), s)
	err := reportStatObj.Start()
	if err != nil {
		log.Error("ReportStat start error:", err)
		return err
	}
	defer reportStatObj.Stop()

	http.Serve(l, mux)

	return nil
}

type Stream struct {
	Key             string `json:"key"`
	Url             string `json:"Url"`
	PeerIP          string `json:"PeerIP"`
	StreamId        uint32 `json:"StreamId"`
	VideoTotalBytes uint64 `json:123456`
	VideoSpeed      uint64 `json:123456`
	AudioTotalBytes uint64 `json:123456`
	AudioSpeed      uint64 `json:123456`
}

type Streams struct {
	PublisherNumber int64
	PlayerNumber    int64
	Publishers      []Stream `json:"publishers"`
	Players         []Stream `json:"players"`
}

/*
得到推流点

格式：
http://127.0.0.1:8090/getPush

参数：
projectId: 项目ID
userType: 用户类型

地址举例：
http://127.0.0.1:8070/getPush?&projectId=12&userType=0

主动获取推流点，不能是球机
*/
func (s *Server) handleGetPush(w http.ResponseWriter, req *http.Request) {

	newProject := false
	var err error
	req.ParseForm()

	//获得参数信息
	tmpProjectId := req.Form["projectId"]
	projectId, errInt := strconv.Atoi(tmpProjectId[0])
	if errInt != nil {

		errString := fmt.Sprintf("handleGetPush projectId Param error, please check them projectId=%s err=%s", tmpProjectId[0], errInt.Error())
		s.responseResult(w, 1, errString)
		log.Errorf("[Web Opera] handleGetPush projectId Param error, please check them ")
		return
	}
	tmpUserType := req.Form["userType"]
	userType, err := strconv.Atoi(tmpUserType[0])
	if err != nil {

		s.responseResult(w, 1, "userType Param error, please check them")
		log.Errorf("[Web Opera] handleGetPush userType Param error, please check them")
		return
	}
	tmpVideoType := req.Form["videoType"]
	videoType, err := strconv.Atoi(tmpVideoType[0])
	if err != nil {

		s.responseResult(w, 1, "videoType Param error, please check them")
		log.Errorf("[Web Opera] handleGetPush videoType Param error, please check them")
		return
	}
	log.Infof("[Web Opera] handleGetPush projectId=%d userType=%d videoType=%d", projectId, userType, videoType)

	//得到Rtmp流的管理对象
	rtmpStream := s.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {

		s.responseResult(w, 1, "Get rtmp Stream information error")
		log.Errorf("[Web Opera] handleGetPush Get rtmp Stream information error")
		return
	}

	//判断项目是否存在
	var errAlloc error
	var liveRoomId string
	ret := rtmpStream.CheckProjectExits(projectId)

	if ret {

		//项目存在，得到项目所对应的房间ID
		err, liveRoom := rtmpStream.GetLiveRoom(projectId)
		if err != nil {

			s.responseResult(w, 1, "Get Push Failed Not Found ProjectId")
			log.Errorf("[Web Opera] handleGetPush Get Push Failed Not Found ProjectId")
			return
		}
		liveRoomId = liveRoom.LiveRoomId

	} else {

		//判断是否所有直播间已经满了
		if rtmpStream.LiveRoomFull() {

			//直播间已经满了，没有空闲的直播间
			s.responseResult(w, 1, "Get Push Failed Live Room Full")
			log.Errorf("[Web Opera] handleGetPush Get Push Failed Live Room Full")
			return
		}

		errAlloc, liveRoomId = rtmpStream.AllocLiveRoomId(projectId)
		if errAlloc != nil {

			//分配直播间失败
			s.responseResult(w, 1, "Get Push Failed Alloc Live Room Failed")
			log.Errorf("[Web Opera] handleGetPush Get Push Failed Alloc Live Room Failed")
			return
		}

		newProject = true
	}

	//根据用户类型判断是否还有推流点
	if rtmpStream.PushUserFull(liveRoomId, configure.UserTypeEunm(userType), configure.VideoTypeEunm(videoType)) {

		//已经满了不能再进行分配了
		s.responseResult(w, 1, "Get Push Failed Live Room User Full Or userType & videoType Not Found")
		log.Errorf("[Web Opera] handleGetPush Get Push Failed Live Room User Full Or userType & videoType Not Found")
		return
	}

	//分配一个推流点
	videoTypeEunm := configure.VideoTypeEunm(videoType)
	err, pushId, pushBase, _, videoName := rtmpStream.GetPushId(liveRoomId, configure.UserTypeEunm(userType), videoTypeEunm)
	if err != nil {

		s.responseResult(w, 1, "Get Push Failed GetPushId Failed")
		log.Errorf("[Web Opera] handleGetPush Get Push Failed GetPushId Failed")
		return
	}
	pushIdString := strconv.Itoa(pushId)

	//给用户分配直播推流点
	listenString := strconv.Itoa(configure.RtmpServercfg.Listen)
	var pushurl string
	if !configure.RtmpServercfg.StaticAddr {

		//推流地址不固定
		date := time.Now().Format("20060102")
		pushurl = pushBase + ":" + listenString + "/live/" + liveRoomId + "/" + date + "/" + tmpProjectId[0] + "/" + videoName + "_" + pushIdString
	} else {

		//推流地址固定
		pushurl = pushBase + ":" + listenString + "/live/" + liveRoomId + "/" + tmpProjectId[0] + "/" + videoName + "_" + pushIdString
	}

	/*启动推流点*/

	//设置key值
	keyString := "push:" + liveRoomId + "/" + pushIdString + "/" + tmpProjectId[0]
	log.Infof("Server handleGetPush Create Key String %s Push Url %s", keyString, pushurl)

	//启动成功，进行设置
	rtmpStream.SetStartState(projectId, liveRoomId, pushId, pushurl, userType)
	ret = rtmpStream.CheckProjectExits(projectId)
	if !ret {

		s.responseResult(w, 1, "Stop Project Failed Not Found ProjectId")
		log.Errorf("[Web Opera] handleGetPush Stop Project Failed Not Found ProjectId")
		return
	}

	//设置此房间的混流推流地址可用
	// if newProject {
	// 	mixUrl := pushBase + ":" + listenString + "/live/" + liveRoomId + "/" + tmpProjectId[0] + "/mixStream"
	// 	rtmpStream.SetMixState(projectId, liveRoomId, mixUrl, true)
	// }

	//返回结构
	var response configure.PushResponse
	response.LiveRoomId = liveRoomId
	response.PushId = pushId
	response.PushUrl = pushurl
	response.UserType = userType
	response.ProjectId = projectId

	//将结构转换成json格式
	jsonData, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)

	log.Infof("[Web Opera] handleGetPush Return %s", string(jsonData))

	//判断是否是分配的新项目
	if newProject && configure.VideoTypeEunm(videoType) != configure.Camera {

		err, newProjectLiveRoom := rtmpStream.GetLiveRoomFromRoomId(liveRoomId)
		if err == nil {

			for _, v := range newProjectLiveRoom.Urls {

				if v.VideoType == configure.Camera {

					tmp := pushBase + ":" + listenString + "/live/" + liveRoomId + "/" + tmpProjectId[0] + "/Camera_" + strconv.Itoa(v.PushId)
					if pushurl != tmp {

						//打开推流地址
						rtmpStream.SetStartState(projectId, liveRoomId, v.PushId, tmp, int(v.UserType))
					}

					requestUrl := v.RequestUrl + "?&roomId=" + liveRoomId + "&pushUrl=" + tmp
					go s.requestUrl(requestUrl, configure.ProjectStart)
				}
			}
		}
	}
}

//断开指定的发布者链接
func (s *Server) closeReaderConn(w http.ResponseWriter, url string) {

	//得到Rtmp流的管理对象closeReaderConn
	rtmpStream := s.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {
		s.responseResult(w, 1, "Get rtmp Stream information error")
		return
	}

	//计算发布者相关信息
	for item := range rtmpStream.GetStreams().IterBuffered() {

		if s, ok := item.Val.(*rtmp.Stream); ok {
			if s.GetReader() != nil {
				switch s.GetReader().(type) {
				case *rtmp.VirReader:
					v := s.GetReader().(*rtmp.VirReader)
					if v.Info().URL == url {
						v.Close(errors.New("Force Close Publisher Conn"))
					}
				}
			}
		}
	}

}

//断开指定的观看者链接
func (s *Server) closeWriteConn(w http.ResponseWriter, url string) {

	//得到Rtmp流的管理对象closeWriteConn
	rtmpStream := s.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {
		s.responseResult(w, 1, "Get rtmp Stream information error")
		return
	}

	//统计观看者相关信息
	for item := range rtmpStream.GetStreams().IterBuffered() {

		ws := item.Val.(*rtmp.Stream).GetWs()

		for s := range ws.IterBuffered() {
			if pw, ok := s.Val.(*rtmp.PackWriterCloser); ok {
				if pw.GetWriter() != nil {
					switch pw.GetWriter().(type) {
					case *rtmp.VirWriter:

						v := pw.GetWriter().(*rtmp.VirWriter)
						if v.Info().URL == url {
							v.Close(errors.New("Force Close Viewers Conn"))
						}
					}
				}
			}
		}
	}
}

/*
项目停止

格式：
http://127.0.0.1:8090/stopProject

参数：
oper：操作类型（start、stop）
app：app类型（live）
projectId: 项目ID
userType: 用户类型

地址举例：
http://127.0.0.1:8090/stopProject?&projectId=12
*/
func (s *Server) handleStopProject(w http.ResponseWriter, req *http.Request) {

	req.ParseForm()

	//获得参数信息
	tmpProjectId := req.Form["projectId"]
	projectId, errInt := strconv.Atoi(tmpProjectId[0])
	if errInt != nil {

		errString := fmt.Sprintf("handleStopProject projectId Param error, please check them projectId=%s err=%s", tmpProjectId[0], errInt.Error())
		s.responseResult(w, 1, errString)
		log.Errorf("[Web Opera] handleStopProject projectId Param error, please check them")
		return
	}
	log.Infof("[Web Opera] handleStopProject projectId=%d", projectId)

	//得到Rtmp流的管理对象
	rtmpStream := s.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {

		s.responseResult(w, 1, "Get rtmp Stream information error")
		log.Errorf("[Web Opera] handleStopProject Get rtmp Stream information error")
		return
	}

	//判断项目是否存在
	ret := rtmpStream.CheckProjectExits(projectId)
	if !ret {

		s.responseResult(w, 1, "Stop Project Failed Not Found ProjectId")
		log.Errorf("[Web Opera] handleStopProject Stop Project Failed CheckProjectExits Not Found ProjectId")
		return
	}

	//得到项目对应的房间
	errRoom, liveRoom := rtmpStream.GetLiveRoom(projectId)
	if errRoom != nil {

		s.responseResult(w, 1, "Stop Project Failed Not Found ProjectId")
		log.Errorf("[Web Opera] handleStopProject Stop Project Failed GetLiveRoom Not Found ProjectId")
		return
	}

	//推送项目结束通知
	for _, v := range liveRoom.Urls {

		if v.VideoType == configure.Camera {

			requestUrl := v.RequestUrl + "?roomId=" + liveRoom.LiveRoomId + "&pushId=" + strconv.Itoa(v.PushId)
			s.requestUrl(requestUrl, configure.ProjectStop)
		}
	}

	//轮询所有推流点进行终止推流
	log.Infof("[Web Opera] handleStopProject Stop LiveRoomId=%s ProjectId=%d", liveRoom.LiveRoomId, liveRoom.ProjectId)
	for _, v := range liveRoom.Urls {

		if v.State == 1 {

			//关闭指定连接的发布者和观看者
			s.closeReaderConn(w, v.PushUrl)
			s.closeWriteConn(w, v.PushUrl)

			//删除信息
			v.LimitAudio = false
			v.State = 0
			v.PushUrl = ""
		}
	}
	_, pushBase, listenString := rtmpStream.GetHolderUrlBase(liveRoom.LiveRoomId)
	mixUrl := pushBase + ":" + listenString + "/live/" + liveRoom.LiveRoomId + "/" + tmpProjectId[0] + "/mixStream"
	s.closeReaderConn(w, mixUrl)
	s.closeWriteConn(w, mixUrl)
	rtmpStream.SetMixState(liveRoom.ProjectId, liveRoom.LiveRoomId, mixUrl, false, "")

	liveRoom.ProjectId = -1
	s.responseResult(w, 0, "Stop Project Success")

	log.Infof("[Web Opera] handleStopProject Success ProjectId=%d", projectId)
}

/*
结束推流点

格式：
http://127.0.0.1:8090/stopPush

参数：
projectId: 项目ID
pushId: 推流点ID

地址举例：
http://127.0.0.1:8090/stopPush?&projectId=12&pushId=1
*/
func (s *Server) handleStopPush(w http.ResponseWriter, req *http.Request) {

	req.ParseForm()

	//获得参数信息
	tmpProjectId := req.Form["projectId"]
	projectId, errInt := strconv.Atoi(tmpProjectId[0])
	if errInt != nil {

		errString := fmt.Sprintf("handleStopPush projectId Param error, please check them projectId=%s err=%s", tmpProjectId[0], errInt.Error())
		s.responseResult(w, 1, errString)
		log.Errorf("[Web Opera] handleStopPush projectId Param error, please check them")
		return
	}
	log.Infof("[Web Opera] handleStopPush projectId=%d", projectId)

	//获得参数信息
	tmpPushId := req.Form["pushId"]
	pushId, err := strconv.Atoi(tmpPushId[0])
	if err != nil {

		errString := fmt.Sprintf("handleStopPush pushId Param error, please check them pushId=%s err=%s", tmpPushId[0], err.Error())
		s.responseResult(w, 1, errString)
		log.Errorf("[Web Opera] handleStopPush pushId Param error, please check them")
		return
	}
	log.Infof("[Web Opera] handleStopPush projectId=%d pushId=%d", projectId, pushId)

	//得到Rtmp流的管理对象
	rtmpStream := s.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {

		s.responseResult(w, 1, "Get rtmp Stream information error")
		log.Errorf("[Web Opera] handleStopPush Get rtmp Stream information error")
		return
	}

	//判断项目是否存在
	ret := rtmpStream.CheckProjectExits(projectId)
	if !ret {

		s.responseResult(w, 1, "Stop Push Failed Not Found ProjectId")
		log.Errorf("[Web Opera] handleStopPush Stop Push Failed CheckProjectExits Not Found ProjectId")
		return
	}

	//得到项目对应的房间
	errRoom, liveRoom := rtmpStream.GetLiveRoom(projectId)
	if errRoom != nil {

		s.responseResult(w, 1, "Stop Push Failed Not Found ProjectId")
		log.Errorf("[Web Opera] handleStopPush Stop Push Failed GetLiveRoom Not Found ProjectId")
		return
	}

	// //推送项目结束通知
	// for _, v := range liveRoom.Urls {

	// 	if v.VideoType == configure.Camera {

	// 		if v.PushId == pushId {

	// 			requestUrl := v.RequestUrl + "?roomId=" + liveRoom.LiveRoomId + "&pushId=" + strconv.Itoa(v.PushId)
	// 			s.requestUrl(requestUrl, configure.ProjectStop)
	// 		}
	// 	}
	// }

	//轮询所有推流点进行终止推流
	log.Infof("[Web Opera] handleStopPush Stop LiveRoomId=%s ProjectId=%d", liveRoom.LiveRoomId, liveRoom.ProjectId)
	for _, v := range liveRoom.Urls {

		if (v.State == 1) && (v.PushId == pushId) {

			//关闭指定连接的发布者和观看者
			s.closeReaderConn(w, v.PushUrl)
			s.closeWriteConn(w, v.PushUrl)

			//删除信息
			v.LimitAudio = false
			v.State = 0
			v.PushUrl = ""
		}
	}
	s.responseResult(w, 0, "Stop Push Success")

	log.Infof("[Web Opera] handleStopPush Success ProjectId=%d PushId=%d", projectId, pushId)
}

/*
得到当前列表

格式：
http://127.0.0.1:8090/getCurrentList?&projectId=12

地址举例：
http://127.0.0.1:8090/getCurrentList?&projectId=12
*/
func (s *Server) handleGetCurrentList(w http.ResponseWriter, req *http.Request) {

	//得到Rtmp流的管理对象
	rtmpStream := s.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {

		s.responseResult(w, 1, "Get rtmp Stream information error")
		return
	}

	req.ParseForm()

	//获得参数信息
	tmpProjectId := req.Form["projectId"]
	if tmpProjectId == nil {

		err, liveRooms := rtmpStream.GetRtmpList()
		if err != nil {

			s.responseResult(w, 1, "Get Rtmp List Failed ")
			return
		}

		jsonData, _ := json.Marshal(liveRooms)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
		log.Infof("Server handleGetCurrentList Return %s", string(jsonData))

	} else {

		projectId, errInt := strconv.Atoi(tmpProjectId[0])
		if errInt != nil {

			errString := fmt.Sprintf("handleGetCurrentList projectId Param error, please check them projectId=%s err=%s", tmpProjectId[0], errInt.Error())
			s.responseResult(w, 1, errString)
			return
		}
		log.Infof("Server handleGetCurrentList projectId=%d", projectId)

		err, liveRooms := rtmpStream.GetSingleRtmpList(projectId)
		if err != nil {

			s.responseResult(w, 1, "Get Rtmp List Failed ")
			return
		}

		jsonData, _ := json.Marshal(liveRooms)
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
		log.Infof("Server handleGetCurrentList Return %s", string(jsonData))
	}
}

/*
得到项目回看列表

格式：
http://127.0.0.1:8090/getReplay?&projectId=12

地址举例：
http://127.0.0.1:8090/getReplay?&projectId=12
*/
func (s *Server) handleGetReplay(w http.ResponseWriter, req *http.Request) {

	//得到Rtmp流的管理对象
	rtmpStream := s.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {

		s.responseResult(w, 1, "Get rtmp Stream information error")
		return
	}
	req.ParseForm()

	//获得参数信息
	tmpProjectId := req.Form["projectId"]
	projectId, errInt := strconv.Atoi(tmpProjectId[0])
	if errInt != nil {

		errString := fmt.Sprintf("handleGetReplay projectId Param error, please check them projectId=%s err=%s", tmpProjectId[0], errInt.Error())
		s.responseResult(w, 1, errString)
		return
	}
	log.Infof("Server handleGetReplay projectId=%d", projectId)

	//这里需要轮训所有的推流点所对应的的目录中是否存在此项目编号
	replayRooms := rtmpStream.GetProjectReplayList(projectId)

	jsonData, _ := json.Marshal(replayRooms)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)

	log.Infof("Server handleGetReplay Return %s", string(jsonData))

}

/*
声音控制

格式：
http://127.0.0.1:8090/setPushIdAudio?&projectId=12&pushId=1&audio=0

地址举例：
http://127.0.0.1:8090/setPushIdAudio?&projectId=12&pushId=1&audio=0
*/
func (s *Server) handleSetAudioFromPushId(w http.ResponseWriter, req *http.Request) {

	req.ParseForm()

	//获得参数信息
	tmpProjectId := req.Form["projectId"]
	projectId, errInt := strconv.Atoi(tmpProjectId[0])
	if errInt != nil {

		errString := fmt.Sprintf("handleSetAudioFromPushId projectId Param error, please check them projectId=%s err=%s", tmpProjectId[0], errInt.Error())
		s.responseResult(w, 1, errString)
		return
	}

	tmpPushId := req.Form["pushId"]
	pushId, errPushId := strconv.Atoi(tmpPushId[0])
	if errPushId != nil {

		s.responseResult(w, 1, "pushId Param error, please check them")
		return
	}

	tmpAudio := req.Form["audio"]
	audio, errAudio := strconv.Atoi(tmpAudio[0])
	if errAudio != nil {

		s.responseResult(w, 1, "audio Param error, please check them")
		return
	}
	log.Infof("Server handleSetAudioFromPushId projectId=%d pushId=%d audio=%d", projectId, pushId, audio)

	//得到Rtmp流的管理对象
	rtmpStream := s.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {

		s.responseResult(w, 1, "Get rtmp Stream information error")
		return
	}

	//设置声音控制
	rtmpStream.SetLimitAudioFromPushId(projectId, pushId, audio)

	writeString := fmt.Sprintf("set Audio Success ProjectId:%d pushId=%d Audio=%d", projectId, pushId, audio)
	s.responseResult(w, 0, writeString)

	log.Infof("Server handleSetAudioFromPushId %s", writeString)

	//混音推流过程
	if audio == 1 {

		/*启动混音转推功能*/

		//得到主持人的URL
		_, mainUrl := rtmpStream.GetHolderUrl(projectId)

		//得到此用户的URL
		_, subUrl := rtmpStream.GetPushIdUrl(projectId, pushId)

		//得到此房间混音地址
		_, mixUrl := rtmpStream.GetMixUrl(projectId)

		//开始合并并推流
		go rtmpStream.Mixffmpeg(projectId, mainUrl, subUrl, mixUrl)

		log.Infof("---->>>>handleSetAudioFromPushId Start projectId=%d mainUrl=%s subUrl=%s MixFFmpeg=%s", projectId, mainUrl, subUrl, mixUrl)

	} else if audio == 0 {

		/*停止混音转推功能*/

		//得到此房间混音地址
		_, mixUrl := rtmpStream.GetMixUrl(projectId)

		//停止混音推流
		go rtmpStream.StopMixffmpeg(projectId, mixUrl)

		log.Infof("---->>>>handleSetAudioFromPushId Stop projectId=%d MixFFmpeg=%s", projectId, mixUrl)
	}
}

func (s *Server) requestUrl(Url string, requestType configure.RequestTypeEunm) bool {

	var requestString string
	switch requestType {

	case configure.ProjectStart:
		requestString = Url + "&status=1"

	case configure.ProjectStop:
		requestString = Url + "&status=1"
	}

	resp, err := http.Get(requestString)
	if err != nil {

		log.Error("requestUrl Failed %s", requestString)
		return false
	}

	defer resp.Body.Close()
	//	_, errResp := ioutil.ReadAll(resp.Body)
	if resp.StatusCode == 200 {

		log.Infof("requestUrl Success %s", requestString)
		return true
	}
	log.Infof("requestUrl Failed %s", requestString)
	return false
}

func NewReportStat(serverlist []string, localserver *Server) *ReportStat {
	return &ReportStat{
		serverList:  serverlist,
		isStart:     false,
		localServer: localserver,
	}
}

func (self *ReportStat) httpsend(data []byte) error {

	return nil
}

func (self *ReportStat) onWork() {
	rtmpStream := self.localServer.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {
		return
	}
	for {
		if !self.isStart {
			break
		}

		if self.serverList == nil || len(self.serverList) == 0 {
			log.Warning("Report statics server list is null.")
			break
		}

		msgs := new(Streams)
		msgs.PublisherNumber = 0
		msgs.PlayerNumber = 0

		for item := range rtmpStream.GetStreams().IterBuffered() {
			if s, ok := item.Val.(*rtmp.Stream); ok {
				if s.GetReader() != nil {
					switch s.GetReader().(type) {
					case *rtmp.VirReader:
						v := s.GetReader().(*rtmp.VirReader)
						msg := Stream{item.Key, v.Info().URL, v.ReadBWInfo.PeerIP, v.ReadBWInfo.StreamId, v.ReadBWInfo.VideoDatainBytes, v.ReadBWInfo.VideoSpeedInBytesperMS,
							v.ReadBWInfo.AudioDatainBytes, v.ReadBWInfo.AudioSpeedInBytesperMS}
						msgs.Publishers = append(msgs.Publishers, msg)
						msgs.PublisherNumber++
					}
				}
			}
		}

		for item := range rtmpStream.GetStreams().IterBuffered() {
			ws := item.Val.(*rtmp.Stream).GetWs()
			for s := range ws.IterBuffered() {
				if pw, ok := s.Val.(*rtmp.PackWriterCloser); ok {
					if pw.GetWriter() != nil {
						switch pw.GetWriter().(type) {
						case *rtmp.VirWriter:
							v := pw.GetWriter().(*rtmp.VirWriter)
							msg := Stream{item.Key, v.Info().URL, v.WriteBWInfo.PeerIP, v.WriteBWInfo.StreamId, v.WriteBWInfo.VideoDatainBytes, v.WriteBWInfo.VideoSpeedInBytesperMS,
								v.WriteBWInfo.AudioDatainBytes, v.WriteBWInfo.AudioSpeedInBytesperMS}
							msgs.Players = append(msgs.Players, msg)
							msgs.PlayerNumber++
						}
					}
				}
			}
		}
		resp, _ := json.Marshal(msgs)

		//log.Info("report statics server list:", self.serverList)
		//log.Info("resp:", string(resp))

		self.httpsend(resp)
		time.Sleep(time.Second * 5)
	}
}

func (self *ReportStat) Start() error {
	if self.isStart {
		return errors.New("Report Statics has already started.")
	}

	self.isStart = true

	go self.onWork()
	return nil
}

func (self *ReportStat) Stop() {
	if !self.isStart {
		return
	}

	self.isStart = false
}
