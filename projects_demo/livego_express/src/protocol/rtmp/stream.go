package rtmp

import (
	"av"
	cmap "concurrent-map"
	"configure"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	log "logging"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"protocol/rtmp/cache"
	"reflect"
	"strconv"
	"strings"
	_ "syscall"
	"time"
)

var (
	EmptyID = ""
)

type LiveRoomMix struct {
	LiveRoomId  string
	ProjectId   int
	MixSavePath string
	MixSaveUrl  string
	MixRtmpBase string
	MixUrl      string
	cmdExec     *exec.Cmd
}

type LiveRoomsMix struct {
	Rooms []*LiveRoomMix
}

type RtmpStream struct {
	streams   cmap.ConcurrentMap  //流管理（包括发布者和观看者）
	liveRooms configure.LiveRooms //直播房间管理
	mixRooms  LiveRoomsMix
	md5s      configure.Md5s
}

func videoName(videoType configure.VideoTypeEunm) (error, string) {

	switch videoType {

	case configure.Camera:
		return nil, "Camera"
	case configure.PCCamera:
		return nil, "PCCamera"
	case configure.DesktopShare:
		return nil, "DesktopShare"
	}
	return errors.New("Not Found VideoTypeEunm"), ""
}

func NewRtmpStream() *RtmpStream {

	ret := &RtmpStream{
		streams: cmap.New(),
	}

	ret.initLiveRooms()
	ret.loadReplayConfig()
	go ret.checkPublisher()
	go ret.checkMediaFile()

	return ret
}

func (rs *RtmpStream) loadReplayConfig() error {

	//读取配置文件
	log.Infof("---->>>> Start Load Replay Configure File")
	data, err := ioutil.ReadFile("replay.json")
	if err != nil {

		log.Errorf("loadReplayConfig error=%v", err)
		return err
	}
	log.Infof("---->>>> Load Replay Configure Data: \r\n%s", string(data))

	//读取Json配置
	log.Infof("---->>>> Load Replay Configure Unmarshal")
	err = json.Unmarshal(data, &rs.md5s)
	if err != nil {
		log.Errorf("---->>>> Load Replay Configure Unmarshal error:%v", err)
		return err
	}
	log.Infof("---->>>> Load Replay Configure Json data:%v", rs.md5s)

	return nil
}

func (rs *RtmpStream) writeReplayConfig() error {

	//读取配置文件
	log.Infof("---->>>> Start Write Replay Configure File")

	data, err := json.Marshal(rs.md5s)
	if err != nil {
		log.Errorf("writeReplayConfig Marshal error=%v", err)
		return err
	}

	err = ioutil.WriteFile("replay.json", data, os.ModeAppend)
	if err != nil {

		log.Errorf("writeReplayConfig WriteFile error=%v", err)
		return err
	}

	log.Infof("---->>>> Write Replay Configure Data: \r\n%s", string(data))
	return nil
}

func (rs *RtmpStream) pathExists(path string) (bool, error) {

	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (rs *RtmpStream) getFilePath(url string) (error, string) {

	//判断是否是混音流
	exist := strings.Contains(url, "mixStream")
	if !exist {

		/*不是混音流*/

		//获取此pushID所对应文件保存地址
		err, pushUrl, liveRoomId, projectId := rs.GetPushFromUrl(url)
		if err != nil {
			return err, ""
		}

		//得到数据目录
		basePath := pushUrl.SavePath

		//拼接目录
		outPath := fmt.Sprintf("%s/%s/%d", basePath, liveRoomId, projectId)

		//判断文件是否存在
		exist, err := rs.pathExists(outPath)
		if err != nil {

			log.Errorf("Check Path Failed! [%v]\n", err)
			return errors.New("Check Path Failed"), ""
		}

		//目录不存在则创建目录
		if !exist {

			// 创建文件夹
			err := os.MkdirAll(outPath, os.ModePerm)
			if err != nil {

				log.Errorf("MkAll Path Failed [%v]\n", err)
				return errors.New("MkAll Path Failed"), ""
			}
		}

		_, curVideoName := videoName(pushUrl.VideoType)

		//创建FFMpeg保存的文件名称
		timeNow := time.Now()
		timeString := timeNow.Format("20060102T150405")
		outFile := fmt.Sprintf("%s/%s_%s.ts", outPath, curVideoName, timeString)

		for {

			fileExists, _ := rs.pathExists(outFile)
			if !fileExists {

				return nil, outFile
			} else {

				time.Sleep(time.Duration(1) * time.Second)
				outFile = fmt.Sprintf("%s/%s_%s.ts", outPath, curVideoName, time.Now().Format("20060102T150405"))
			}
		}
		return errors.New("Get FFMpeg File Failed"), ""
	} else {

		/*是混音流*/

		//获取此pushID所对应文件保存地址
		err, liveRoomId, projectId, mixSaveBase := rs.GetMixFromUrl(url)
		if err != nil {
			return err, ""
		}

		//拼接目录
		outPath := fmt.Sprintf("%s/%s/%d", mixSaveBase, liveRoomId, projectId)

		//判断文件是否存在
		exist, err := rs.pathExists(outPath)
		if err != nil {

			log.Errorf("Check Path Failed! [%v]\n", err)
			return errors.New("Check Path Failed"), ""
		}

		//目录不存在则创建目录
		if !exist {

			// 创建文件夹
			err := os.MkdirAll(outPath, os.ModePerm)
			if err != nil {

				log.Errorf("MkAll Path Failed [%v]\n", err)
				return errors.New("MkAll Path Failed"), ""
			}
		}

		curVideoName := "mixStream"

		//创建FFMpeg保存的文件名称
		timeNow := time.Now()
		timeString := timeNow.Format("20060102T150405")
		outFile := fmt.Sprintf("%s/%s_%s.ts", outPath, curVideoName, timeString)

		for {

			fileExists, _ := rs.pathExists(outFile)
			if !fileExists {

				return nil, outFile
			} else {

				time.Sleep(time.Duration(1) * time.Second)
				outFile = fmt.Sprintf("%s/%s_%s.ts", outPath, curVideoName, time.Now().Format("20060102T150405"))
			}
		}
		return errors.New("Get FFMpeg File Failed"), ""
	}
}

func (rs *RtmpStream) StreamExist(key string) bool {

	log.Infof("RtmpStream StreamExist %s", key)

	if i, ok := rs.streams.Get(key); ok {

		if ns, stream_ok := i.(*Stream); stream_ok {

			log.Errorf("RtmpStream Stream Exist %s", ns.info.String())
			return true
		}
	}

	return false
}

func (rs *RtmpStream) isCamera(uri string) bool {

	log.Infof("RtmpStream isCamera %s", uri)

	u, err := url.Parse(uri)
	if err != nil {
		return false
	}

	m, queryErr := url.ParseQuery(u.RawQuery)
	if queryErr != nil {
		return false
	}

	if len(m) <= 0 {
		return false
	}

	md5String := m["v"][0]

	beginPos := strings.Index(uri, "/live")
	endPos := strings.Index(uri, "?v")

	pathString := uri[beginPos:endPos]
	if pathString != "" && md5String != "" {
		return true
	}
	return false
}

//发布者
func (rs *RtmpStream) HandleReader(r av.ReadCloser) {

	info := r.Info()
	log.Infof("RtmpStream HandleReader %s", info.String())

	var stream *Stream
	i, ok := rs.streams.Get(info.Key)
	if stream, ok = i.(*Stream); ok {

		//发布者地址有流再次推送过来
		log.Infof("RtmpStream HandleReader TransStop Old Stream")
		stream.TransStop()
		id := stream.ID()

		if id != EmptyID && id != info.UID {

			exist := strings.Contains(info.URL, "mixStream")
			if !exist {

			}

			log.Infof("RtmpStream HandleReader NewStream")

			//判断地址是否是球机的地址
			if !rs.isCamera(info.URL) {

				//判断此发布者是否具有语音的功能
				err, pushStreamUrl := rs.FindPushStream(info.URL)
				if err != nil {
					log.Error("RtmpStream HandleReader Not Found Url=%s", info.URL)
					return
				}

				if pushStreamUrl.UserType == configure.Holder {
					pushStreamUrl.LimitAudio = false
				} else {
					pushStreamUrl.LimitAudio = true
				}

				ns := NewStream(rs, pushStreamUrl.LimitAudio)
				stream.Copy(ns)
				stream = ns
				rs.streams.Set(info.Key, ns)

			} else {
				ns := NewStream(rs, false)
				stream.Copy(ns)
				stream = ns
				rs.streams.Set(info.Key, ns)
			}
		}

		//判断是否启动FFmpeg
		if stream.cmdExec != nil {
			log.Infof("RtmpStream HandleReader Repeat come into HandleReader repeat process = %p", stream.cmdExec.Process)
		}

		if "enable" == configure.GetEngineEnable() {

			log.Infof("RtmpStream HandleReader configure.GetEngineEnable() = enable Startffmpeg %s", info.URL)

			err, currFile := rs.getFilePath(info.URL)
			if err == nil {
				log.Infof("RtmpStream HandleReader Start FFMpeg URL=%s File=%s", info.URL, currFile)
				go stream.Startffmpeg(info.URL, currFile)
			} else {
				log.Error("RtmpStream HandleReader Start FFMpeg Failed URL=%s", info.URL)
			}

		}

	} else {

		//创建发布者
		log.Infof("RtmpStream HandleReader NewStream Key=%s", info.Key)

		exist := strings.Contains(info.URL, "mixStream")
		if !exist {

			/*正常流*/

			//判断地址是否是球机的地址
			if !rs.isCamera(info.URL) {

				//判断此发布者是否具有语音的功能
				err, pushStreamUrl := rs.FindPushStream(info.URL)
				if err != nil {

					log.Error("RtmpStream HandleReader Not Found Url=%s", info.URL)
					return
				}

				//根据用户类型设置是否进行音频限制
				if pushStreamUrl.UserType == configure.Holder {
					pushStreamUrl.LimitAudio = false
				} else {
					pushStreamUrl.LimitAudio = true
				}
				stream = NewStream(rs, pushStreamUrl.LimitAudio)
				rs.streams.Set(info.Key, stream)
				stream.info = info

				if stream.cmdExec != nil {
					log.Infof("RtmpStream HandleReader first come into HandleReader first process = %p ", stream.cmdExec.Process)
				}

				if "enable" == configure.GetEngineEnable() {

					err, currFile := rs.getFilePath(info.URL)
					if err == nil {
						log.Infof("RtmpStream HandleReader Start FFMpeg URL=%s File=%s", info.URL, currFile)
						go stream.Startffmpeg(info.URL, currFile)
					} else {
						log.Error("RtmpStream HandleReader Start FFMpeg Failed URL=%s", info.URL)
					}
				}

				//根据Url地址得到pushId
				err, liveRoomId, _ := rs.GetPushIdFromUrl(info.URL)
				if err != nil {
					return
				}
				stream.AddReader(r, liveRoomId)
			} else {

				stream = NewStream(rs, false)
				rs.streams.Set(info.Key, stream)
				stream.info = info

				if stream.cmdExec != nil {
					log.Infof("RtmpStream HandleReader first come into HandleReader first process = %p ", stream.cmdExec.Process)
				}

				if "enable" == configure.GetEngineEnable() {

					err, currFile := rs.getFilePath(info.URL)
					if err == nil {
						log.Infof("RtmpStream HandleReader Start FFMpeg URL=%s File=%s", info.URL, currFile)
						go stream.Startffmpeg(info.URL, currFile)
					} else {
						log.Error("RtmpStream HandleReader Start FFMpeg Failed URL=%s", info.URL)
					}
				}

				liveRoomId := "99999"
				stream.AddReader(r, liveRoomId)
			}

		} else {

			/*混音流*/

			limitAudio := false
			stream = NewStream(rs, limitAudio)
			rs.streams.Set(info.Key, stream)
			stream.info = info

			if stream.cmdExec != nil {
				log.Infof("RtmpStream HandleReader first come into HandleReader first process = %p ", stream.cmdExec.Process)
			}

			if "enable" == configure.GetEngineEnable() {

				err, currFile := rs.getFilePath(info.URL)
				if err == nil {
					log.Infof("RtmpStream HandleReader Start FFMpeg URL=%s File=%s", info.URL, currFile)
					go stream.Startffmpeg(info.URL, currFile)
				} else {
					log.Error("RtmpStream HandleReader Start FFMpeg Failed URL=%s", info.URL)
				}
			}

			err, liveRoomId, _, _ := rs.GetMixFromUrl(info.URL)
			if err != nil {
				return
			}
			stream.AddReader(r, liveRoomId)
		}
	}
}

//观看者
func (rs *RtmpStream) HandleWriter(w av.WriteCloser) {

	info := w.Info()
	log.Infof("RtmpStream HandleWriter info %s, type %v", info.String(), reflect.TypeOf(w))

	var s *Stream
	ok := rs.streams.Has(info.Key)
	if !ok {

		log.Infof("RtmpStream NewStream %s", info.Key)
		s = NewStream(rs, true)
		rs.streams.Set(info.Key, s)
		s.info = info
	} else {

		log.Infof("RtmpStream HandleWriter Get %s", info.Key)
		item, ok := rs.streams.Get(info.Key)
		if ok {
			s = item.(*Stream)
			s.AddWriter(w)
		}
	}
}

func (rs *RtmpStream) GetStreams() cmap.ConcurrentMap {

	return rs.streams
}

func (rs *RtmpStream) CheckProjectExits(projectId int) bool {

	log.Infof("RtmpStream CheckProjectExits projectId=%d", projectId)

	for _, v := range rs.liveRooms.Rooms {

		log.Infof("RtmpStream CheckProjectExits get projectId=%d", v.ProjectId)
		if v.ProjectId == projectId {
			return true
		}
	}

	return false
}

func (rs *RtmpStream) GetLiveRoom(projectId int) (error, *configure.LiveRoom) {

	log.Infof("RtmpStream GetLiveRoom projectId=%d", projectId)
	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == projectId {
			return nil, v
		}
	}
	return errors.New("Not Found Live Room"), nil
}

func (rs *RtmpStream) GetLiveRoomFromRoomId(liveRoomId string) (error, *configure.LiveRoom) {

	log.Infof("RtmpStream GetLiveRoomFromRoomId liveRoomId=%s", liveRoomId)
	for _, v := range rs.liveRooms.Rooms {

		if v.LiveRoomId == liveRoomId {
			return nil, v
		}
	}
	return errors.New("Not Found Live Room"), nil
}

func (rs *RtmpStream) LiveRoomFull() bool {

	log.Infof("RtmpStream Check LiveRoomFull")

	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == -1 {
			return false
		}
	}

	return true
}

func (rs *RtmpStream) AllocLiveRoomId(projectId int) (error, string) {

	log.Infof("RtmpStream AllocLiveRoomId=%d", projectId)

	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == -1 {
			return nil, v.LiveRoomId
		}
	}

	return errors.New("Live Room Full"), ""
}

func (rs *RtmpStream) PushUserFull(liveRoomId string, userType configure.UserTypeEunm, videoType configure.VideoTypeEunm) bool {

	log.Infof("RtmpStream PushUserFull liveRoomId=%s userType=%d videoType=%d", liveRoomId, userType, videoType)

	for _, v := range rs.liveRooms.Rooms {

		if v.LiveRoomId == liveRoomId {

			for _, value := range v.Urls {

				if value.State == 0 && value.UserType == userType && videoType == value.VideoType {
					return false
				}
			}
		}
	}
	return true
}

func (rs *RtmpStream) GetPushId(liveRoomId string, userType configure.UserTypeEunm, videoType configure.VideoTypeEunm) (error, int, string, configure.VideoTypeEunm, string) {

	log.Infof("RtmpStream GetPushId liveRoomId=%s userType=%d videoType=%d", liveRoomId, userType, videoType)

	for _, v := range rs.liveRooms.Rooms {

		if v.LiveRoomId == liveRoomId {

			for _, value := range v.Urls {

				if value.State == 0 && value.UserType == userType && value.VideoType == videoType {

					log.Infof("Find Can Use Push Id liveRoomId=%s UserType=%d VideoType=%d PushId=%d ", liveRoomId, userType, videoType, value.PushId)
					_, curVideoName := videoName(value.VideoType)

					return nil, value.PushId, value.RtmpBase, value.VideoType, curVideoName
				}

			}
		}
	}
	return errors.New("Not Found PushId"), -1, "", configure.Camera, ""
}

func (rs *RtmpStream) GetHolderUrl(projectId int) (error, string) {

	log.Infof("RtmpStream GetHolderUrl projectId=%d", projectId)

	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == projectId {

			for _, value := range v.Urls {

				if value.State == 1 && value.UserType == configure.Holder && (value.VideoType == configure.DesktopShare) {

					return nil, value.PushUrl
				}

			}
		}
	}
	return errors.New("Not Found Holder Url"), ""
}

func (rs *RtmpStream) FindPushStream(pushUrl string) (error, *configure.PushStreamUrl) {

	log.Infof("RtmpStream FindPushStream pushUrl=%s", pushUrl)

	//判断是否是正常的流
	for _, v := range rs.liveRooms.Rooms {

		for _, value := range v.Urls {

			if value.PushUrl == pushUrl && 1 == value.State {

				return nil, value
			}
		}
	}
	return errors.New("Not Found PushUrl"), nil
}

func (rs *RtmpStream) SetPushPushing(pushUrl string, pushing bool) {

	log.Info("RtmpStream SetPushPushing", "pushUrl=", pushUrl, "pushing=", pushing)

	err, pushStream := rs.FindPushStream(pushUrl)
	if err != nil {
		return
	}
	if pushing {
		pushStream.Pushing = 1
	} else {
		pushStream.Pushing = 0
	}
}

func (rs *RtmpStream) FindMixStream(pushUrl string) (error, *configure.LiveRoom) {

	log.Infof("RtmpStream FindMixStream pushUrl=%s", pushUrl)

	for _, v := range rs.liveRooms.Rooms {

		log.Infof("RtmpStream FindMixStream pushUrl=%s MixUrl=%s LiveRoomId=%s", pushUrl, v.MixUrl, v.LiveRoomId)

		if v.MixUrl == pushUrl {

			return nil, v
		}
	}
	return errors.New("Not Found PushUrl"), nil
}

func (rs *RtmpStream) SetLimitAudioFromPushId(projectId int, pushId int, limit int) error {

	log.Infof("RtmpStream SetLimitAudioFromPushId liveRoomId=%d pushId=%d limit=%d", projectId, pushId, limit)

	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == projectId {

			for _, value := range v.Urls {

				if value.PushId == pushId && 1 == value.State {

					value.LimitAudio = !(limit != 0)
					log.Infof("RtmpStream SetLimitAudioFromPushId liveRoomId=%d pushId=%d value.LimitAudio=%d", projectId, pushId, value.LimitAudio)
					return nil
				}
			}
		}
	}
	return errors.New("Not Found SetLimitAudio")
}

func (rs *RtmpStream) GetLimitAudioFromPushId(projectId int, pushId int) (error, bool) {

	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == projectId {

			for _, value := range v.Urls {

				if value.PushId == pushId {

					return nil, value.LimitAudio
				}
			}
		}
	}
	return errors.New("Not Found PushId"), false
}

func (rs *RtmpStream) GetPushIdUrl(projectId int, pushId int) (error, string) {

	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == projectId {

			for _, value := range v.Urls {

				if value.PushId == pushId {

					return nil, value.PushUrl
				}
			}
		}
	}
	return errors.New("Not Found PushId"), ""
}

func (rs *RtmpStream) GetMixUrl(projectId int) (error, string) {

	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == projectId {

			listenString := strconv.Itoa(configure.RtmpServercfg.Listen)
			projectIdString := strconv.Itoa(projectId)
			mixUrl := v.MixRtmpBase + ":" + listenString + "/live/" + v.LiveRoomId + "/" + projectIdString + "/mixStream"
			return nil, mixUrl
		}
	}
	return errors.New("Not Found ProjectId"), ""
}

func (rs *RtmpStream) SetLimitAudioFromUrl(Url string, limit bool) error {

	log.Infof("RtmpStream SetLimitAudioFromUrl Url=%s limit=%d", Url, limit)

	for _, v := range rs.liveRooms.Rooms {

		for _, value := range v.Urls {

			if value.PushUrl == Url && 1 == value.State {

				value.LimitAudio = limit
				return nil
			}
		}
	}
	return errors.New("Not Found SetLimitAudio")
}

func (rs *RtmpStream) GetPushIdFromUrl(Url string) (error, string, int) {

	log.Infof("RtmpStream GetPushIdFromUrl Url=%s", Url)

	for _, v := range rs.liveRooms.Rooms {

		for _, value := range v.Urls {

			if value.PushUrl == Url {

				return nil, v.LiveRoomId, value.PushId
			}
		}
	}
	return errors.New("Not Found SetLimitAudio"), "", -1
}

func (rs *RtmpStream) GetProjectPushIdFromUrl(Url string) (error, int, int) {

	log.Infof("RtmpStream GetProjectPushIdFromUrl Url=%s", Url)

	for _, v := range rs.liveRooms.Rooms {

		for _, value := range v.Urls {

			if value.PushUrl == Url {

				return nil, v.ProjectId, value.PushId
			}
		}
	}
	return errors.New("Not Found SetLimitAudio"), -1, -1
}

func (rs *RtmpStream) GetPushFromUrl(Url string) (error, *configure.PushStreamUrl, string, int) {

	log.Infof("RtmpStream GetPushFromUrl Url=%s", Url)

	for _, v := range rs.liveRooms.Rooms {

		for _, value := range v.Urls {

			if value.PushUrl == Url {

				return nil, value, v.LiveRoomId, v.ProjectId
			}
		}
	}
	return errors.New("Not Found GetPushFromUrl"), nil, "", -1
}

func (rs *RtmpStream) GetMixFromUrl(Url string) (error, string, int, string) {

	log.Infof("RtmpStream GetMixFromUrl Url=%s", Url)

	for _, v := range rs.liveRooms.Rooms {

		if v.MixUrl == Url {
			return nil, v.LiveRoomId, v.ProjectId, v.MixSavePath
		}
	}
	return errors.New("Not Found GetPushFromUrl"), "", -1, ""
}

func (rs *RtmpStream) getLive(liveId string) (error, configure.Live) {

	log.Infof("getLive liveId=%s", liveId)
	for _, v := range configure.LiveRtmpcfg.Lives {

		if v.LiveId == liveId {

			return nil, v
		}
	}
	return errors.New("Not Found Live"), configure.Live{}
}

func (rs *RtmpStream) GetHolderUrlBase(liveRoomId string) (error, string, string) {

	log.Infof("RtmpStream GetHolderUrlBase liveRoomId=%s", liveRoomId)

	for _, v := range rs.liveRooms.Rooms {

		for _, value := range v.Urls {

			if value.UserType == configure.Holder {

				listenString := strconv.Itoa(configure.RtmpServercfg.Listen)
				return nil, value.RtmpBase, listenString
			}
		}
	}
	return errors.New("Not Found liveRoomId"), "", "-1"
}

func (rs *RtmpStream) getUrl(pushId int, live configure.Live) (error, configure.Url) {

	log.Infof("getUrl pushId=%s", pushId)
	for _, v := range live.Urls {

		if v.PushId == pushId {

			return nil, v
		}
	}
	return errors.New("Not Found Url"), configure.Url{}
}

//保存地址路径格式
//save+"/"+liveID+"/"+pushID+"/"+data+"/
func (rs *RtmpStream) getLocalFiles(liveId string, pushId int, date string) (error, []string) {

	log.Infof("getTimerFiles liveId=%s channalId=%s date=%s", liveId, pushId, date)

	//通过配置得到这个保存的目录
	var localPath string
	var urls []string

	errLive, live := rs.getLive(liveId)
	if errLive != nil {

		log.Error("getTimerFiles Failed getLive err != nil")
		return errLive, urls
	}

	errUrl, url := rs.getUrl(pushId, live)
	if errUrl != nil {

		log.Error("getTimerFiles Failed getUrl err != nil")
		return errUrl, urls
	}

	//得到保存地址
	pushIdString := strconv.Itoa(pushId)
	localPath = url.SavePath + "/" + liveId + "/" + pushIdString + "/" + date + "/"

	//得到指定目录下的所有文件
	files, err := ioutil.ReadDir(localPath)
	if err != nil {

		log.Error("getTimerFiles Failed ReadDir err=%s", err.Error())
		return errors.New("Read Dir Failed"), urls
	}

	// 获取文件，并输出它们的名字
	for _, file := range files {

		filePath := localPath + file.Name()
		urls = append(urls, filePath)
	}

	return nil, urls

}

//得到文件大小
func (rs *RtmpStream) getFileSize(path string) int64 {

	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (rs *RtmpStream) getFileMd5(filename string, md5str *string) error {

	f, err := os.Open(filename)
	if err != nil {

		errString := "Open File Failed err:" + err.Error()
		return errors.New(errString)
	}

	defer f.Close()

	md5hash := md5.New()
	if _, err := io.Copy(md5hash, f); err != nil {

		errString := "Copy Data Failed err:" + err.Error()
		return errors.New(errString)
	}

	md5hash.Sum(nil)
	*md5str = fmt.Sprintf("%x", md5hash.Sum(nil))
	return nil
}

func (rs *RtmpStream) findFileMD5(path string, size int64) (bool, string) {

	for _, v := range rs.md5s.Files {

		if (v.FilePath == path) && (v.Size == size) {

			return true, v.MD5
		}
	}
	return false, ""
}

//得到路径下所有文件
func (rs *RtmpStream) getDirFiles(pathDir string) (error, []string, []int64, []string, []string, []string, []string) {

	log.Infof("getDirFiles path=%s", pathDir)

	//得到指定目录下的所有文件
	var dirFiles []string
	var dirFilesSize []int64
	var dirFilesMd5 []string
	var startTimers []string
	var finishTimers []string
	var videoString []string
	files, err := ioutil.ReadDir(pathDir)
	if err != nil {

		log.Error("getDirFiles Failed ReadDir err=%s", err.Error())
		return errors.New("Read Dir Failed"), dirFiles, dirFilesSize, dirFilesMd5, startTimers, finishTimers, videoString
	}

	// 获取文件，并输出它们的名字
	for _, file := range files {

		filePath := pathDir + "/" + file.Name()
		fileSize := rs.getFileSize(filePath)
		dirFilesSize = append(dirFilesSize, fileSize)

		//分解文件的起始和终止时间
		nameWithSuffix := path.Base(filePath)
		//得到后缀
		suffix := path.Ext(nameWithSuffix)
		//得到文件名不带后缀
		nameOnly := strings.TrimSuffix(nameWithSuffix, suffix)

		arr := strings.Split(nameOnly, "_")

		if len(arr) != 3 {

			log.Error("Replay File=%s Name Split Failed")
			continue

		}
		videoName := arr[0]
		videoString = append(videoString, videoName)

		startTimer := arr[1]
		startTimers = append(startTimers, startTimer)

		finishTimer := arr[2]
		finishTimers = append(finishTimers, finishTimer)

		log.Infof("Replay File=%s Start Timer=%s Finish Timer=%s videoString=%s", filePath, startTimer, finishTimer, videoName)

		exits, md5str := rs.findFileMD5(filePath, fileSize)
		if exits {

			dirFilesMd5 = append(dirFilesMd5, md5str)

		} else {

			//得到文件的md5
			log.Infof("Begin Get %s File MD5", filePath)
			md5Err := rs.getFileMd5(filePath, &md5str)
			if md5Err != nil {
				md5str = ""
			}
			log.Infof("Get Md5 %s File Md5 %s", filePath, md5str)
			dirFilesMd5 = append(dirFilesMd5, md5str)

			//记录新的文件MD5
			fileMd5 := &configure.FileToMd5{
				FilePath: filePath,
				Size:     fileSize,
				MD5:      md5str,
				Start:    startTimer,
				Finish:   finishTimer,
			}
			rs.md5s.Files = append(rs.md5s.Files, fileMd5)
		}
		dirFiles = append(dirFiles, file.Name())

	}
	//会写文件
	rs.writeReplayConfig()
	return nil, dirFiles, dirFilesSize, dirFilesMd5, startTimers, finishTimers, videoString
}

func (rs *RtmpStream) initLiveRooms() {

	log.Infof("RtmpStream initLiveRooms")

	//设置静态数据
	for _, value := range configure.LiveRtmpcfg.Lives {

		//初始化配置属性
		var liveRoom *configure.LiveRoom = new(configure.LiveRoom)
		liveRoom.LiveRoomId = value.LiveId
		liveRoom.MixRtmpBase = value.MixRtmpBase
		liveRoom.MixSavePath = value.MixSavePath
		liveRoom.MixSaveUrl = value.MixSaveUrl
		liveRoom.MixUrl = ""
		liveRoom.ProjectId = -1
		for _, v := range value.Urls {

			var pushUrl *configure.PushStreamUrl = new(configure.PushStreamUrl)
			pushUrl.PushId = v.PushId
			pushUrl.UserType = v.UserType
			pushUrl.VideoType = v.VideoType
			pushUrl.RtmpBase = v.RtmpBase
			pushUrl.SavePath = v.SavePath
			pushUrl.SaveUrl = v.SaveUrl
			pushUrl.RequestUrl = v.RequestUrl
			pushUrl.LimitAudio = false

			if pushUrl.VideoType == configure.Camera {
				pushUrl.VideoSize = configure.Camera_Size
			} else if pushUrl.VideoType == configure.PCCamera {
				pushUrl.VideoSize = configure.PCCamera_Size
			} else if pushUrl.VideoType == configure.DesktopShare {
				pushUrl.VideoSize = configure.DesktopShare_Size
			}

			liveRoom.Urls = append(liveRoom.Urls, pushUrl)
		}
		rs.liveRooms.Rooms = append(rs.liveRooms.Rooms, liveRoom)

		//初始化混音属性
		var liveRoomMix *LiveRoomMix = new(LiveRoomMix)
		liveRoomMix.LiveRoomId = liveRoom.LiveRoomId
		liveRoomMix.MixRtmpBase = liveRoom.MixRtmpBase
		liveRoomMix.MixSavePath = liveRoom.MixSavePath
		liveRoom.MixSaveUrl = liveRoom.MixSaveUrl
		liveRoomMix.MixUrl = liveRoom.MixUrl
		liveRoomMix.ProjectId = liveRoom.ProjectId
		liveRoomMix.cmdExec = nil
		rs.mixRooms.Rooms = append(rs.mixRooms.Rooms, liveRoomMix)

		// log.Infof("liveRoomMix LiveRoomId:%s MixUrl:%s ProjectId:%d", liveRoomMix.LiveRoomId, liveRoomMix.MixUrl, liveRoomMix.ProjectId)
	}
	log.Infof("Get Static Json:%v", rs.liveRooms.Rooms)
}

//设置rtmp的状态
func (rs *RtmpStream) SetMixState(projectId int, liveRoomId string, mixUrl string, state bool, mixSrc string) (err error) {

	log.Infof("RtmpStream SetMixState")

	for _, value := range rs.mixRooms.Rooms {

		if value.LiveRoomId == liveRoomId {

			if state {
				value.MixUrl = mixUrl
				value.ProjectId = projectId
			} else {
				value.MixUrl = ""
				value.ProjectId = -1
			}
		}
	}

	for _, value := range rs.liveRooms.Rooms {

		if value.LiveRoomId == liveRoomId {

			if state {
				value.MixSrc = mixSrc
				value.MixUrl = mixUrl
			} else {
				value.MixUrl = ""
				value.MixSrc = ""
			}
			return nil
		}
	}
	return errors.New("SetMixState Not Found LiveRoomid")
}

//设置rtmp的状态
func (rs *RtmpStream) IsMixSrc(mixUrl string) (bool, int) {

	log.Infof("RtmpStream IsMixSrc")

	for _, value := range rs.liveRooms.Rooms {

		if value.MixUrl == mixUrl {

			return true, value.ProjectId
		}

	}
	return false, -1
}

//设置rtmp的状态
func (rs *RtmpStream) SetStartState(projectId int, liveRoomId string, pushId int, url string, user_type int) (err error) {

	log.Infof("RtmpStream SetStartState")

	for _, value := range rs.liveRooms.Rooms {

		if value.LiveRoomId == liveRoomId {
			value.ProjectId = projectId

			for _, v := range value.Urls {

				if v.PushId == pushId {

					v.PushUrl = url
					v.UserType = configure.UserTypeEunm(user_type)
					v.State = 1
					return nil
				}
			}

		}

		if value.LiveRoomId == liveRoomId {

			value.ProjectId = projectId
			log.Infof("RtmpStream SetStartState set ProjectId=%d", value.ProjectId)
			for _, v := range value.Urls {

				if v.PushId == pushId {

					v.PushUrl = url
					v.UserType = configure.UserTypeEunm(user_type)
					v.State = 1
					return nil
				}
			}
		}
	}
	return errors.New("SetStartState Not Found LiveRoomid/PushId/Url")
}

func (rs *RtmpStream) SetMixCmd(projectId int, cmd *exec.Cmd) {

	log.Infof("RtmpStream SetMixCmd")

	for _, value := range rs.mixRooms.Rooms {

		if value.ProjectId == projectId {
			value.cmdExec = cmd
			return
		}
	}
}

func (rs *RtmpStream) GetMixCmd(projectId int) *exec.Cmd {

	log.Infof("RtmpStream GetMixCmd")

	for _, value := range rs.mixRooms.Rooms {

		if value.ProjectId == projectId {
			return value.cmdExec
		}
	}
	return nil
}

//获得当前所有的直播房间结构
func (rs *RtmpStream) GetRtmpList() (error, configure.LiveRooms) {

	log.Infof("---->>>> RtmpStream CreateRtmpList")

	//得到当前推流列表
	currentLiveRooms := rs.liveRooms

	//设置回看文件
	for _, v := range currentLiveRooms.Rooms {

		//Mix
		checkMixDir := v.MixSavePath + "/" + v.LiveRoomId + "/" + strconv.Itoa(v.ProjectId)
		exist, pathErr := rs.pathExists(checkMixDir)
		if pathErr == nil && exist {

			// err, mixDirFiles, mixDirFilesSize, mixDirFilesMd5, mixDirFilesStart, mixDirFilesFinish, _ := rs.getDirFiles(checkMixDir)
			// if err != nil {
			// 	return nil, currentLiveRooms
			// }
			v.MixReplays = v.MixReplays[0:0]
			// for i, k := range mixDirFiles {

			// 	replay := &configure.Replay{
			// 		Addr:   v.MixSaveUrl + "/" + v.LiveRoomId + "/" + strconv.Itoa(v.ProjectId) + "/" + k,
			// 		Size:   mixDirFilesSize[i],
			// 		Md5:    mixDirFilesMd5[i],
			// 		Start:  mixDirFilesStart[i],
			// 		Finish: mixDirFilesFinish[i],
			// 	}
			// 	v.MixReplays = append(v.MixReplays, replay)
			// }
		}

		//Replay
		// for _, value := range v.Urls {

		// 	checkDir := value.SavePath + "/" + v.LiveRoomId + "/" + strconv.Itoa(v.ProjectId)
		// 	err, dirFiles, dirFilesSize, dirFilesMd5, dirFilesStart, dirFilesFinish, dirVideoString := rs.getDirFiles(checkDir)
		// 	if err != nil {

		// 		return nil, currentLiveRooms
		// 	}
		// 	value.Replays = value.Replays[0:0]

		// 	for index, w := range dirFiles {

		// 		var videoType configure.VideoTypeEunm
		// 		var videoSize string
		// 		if dirVideoString[index] == "Camera" {
		// 			videoType = configure.Camera
		// 			videoSize = ""
		// 		} else if dirVideoString[index] == "PCCamera" {

		// 			videoType = configure.PCCamera
		// 			videoSize = configure.PCCamera_Size
		// 		} else if dirVideoString[index] == "Camera" {

		// 			videoType = configure.DesktopShare
		// 			videoSize = configure.DesktopShare_Size
		// 		}

		// 		// replay := &configure.Replay{
		// 		// 	Addr:      value.SaveUrl + "/" + v.LiveRoomId + "/" + strconv.Itoa(v.ProjectId) + "/" + w,
		// 		// 	Size:      dirFilesSize[index],
		// 		// 	Md5:       dirFilesMd5[index],
		// 		// 	Start:     dirFilesStart[index],
		// 		// 	Finish:    dirFilesFinish[index],
		// 		// 	VideoType: videoType,
		// 		// 	VideoSize: videoSize,
		// 		// }
		// 		// value.Replays = append(value.Replays, replay)
		// 	}
		// }
	}

	return nil, currentLiveRooms
}

//获得当前指定的直播房间结构
func (rs *RtmpStream) GetSingleRtmpList(projectId int) (error, configure.LiveRooms) {

	log.Infof("---->>>> RtmpStream GetSingleRtmpList")

	//得到当前推流列表
	var currentLiveRooms configure.LiveRooms

	//设置回看文件
	for _, v := range rs.liveRooms.Rooms {

		if v.ProjectId == projectId {

			liveRoom := &configure.LiveRoom{

				LiveRoomId:  v.LiveRoomId,
				ProjectId:   v.ProjectId,
				MixSavePath: v.MixSavePath,
				MixSaveUrl:  v.MixSaveUrl,
				MixRtmpBase: v.MixRtmpBase,
				MixUrl:      v.MixUrl,
			}
			currentLiveRooms.Rooms = append(currentLiveRooms.Rooms, liveRoom)

			//Mix
			checkMixDir := v.MixSavePath + "/" + v.LiveRoomId + "/" + strconv.Itoa(projectId)
			exist, pathErr := rs.pathExists(checkMixDir)
			if pathErr == nil && exist {

				/*err, mixDirFiles, mixDirFilesSize, mixDirFilesMd5, mixDirFilesStart, mixDirFilesFinish, _ := rs.getDirFiles(checkMixDir)
				if err != nil {
					return nil, currentLiveRooms
				}*/
				liveRoom.MixReplays = liveRoom.MixReplays[0:0]
				/*for i, k := range mixDirFiles {

					replay := &configure.Replay{
						Addr:   v.MixSaveUrl + "/" + v.LiveRoomId + "/" + strconv.Itoa(v.ProjectId) + "/" + k,
						Size:   mixDirFilesSize[i],
						Md5:    mixDirFilesMd5[i],
						Start:  mixDirFilesStart[i],
						Finish: mixDirFilesFinish[i],
					}
					liveRoom.MixReplays = append(liveRoom.MixReplays, replay)
				}*/
			}

			//Replayer
			for _, value := range v.Urls {

				pushStreamUrl := value
				liveRoom.Urls = append(liveRoom.Urls, pushStreamUrl)

				/*checkDir := value.SavePath + "/" + v.LiveRoomId + "/" + strconv.Itoa(v.ProjectId)
				exist, pathErr := rs.pathExists(checkMixDir)
				if pathErr == nil && exist {
					err, dirFiles, dirFilesSize, dirFilesMd5, dirFilesStart, dirFilesFinish, dirVideoString := rs.getDirFiles(checkDir)
					if err != nil {

						return nil, currentLiveRooms
					}

					for index, w := range dirFiles {

						var videoType configure.VideoTypeEunm
						var videoSize string
						if dirVideoString[index] == "Camera" {
							videoType = configure.Camera
							videoSize = ""
						} else if dirVideoString[index] == "PCCamera" {

							videoType = configure.PCCamera
							videoSize = configure.PCCamera_Size
						} else if dirVideoString[index] == "Camera" {

							videoType = configure.DesktopShare
							videoSize = configure.DesktopShare_Size
						}

						replay := &configure.Replay{
							Addr:      value.SaveUrl + "/" + v.LiveRoomId + "/" + strconv.Itoa(v.ProjectId) + "/" + w,
							Size:      dirFilesSize[index],
							Md5:       dirFilesMd5[index],
							Start:     dirFilesStart[index],
							Finish:    dirFilesFinish[index],
							VideoType: videoType,
							VideoSize: videoSize,
						}
						pushStreamUrl.Replays = append(pushStreamUrl.Replays, replay)
					}
				}*/
			}
		}
	}

	return nil, currentLiveRooms
}

//获得当前指定的直播房间结构
func (rs *RtmpStream) GetProjectReplayList(projectId int) configure.ReplayRooms {

	log.Infof("---->>>> RtmpStream GetProjectReplayList")

	//得到当前推流列表
	var replayRooms configure.ReplayRooms

	//设置回看文件
	for _, v := range rs.liveRooms.Rooms {

		replayRoom := &configure.ReplayRoom{

			LiveRoomId: v.LiveRoomId,
		}

		//Mix
		checkMixDir := v.MixSavePath + "/" + v.LiveRoomId + "/" + strconv.Itoa(projectId)
		log.Infof("---->>>> RtmpStream GetProjectReplayList checkMixDir=%s", checkMixDir)

		exist, pathErr := rs.pathExists(checkMixDir)
		if pathErr == nil && exist {

			_, mixDirFiles, mixDirFilesSize, mixDirFilesMd5, mixDirFilesStart, mixDirFilesFinish, _ := rs.getDirFiles(checkMixDir)
			replayRoom.MixReplays = v.MixReplays[0:0]
			for i, k := range mixDirFiles {

				replay := &configure.Replay{
					Addr:   v.MixSaveUrl + "/" + v.LiveRoomId + "/" + strconv.Itoa(v.ProjectId) + "/" + k,
					Size:   mixDirFilesSize[i],
					Md5:    mixDirFilesMd5[i],
					Start:  mixDirFilesStart[i],
					Finish: mixDirFilesFinish[i],
				}
				replayRoom.MixReplays = append(replayRoom.MixReplays, replay)
			}
		}

		log.Infof("pool liveRoomId=%s", replayRoom.LiveRoomId)
		for _, value := range v.Urls {

			checkDir := value.SavePath + "/" + v.LiveRoomId + "/" + strconv.Itoa(projectId)
			log.Infof("---->>>> RtmpStream GetProjectReplayList checkDir=%s", checkDir)

			exist, pathErr = rs.pathExists(checkDir)
			if pathErr == nil && exist {

				log.Infof("pool Get Replay File Begin UrlDir=%s", checkDir)

				err, dirFiles, dirFilesSize, dirFilesMd5, dirFilesStart, dirFilesFinish, dirVideoString := rs.getDirFiles(checkDir)
				if err != nil {

					continue
				}
				log.Infof("pool Get Replay File End UrlDir=%s", checkDir)

				replayStream := &configure.ReplayStreamUrl{
					PushId: value.PushId,
				}
				replayRoom.Urls = append(replayRoom.Urls, replayStream)

				for index, w := range dirFiles {

					var videoType configure.VideoTypeEunm
					var videoSize string
					if dirVideoString[index] == "Camera" {
						videoType = configure.Camera
						videoSize = ""
					} else if dirVideoString[index] == "PCCamera" {

						videoType = configure.PCCamera
						videoSize = configure.PCCamera_Size
					} else if dirVideoString[index] == "Camera" {

						videoType = configure.DesktopShare
						videoSize = configure.DesktopShare_Size
					}

					replay := &configure.Replay{
						Addr:      value.SaveUrl + "/" + v.LiveRoomId + "/" + strconv.Itoa(projectId) + "/" + w,
						Size:      dirFilesSize[index],
						Md5:       dirFilesMd5[index],
						Start:     dirFilesStart[index],
						Finish:    dirFilesFinish[index],
						VideoType: videoType,
						VideoSize: videoSize,
					}
					replayStream.Replays = append(replayStream.Replays, replay)
				}

			}

		}

		replayRooms.Rooms = append(replayRooms.Rooms, replayRoom)
	}
	log.Infof("---->>>> RtmpStream GetProjectReplayList---->>>>")

	return replayRooms
}

func (rs *RtmpStream) checkViewers(Url string) {

	//统计观看者相关信息
	var count int = 0
	for item := range rs.GetStreams().IterBuffered() {
		ws := item.Val.(*Stream).GetWs()
		for s := range ws.IterBuffered() {
			if pw, ok := s.Val.(*PackWriterCloser); ok {
				if pw.GetWriter() != nil {
					switch pw.GetWriter().(type) {
					case *VirWriter:
						v := pw.GetWriter().(*VirWriter)
						count++

						if v.Info().URL == Url {
							log.Infof("---->>>>Current Viewers Url=%s PeerIp=%s\n", v.Info().URL, v.WriteBWInfo.PeerIP)
						}
					}
				}
			}
		}
	}
}

//判断被观看的发布者是否存在
func (rs *RtmpStream) closeViewers() {

	//统计观看者相关信息
	for item := range rs.GetStreams().IterBuffered() {
		ws := item.Val.(*Stream).GetWs()
		for s := range ws.IterBuffered() {
			if pw, ok := s.Val.(*PackWriterCloser); ok {
				if pw.GetWriter() != nil {
					switch pw.GetWriter().(type) {
					case *VirWriter:
						v := pw.GetWriter().(*VirWriter)

						publishExist := rs.existPublisher(v.Info().URL)
						if !publishExist {
							rs.streams.Remove(v.Info().Key)
							log.Infof("Remove Viewer---->>>> URL=%s Key=%s", v.Info().URL, v.Info().Key)
						}
					}
				}
			}
		}
	}
}

func (rs *RtmpStream) existPublisher(Url string) bool {

	for item := range rs.GetStreams().IterBuffered() {

		if s, ok := item.Val.(*Stream); ok {
			if s.GetReader() != nil {
				switch s.GetReader().(type) {
				case *VirReader:
					v := s.GetReader().(*VirReader)

					if v.Info().URL == Url {
						return true
					}
				}
			}
		}
	}
	return false
}

func (rs *RtmpStream) PublisherExist(uri string) bool {

	//发布者统计
	for item := range rs.GetStreams().IterBuffered() {

		if s, ok := item.Val.(*Stream); ok {
			if s.GetReader() != nil {
				switch s.GetReader().(type) {
				case *VirReader:
					v := s.GetReader().(*VirReader)

					if v.Info().URL == uri {
						return true
					}
				}
			}
		}
	}
	return false
}

func (rs *RtmpStream) checkCamera(uri string) bool {

	log.Infof("RtmpStream checkCamera uri %s", uri)
	return strings.Contains(uri, "?v")
}

//定时查询发布者和观看者信息
func (rs *RtmpStream) checkPublisher() {

	for {

		time.Sleep(time.Duration(10) * time.Second)

		log.Infof("checkPublisher CheckStatic ---->>>>")

		//发布者统计
		for item := range rs.GetStreams().IterBuffered() {

			if s, ok := item.Val.(*Stream); ok {
				if s.GetReader() != nil {
					switch s.GetReader().(type) {
					case *VirReader:
						v := s.GetReader().(*VirReader)

						log.Infof("Current Publisher Url=%s PeerIp=%s\n", v.Info().URL, v.ReadBWInfo.PeerIP)

						rs.checkViewers(v.Info().URL)
					}
				}
			}
		}

		//保活检测
		log.Infof("checkPublisher CheckAlive ---->>>>")
		for item := range rs.streams.IterBuffered() {

			v := item.Val.(*Stream)
			log.Infof("RtmpStream checkAlive %s", v.info.String())

			if v.CheckAlive() == 0 {

				//判断是否不是球机
				// if rs.checkCamera(v.info.URL) {

				log.Infof("RtmpStream checkAlive remove %s", item.Key)

				//查看对应的切片机起来没，起来的话，关闭切片机
				if v.FFmpeg {
					v.FFmpeg = false
					v.Stopffmpeg()
				}
				rs.streams.Remove(item.Key)
				// }
			}
		}

		//可使用检测（只针对观看者检测，如果判断被观看的发布对象已经下线，则强制关闭观看对象）
		log.Infof("checkPublisher closeViewers ---->>>>")
		rs.closeViewers()
	}
}

//定时查询发布者和观看者信息
func (rs *RtmpStream) getPublisherStream(mixUrl string) *Stream {

	for item := range rs.GetStreams().IterBuffered() {

		if s, ok := item.Val.(*Stream); ok {
			if s.GetReader() != nil {
				switch s.GetReader().(type) {
				case *VirReader:
					v := s.GetReader().(*VirReader)

					if v.Info().URL == mixUrl {
						return s
					}
				}
			}
		}
	}
	return nil
}

//定期清空不存在的文件
func (rs *RtmpStream) checkMediaFile() {

	for {

		time.Sleep(time.Duration(10) * time.Second)

		// log.Infof("RtmpStream checkMediaFile ---->>>>")
		var changed bool = false
		for i := 0; i < len(rs.md5s.Files); {

			filePath := rs.md5s.Files[i].FilePath
			exits, _ := rs.pathExists(filePath)
			if !exits {

				changed = true
				rs.md5s.Files = append(rs.md5s.Files[:i], rs.md5s.Files[i+1:]...)
				log.Infof("checkMediaFile Delete Not Found File=%s", filePath)
			} else {
				i++
			}
		}

		if changed {
			rs.writeReplayConfig()
		}
	}
}

//混音推流
func (rs *RtmpStream) Mixffmpeg(projectId int, mainUrl string, subUrl string, pushUrl string) {

	log.Infof("Mixffmpeg---->>>> projectId=%d mainUrl=%s subUrl=%s pushUrl=%s", projectId, mainUrl, subUrl, pushUrl)

	// time.Sleep(time.Duration(2) * time.Second)

	//判断拉流地址是否存在
	mainExists := rs.existPublisher(mainUrl)
	subExists := rs.existPublisher(subUrl)
	if !mainExists || !subExists {

		log.Infof("Mixffmpeg Failed mainUrl=%s mainExists=%d subUrl=%s subExists=%d", mainUrl, mainExists, subUrl, subExists)
		return
	}

	//判断推流地址是否存在
	mixExists := rs.existPublisher(pushUrl)
	if mixExists {

		log.Infof("Mixffmpeg Failed pushUrl=%s mixExists=%d", pushUrl, mixExists)
		return
	}

	//检测两条数据流是否都有音频（如果没有则进行混流的方法会报错）

	//停止以前的混音
	input := pushUrl
	cmd := ""
	cmd += "a="
	cmd += input
	cmd += ";b=`ps  -ef |grep $a|grep -v grep`;if [ \"x$b\" != \"x\" ];then  ps  -ef|grep $a|grep -v grep|cut -c 9-15 | xargs kill -9 ;fi"
	log.Infof("Stop Old FFMpeg Mix Process When Start Mix FFMpeg cmd---->>>>\n %s", cmd)

	lsCmd := exec.Command("/bin/sh", "-c", cmd)
	lsCmd.Run()

	//*置混音参数*/
	/*
		args := configure.GetFfmpeg() //FFmpeg可执行路径
		// args += " -thread_queue_size 32 "                                                                        //队列长度
		args += " -rtbufsize 1024 "                                                                              //缓冲区大小
		args += " -start_time_realtime 0 "                                                                       //起始时钟
		args += " -i " + mainUrl                                                                                 //主流（背景流）
		args += " -i " + subUrl                                                                                  //子流（小窗流）
		args += " -filter_complex \"[1]scale=200:200[pip];[0][pip]overlay=0:0;[0:a][1:a]amerge=inputs=2[aout]\"" //混流内容
		args += "  -tune zerolatency"                                                                            //实时编码
		args += "  -c:v libx264 "                                                                                //视频格式
		args += "  -preset superfast "                                                                           //编码形式
		// args += "  -vprofile baseline "                                                                          //参数profile
		args += "  -x264opts \"bframes=0\" -b:v 700k " //视频配置
		args += " -g 1 "                               //GOP参数
		args += " -map [aout] "                        //对应上面的音频输出aout
		args += " -acodec aac -ac 2  -ar 44100 "       //音频推流内容
		args += " -y -f flv "                          //推流格式
		args += " -flvflags no_duration_filesize "     //屏蔽提示
		args += pushUrl + "\n"                         //推流地址*/

	args := configure.GetFfmpeg() //FFmpeg可执行路径
	// args += " -thread_queue_size 32 "                                                                        //队列长度
	args += " -rtbufsize 1024 "                                                                              //缓冲区大小
	args += " -start_time_realtime 0 "                                                                       //起始时钟
	args += " -i " + mainUrl                                                                                 //主流（背景流）
	args += " -i " + subUrl                                                                                  //子流（小窗流）
	args += " -filter_complex \"[1]scale=200:200[pip];[0][pip]overlay=0:0;[0:a][1:a]amerge=inputs=2[aout]\"" //混流内容
	args += "  -tune zerolatency"                                                                            //实时编码
	args += "  -c:v libx264 "                                                                                //视频格式
	args += "  -x264opts \"bframes=0\" -b:v 700k "                                                           //视频配置
	args += " -g 5 "                                                                                         //GOP参数
	args += " -map [aout] "                                                                                  //对应上面的音频输出aout
	args += " -acodec aac -ac 2  -ar 44100 "                                                                 //音频推流内容
	args += " -y -f flv "                                                                                    //推流格式
	args += " -flvflags no_duration_filesize "                                                               //屏蔽提示
	args += pushUrl + "\n"                                                                                   //推流地址

	log.Infof("Mixffmpeg Cmd ---->>>> %s", args)
	cmdExec := exec.Command("/bin/sh", "-c", args)

	// 命令的错误输出和标准输出都连接到同一个管道
	stdout, _ := cmdExec.StdoutPipe()
	cmdExec.Stderr = cmdExec.Stdout

	errCmd := cmdExec.Start()
	if errCmd != nil {

		log.Error("Mixffmpeg Failed")
		return
	}

	errRoom, liveRoom := rs.GetLiveRoom(projectId)
	if errRoom != nil {
		log.Error("---->>>>Server Mixffmpeg Start Failed MixFFmpeg %s", pushUrl)
		return
	}

	rs.SetMixState(projectId, liveRoom.LiveRoomId, pushUrl, true, mainUrl)

	rs.SetMixCmd(projectId, cmdExec)

	// 从管道中实时获取输出并打印到终端
	for {
		tmp := make([]byte, 1024)
		_, err := stdout.Read(tmp)
		fmt.Print(string(tmp))
		if err != nil {
			break
		}
	}

	cmdExec.Wait()
}

func (rs *RtmpStream) StopMixffmpeg(projectId int, mixUrl string) {

	defer func() {
		if r := recover(); r != nil {
			log.Error("rtmp Stopffmpeg  panic: ", r)
		}
	}()

	errRoom, liveRoom := rs.GetLiveRoom(projectId)
	if errRoom != nil {
		log.Error("---->>>>Server StopMixffmpeg Failed MixFFmpeg %s", mixUrl)
		return
	}
	rs.SetMixState(projectId, liveRoom.LiveRoomId, mixUrl, false, "")

	s := rs.getPublisherStream(mixUrl)
	if s != nil {

		//修改文件名称为结束时间
		curPath, _ := filepath.Split(s.saveFile)
		//得到文件名带后缀
		nameWithSuffix := path.Base(s.saveFile)
		//得到后缀
		suffix := path.Ext(nameWithSuffix)
		//得到文件名不带后缀
		nameOnly := strings.TrimSuffix(nameWithSuffix, suffix)
		//添加结束时间
		newName := nameOnly + "_" + time.Now().Format("20060102T150405") + suffix
		//得到新的文件名
		newMediaFile := curPath + newName
		//重命名
		err := os.Rename(s.saveFile, newMediaFile)
		if err != nil {

			log.Errorf("Media File ReName Failed %s err=%s", s.saveFile, err.Error())
			return
		}
		log.Infof("Media File ReName Success %s", newMediaFile)
	}

	//停止原有的拉流FFMpeg进程
	input := mixUrl
	cmd := ""
	cmd += "a="
	cmd += input
	cmd += ";b=`ps  -ef |grep $a|grep -v grep`;if [ \"x$b\" != \"x\" ];then  ps  -ef|grep $a|grep -v grep|cut -c 9-15 | xargs kill -9 ;fi"
	log.Infof("Stop Old FFMpeg Process When Start FFMpeg cmd---->>>>\n %s", cmd)

	//执行进程停止
	lsCmd := exec.Command("/bin/sh", "-c", cmd)
	lsCmd.Run()

	mixCmd := rs.GetMixCmd(projectId)
	if nil != mixCmd {

		log.Infof("kill s.process: ", mixCmd.Process)
		mixCmd.Process.Kill()
		rs.SetMixCmd(projectId, nil)
	}

	//对于所有正在观看混音的流全都关闭
	for item := range rs.GetStreams().IterBuffered() {

		ws := item.Val.(*Stream).GetWs()
		for s := range ws.IterBuffered() {
			if pw, ok := s.Val.(*PackWriterCloser); ok {
				if pw.GetWriter() != nil {
					switch pw.GetWriter().(type) {
					case *VirWriter:
						v := pw.GetWriter().(*VirWriter)

						if v.Info().URL == mixUrl {

							log.Infof("StopMixffmpeg  %s", v.Info().URL)
							item.Val.(*Stream).closeInter()
						}
					}
				}
			}
		}
	}

}

type Stream struct {
	isStart    bool
	FFmpeg     bool
	cache      *cache.Cache
	r          av.ReadCloser
	ws         cmap.ConcurrentMap
	info       av.Info
	cmdExec    *exec.Cmd
	liveRoomId string
	limitAudio bool
	rtmpStream *RtmpStream
	saveFile   string
	lastAlive  time.Time
}

type PackWriterCloser struct {
	init bool
	w    av.WriteCloser
}

func (p *PackWriterCloser) GetWriter() av.WriteCloser {
	return p.w
}

func NewStream(rs *RtmpStream, limitAudio bool) *Stream {
	return &Stream{
		cache:      cache.NewCache(),
		ws:         cmap.New(),
		FFmpeg:     false,
		rtmpStream: rs,
		limitAudio: limitAudio,
	}
}

func (s *Stream) ID() string {
	if s.r != nil {
		return s.r.Info().UID
	}
	return EmptyID
}

func (s *Stream) GetReader() av.ReadCloser {
	return s.r
}

func (s *Stream) GetWs() cmap.ConcurrentMap {
	return s.ws
}

func (s *Stream) Copy(dst *Stream) {
	for item := range s.ws.IterBuffered() {
		v := item.Val.(*PackWriterCloser)
		s.ws.Remove(item.Key)
		v.w.CalcBaseTimestamp()
		dst.AddWriter(v.w)
	}
}

//启动FFmpeg
func (s *Stream) Startffmpeg(Url string, SaveFile string) {

	time.Sleep(time.Duration(5) * time.Second)

	log.Infof("Startffmpeg---->>>> Url=%s SaveFile=%s", Url, SaveFile)

	exist := s.rtmpStream.existPublisher(Url)
	if !exist {

		log.Infof("Startffmpeg---->>>> Not Found Url Publisher Url=%s", Url)
		return
	}

	//检测是否是混流，如果是混流则不进行关闭之前的（因为混流地址为固定地址）
	existMix := strings.Contains(Url, "mixStream")
	if !existMix {

		//停止原有的拉流FFMpeg进程
		input := Url
		cmd := ""
		cmd += "a="
		cmd += input
		cmd += ";b=`ps  -ef |grep $a|grep -v grep`;if [ \"x$b\" != \"x\" ];then  ps  -ef|grep $a|grep -v grep|cut -c 9-15 | xargs kill -9 ;fi"
		log.Infof("Stop Old FFMpeg Process When Start FFMpeg cmd---->>>>\n %s", cmd)

		//执行进程停止
		lsCmd := exec.Command("/bin/sh", "-c", cmd)
		lsCmd.Run()
	}

	args := configure.GetFfmpeg() + " -v verbose -i " + Url + " -codec copy " + SaveFile + "\n"

	log.Infof("Startffmpeg %s", args)

	cmdExec := exec.Command("/bin/sh", "-c", args)

	err := cmdExec.Start()
	if err != nil {

		log.Error("Startffmpeg Failed")
		return
	}

	s.saveFile = SaveFile
	s.FFmpeg = true
	s.cmdExec = cmdExec
	err = cmdExec.Wait()
}

func PathExists(path string) (bool, error) {

	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *Stream) Stopffmpeg() {

	defer func() {
		if r := recover(); r != nil {
			log.Error("rtmp Stopffmpeg  panic: ", r)
		}
	}()

	//终止FFMpeg
	if nil != s.cmdExec {

		log.Infof("kill s.process: ", s.cmdExec.Process)
		s.cmdExec.Process.Kill()
	}

	s.FFmpeg = false
	s.cmdExec = nil

	//修改文件名称为结束时间
	curPath, _ := filepath.Split(s.saveFile)
	//得到文件名带后缀
	nameWithSuffix := path.Base(s.saveFile)
	//得到后缀
	suffix := path.Ext(nameWithSuffix)
	//得到文件名不带后缀
	nameOnly := strings.TrimSuffix(nameWithSuffix, suffix)
	//添加结束时间
	newName := nameOnly + "_" + time.Now().Format("20060102T150405") + suffix
	//得到新的文件名
	newMediaFile := curPath + newName

	exists, exErr := PathExists(s.saveFile)
	if exErr != nil {
		log.Errorf("Media File ReName Not Exists saveFile=%s err=%s", s.saveFile, exErr.Error())
		return
	}

	if !exists {
		log.Errorf("Media File ReName Not Exists saveFile=%s", s.saveFile)
		return
	}

	//重命名
	err := os.Rename(s.saveFile, newMediaFile)
	if err != nil {

		log.Errorf("Media File ReName Failed saveFile=%s newMediaFile=%s err=%s", s.saveFile, newMediaFile, err.Error())
		return
	}
	log.Infof("Media File ReName Success %s", newMediaFile)
}

func (s *Stream) AddReader(r av.ReadCloser, liveRoomId string) {

	s.r = r
	s.liveRoomId = liveRoomId
	log.Infof("Stream AddReader Info=%s liveRoomId=%s", s.info.String(), liveRoomId)
	go s.TransStart()
}

func (s *Stream) AddWriter(w av.WriteCloser) {

	info := w.Info()
	log.Infof("AddWriter:%v", s.info)
	pw := &PackWriterCloser{w: w}
	s.ws.Set(info.UID, pw)

	log.Infof("AddWriter ws Count:%v", s.ws.Count())
}

func (s *Stream) TransStart() {
	s.isStart = true
	var p av.Packet

	log.Infof("TransStart:%v", s.info)

	time.Sleep(time.Duration(1) * time.Second)

	for {
		if !s.isStart {
			log.Info("Stream stop: call closeInter", s.info)
			s.closeInter()
			return
		}

		//从网络中读取视频数据
		for {
			err := s.r.Read(&p)
			if err != nil {
				log.Error("Stream Read error:", s.info, err)
				s.isStart = false
				s.closeInter()
				return
			}
			break
		}

		s.cache.Write(p)

		if s.ws.IsEmpty() {
			continue
		}

		for item := range s.ws.IterBuffered() {
			v := item.Val.(*PackWriterCloser)
			if !v.init {
				//log.Infof("cache.send: %v", v.w.Info())
				if err := s.cache.Send(v.w); err != nil {
					log.Infof("[%s] send cache packet error: %v, remove", v.w.Info(), err)
					s.ws.Remove(item.Key)
					continue
				}
				v.init = true
			} else {
				new_packet := p
				//writeType := reflect.TypeOf(v.w)
				//log.Infof("w.Write: type=%v, %v", writeType, v.w.Info())
				if err := v.w.Write(&new_packet); err != nil {
					//log.Infof("[%s] write packet error: %v, remove", v.w.Info(), err)
					s.ws.Remove(item.Key)
				}
			}
		}

	}
}

func (s *Stream) TransStop() {

	log.Infof("TransStop: %s", s.info.Key)

	if s.isStart && s.r != nil {
		s.r.Close(errors.New("stop old"))
	}

	s.isStart = false
}

func (s *Stream) CheckAlive() (n int) {

	if s.r != nil && s.isStart {

		if s.r.Alive() {
			n++
		} else {
			log.Errorf("CheckAlive Read Failed Timeout %s", s.info.String())
			s.r.Close(errors.New("read timeout"))
		}
	}

	for item := range s.ws.IterBuffered() {

		v := item.Val.(*PackWriterCloser)
		if v.w != nil {

			if !v.w.Alive() && s.isStart {
				s.ws.Remove(item.Key)
				log.Error("CheckAlive Write Failed Write Timeout :", s.info.String())
				v.w.Close(errors.New("write timeout"))
				continue
			}
			n++
		}

	}
	return
}

func (s *Stream) ExecPushDone(key string) {
	execList := configure.GetExecPushDone()

	for _, execItem := range execList {
		cmdString := fmt.Sprintf("%s -k %s", execItem, key)
		go func(cmdString string) {
			log.Info("ExecPushDone:", cmdString)
			cmd := exec.Command("/bin/sh", "-c", cmdString)
			_, err := cmd.Output()
			if err != nil {
				log.Info("Excute error:", err)
			}
		}(cmdString)
	}
}
func (s *Stream) closeInter() {

	if s.r != nil {

		s.rtmpStream.SetPushPushing(s.r.Info().URL, false)

		is, projectId := s.rtmpStream.IsMixSrc(s.r.Info().URL)
		if is {

			//得到此房间混音地址
			_, mixUrl := s.rtmpStream.GetMixUrl(projectId)
			s.rtmpStream.StopMixffmpeg(projectId, mixUrl)
		}

		//停止发布者所启动的ffmpeg
		log.Infof("Stream closeInter Check FFMpeg ")
		if s.FFmpeg {

			log.Infof("Stream Kill Publisher Start FFMpeg")

			s.FFmpeg = false
			s.Stopffmpeg()
		}

		log.Infof("Stream closeInter Close Publisher: [%s]", s.r.Info().String())
		s.rtmpStream.GetStreams().Remove(s.r.Info().Key)

	}
	s.ExecPushDone(s.r.Info().Key)

	//删除观看者
	for item := range s.ws.IterBuffered() {
		v := item.Val.(*PackWriterCloser)
		if v.w != nil {
			if v.w.Info().IsInterval() {
				v.w.Close(errors.New("closed"))
				s.ws.Remove(item.Key)

				log.Infof("Stream closeInter Close Viewers [%v] And Remove \n", v.w.Info().String())
			}
		}
	}
}
