package rtmp

import (
	"av"
	"configure"
	"container/flv"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	_ "io"
	log "logging"
	"net"
	"net/url"
	"os/exec"
	"protocol/rtmp/core"
	_ "reflect"
	_ "strconv"
	"strings"
	"time"
	"utils/uid"
)

const (
	maxQueueNum           = 1024
	maxMixQueueNum        = 1024 //2 * 1024
	SAVE_STATICS_INTERVAL = 5000
)

var (
	readTimeout  = flag.Int("readTimeout", 10, "read time out")
	writeTimeout = flag.Int("writeTimeout", 10, "write time out")
)

type Client struct {
	handler av.Handler
	getter  av.GetWriter
}

func (c *Client) Dial(url string, method string) error {

	connClient := core.NewConnClient()
	if err := connClient.Start(url, method); err != nil {
		return err
	}

	if method == av.PUBLISH {

		//客户端推流(这里可以进行控制)
		exist := strings.Contains(url, "mixStream")
		if !exist {

			writer := NewVirWriter(connClient, false, nil, -1, -1, false)
			log.Infof("---->>>>client Dial method is av.PUBLISH NewVirWriter url=%s, method=%s", url, method)
			c.handler.HandleWriter(writer)

		} else {

			writer := NewVirWriter(connClient, false, nil, -1, -1, true)
			log.Infof("---->>>>client Dial method is av.PUBLISH NewVirWriter url=%s, method=%s", url, method)
			c.handler.HandleWriter(writer)
		}

	} else if method == av.PLAY {

		//客户端观看
		reader := NewVirReader(connClient)
		log.Infof("---->>>>client Dial method is av.PLAY NewVirReader url=%s, method=%s", url, method)
		c.handler.HandleReader(reader)

		if c.getter != nil {

			log.Infof("---->>>>client Dial method is av.PLAY getter != nil")
			writer := c.getter.GetWriter(reader.Info())
			c.handler.HandleWriter(writer)
		}
	}
	return nil
}

func (c *Client) GetHandle() av.Handler {
	return c.handler
}

type Server struct {
	handler av.Handler
	getter  av.GetWriter
}

func NewRtmpServer(h av.Handler, getter av.GetWriter) *Server {

	return &Server{
		handler: h,
		getter:  getter,
	}
}

func (s *Server) Serve(listener net.Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("rtmp serve panic: ", r)
		}
	}()

	for {
		var netconn net.Conn

		netconn, err = listener.Accept()
		if err != nil {
			return
		}

		conn := core.NewConn(netconn, 4*1024)
		go s.handleConn(conn)
	}
}

func (s *Server) ExecPush(key string) {

	execList := configure.GetExecPush()

	for _, execItem := range execList {

		cmdString := fmt.Sprintf("%s -k %s", execItem, key)

		go func(cmdString string) {
			log.Info("(s *Server) ExecPush:", cmdString)
			cmd := exec.Command("/bin/sh", "-c", cmdString)
			_, err := cmd.Output()
			if err != nil {
				log.Info("Excute error:", err)
			}
		}(cmdString)
	}
}

func getKey(conn StreamReadWriteCloser) string {

	_, _, URL, _ := conn.GetInfo()
	_url, err := url.Parse(URL)
	if err != nil {
		log.Info(err)
	}
	key := strings.TrimLeft(_url.Path, "/")
	return key
}

func (s *Server) Url2Md5(uri string) (string, string, error) {

	u, err := url.Parse(uri)
	if err != nil {
		return "", "", err
	}

	m, _ := url.ParseQuery(u.RawQuery)

	m, queryErr := url.ParseQuery(u.RawQuery)
	if queryErr != nil {
		return "", "", queryErr
	}

	if len(m) <= 0 {
		return "", "", errors.New("Not Found Params")
	}

	existV := false
	if m.Get("v") != "" {
		existV = true
	}

	if !existV {
		return "", "", errors.New("Not Found v Params")
	}

	// if len(m["v"] <= 0) {
	// 	return "", "", errors.New("Not Found v Params")
	// }

	md5String := m["v"][0]

	beginPos := strings.Index(uri, "/live")
	endPos := strings.Index(uri, "?v")
	if beginPos == -1 || endPos == -1 {

		return "", "", errors.New("v Params Error")
	}

	pathString := uri[beginPos:endPos]

	return pathString, md5String, nil
}

func (s *Server) md5V3(str string) string {
	w := md5.New()
	io.WriteString(w, str)
	md5str := fmt.Sprintf("%x", w.Sum(nil))
	return md5str
}

func (s *Server) handleConn(conn *core.Conn) error {

	log.Info("(s *Server) handleConn")

	if err := conn.HandshakeServer(); err != nil {
		conn.Close()
		return err
	}

	connServer := core.NewConnServer(conn)
	if err := connServer.ReadMsg(); err != nil {
		conn.Close()
		log.Error("(s *Server) handleConn read msg err:", err)
		return err
	}

	//得到推流信息;
	appname, name, url, remoteconn := connServer.GetInfo()
	log.Infof("---->>>> Server handleConn appname=%s, name=%s, url=%s, peerIP=%s", appname, name, url, remoteconn.RemoteAddr().String())

	//得到Rtmp流的管理对象
	rtmpStream := s.handler.(*RtmpStream)
	if rtmpStream == nil {

		log.Error("Get rtmp Stream information error")
		return errors.New("Get rtmp Stream information error")
	}

	//判断是否是混音流
	exist := strings.Contains(url, "mixStream")
	if !exist {

		/*不是混音流，走原有逻辑*/

		//判断推流点地址是否已经被允许
		err, pushStream := rtmpStream.FindPushStream(url)
		if err != nil {

			/*这里增加了对球机的判断*/
			pathString, md5String, uriErr := s.Url2Md5(url)
			if uriErr != nil {

				//有错误，直接断开连接
				conn.Close()
				log.Error("Not Found PushStream url=%s", url)
				return uriErr
			} else {

				//没有错误，需要分解MD5信息
				if md5String == "" {
					conn.Close()
					log.Error("Not Found MD5 From url=%s", url)
					return errors.New("Not Found MD5 From url")
				}

				//判断格式得到的md5和上传的是否一致
				mathMd5 := s.md5V3(pathString + "&secret=E3ULXDeId0xP6RZVGFgF7hbxPctRoDOz")
				if mathMd5 != md5String {

					conn.Close()
					log.Error("MD5 Param Failed From url=%s", url)
					return errors.New("MD5 Param Failed")
				}
			}

			//判断客户端是发布者还是观看者
			if connServer.IsPublisher() {

				//判断这个发布者的地址是否已经存在了推流
				exist := rtmpStream.PublisherExist(url)
				if exist {

					conn.Close()
					log.Error("Pushlisher Is Exists url=%s", url)
					return errors.New("Pushlisher Is Exists")
				}

				rtmpStream.SetPushPushing(url, true)

				//发布者
				reader := NewVirReader(connServer)
				s.handler.HandleReader(reader)
				log.Infof("---->>>> Server handleConn New Publisher: %s", reader.Info().String())
				s.ExecPush(reader.Info().Key)

			} else {

				//观看者
				projectId := 999999
				pushId := 999999
				writer := NewVirWriter(connServer, false, rtmpStream, projectId, pushId, false)
				s.handler.HandleWriter(writer)
				log.Infof("---->>>> Server handleConn New Watch: %s ", writer.Info().String())
			}

		} else {

			//查询项目ID和推流ID
			errProject, projectId, pushId := rtmpStream.GetProjectPushIdFromUrl(url)
			if errProject != nil {

				conn.Close()
				log.Error("Not Found projectId and pushId url=%s", url)
				return err
			}

			//判断客户端是发布者还是观看者
			if connServer.IsPublisher() {

				//
				rtmpStream.SetPushPushing(url, true)

				//发布者
				reader := NewVirReader(connServer)
				s.handler.HandleReader(reader)
				log.Infof("---->>>> Server handleConn New Publisher: %s", reader.Info().String())
				s.ExecPush(reader.Info().Key)

			} else {

				//观看者
				writer := NewVirWriter(connServer, pushStream.LimitAudio, rtmpStream, projectId, pushId, false)
				s.handler.HandleWriter(writer)
				log.Infof("---->>>> Server handleConn New Watch: %s ", writer.Info().String())
			}
		}

		return nil
	} else {

		/*混音流，走原有逻辑*/

		//判断推流点地址是否已经被允许
		err, liveRoom := rtmpStream.FindMixStream(url)
		if err != nil {
			conn.Close()
			log.Error("Not Found FindMixStream url=%s", url)
			return err
		}

		//判断客户端是发布者还是观看者
		if connServer.IsPublisher() {

			//发布者
			reader := NewVirReader(connServer)
			s.handler.HandleReader(reader)
			log.Infof("---->>>> Server handleConn New Publisher: %s", reader.Info().String())
			s.ExecPush(reader.Info().Key)

		} else {

			//观看者
			limitAudio := false
			writer := NewVirWriter(connServer, limitAudio, rtmpStream, liveRoom.ProjectId, 9999, true)
			s.handler.HandleWriter(writer)
			log.Infof("---->>>> Server handleConn New Watch: %s ", writer.Info().String())
		}

		return nil

	}

}

type GetInFo interface {

	//app string, name string, url string, *core.Conn
	GetInfo() (string, string, string, *core.Conn)
}

type StreamReadWriteCloser interface {
	GetInFo
	Close(error)
	Write(core.ChunkStream) error
	Read(c *core.ChunkStream) error
}

type StaticsBW struct {
	StreamId               uint32
	PeerIP                 string
	VideoDatainBytes       uint64
	LastVideoDatainBytes   uint64
	VideoSpeedInBytesperMS uint64

	AudioDatainBytes       uint64
	LastAudioDatainBytes   uint64
	AudioSpeedInBytesperMS uint64

	LastTimestamp int64
}

type VirWriter struct {
	Uid    string
	closed bool
	av.RWBaser
	conn        StreamReadWriteCloser
	packetQueue chan *av.Packet
	WriteBWInfo StaticsBW

	mixStream      bool
	projectId      int
	pushId         int
	rtmpStream     *RtmpStream
	limitAudio     bool //是否被限制语音
	lastWriteTimer time.Time
}

func NewVirWriter(conn StreamReadWriteCloser, forceAudio bool, rs *RtmpStream, projectId int, pushId int, mixStream bool) *VirWriter {

	log.Infof("NewVirWriter mixStream=%d projectId=%d pushID=%d", mixStream, projectId, pushId)

	var ret *VirWriter
	if mixStream {
		ret = &VirWriter{
			Uid:         uid.NewId(),
			conn:        conn,
			RWBaser:     av.NewRWBaser(time.Second * time.Duration(*writeTimeout)),
			packetQueue: make(chan *av.Packet, maxMixQueueNum),
			WriteBWInfo: StaticsBW{0, "", 0, 0, 0, 0, 0, 0, 0},
			rtmpStream:  rs,
			projectId:   projectId,
			pushId:      pushId,
			limitAudio:  forceAudio,
			mixStream:   mixStream,
			closed:      false,
		}
	} else {
		ret = &VirWriter{
			Uid:         uid.NewId(),
			conn:        conn,
			RWBaser:     av.NewRWBaser(time.Second * time.Duration(*writeTimeout)),
			packetQueue: make(chan *av.Packet, maxQueueNum),
			WriteBWInfo: StaticsBW{0, "", 0, 0, 0, 0, 0, 0, 0},
			rtmpStream:  rs,
			projectId:   projectId,
			pushId:      pushId,
			limitAudio:  forceAudio,
			mixStream:   mixStream,
			closed:      false,
		}
	}

	go ret.CheckAudio()
	go ret.Check()
	go func() {
		err := ret.SendPacket()
		if err != nil {
			log.Error(err)
		}
	}()
	return ret
}

//检测音频控制
func (v *VirWriter) CheckAudio() {

	for {

		time.Sleep(time.Duration(500) * time.Millisecond)

		if !v.mixStream {

			err, limitAudio := v.rtmpStream.GetLimitAudioFromPushId(v.projectId, v.pushId)
			if err == nil {

				if v.limitAudio != limitAudio {

					// log.Info("CheckAudio Audio Changle %d projectId=%d pushID=%d", audio:=limitAudio ? 0 : 1, v.projectId, v.pushId)
					log.Info("CheckAudio Audio Changle", "Audio=", limitAudio, "projectId=", v.projectId, "pushID=", v.pushId)
					v.limitAudio = limitAudio
				}
			}
		}
	}
}

func (v *VirWriter) SaveStatics(streamid uint32, length uint64, isVideoFlag bool) {

	nowInMS := int64(time.Now().UnixNano() / 1e6)

	_, _, _, conn := v.conn.GetInfo()
	v.WriteBWInfo.PeerIP = conn.RemoteAddr().String()
	v.WriteBWInfo.StreamId = streamid
	if isVideoFlag {
		v.WriteBWInfo.VideoDatainBytes = v.WriteBWInfo.VideoDatainBytes + length
	} else {
		v.WriteBWInfo.AudioDatainBytes = v.WriteBWInfo.AudioDatainBytes + length
	}

	if v.WriteBWInfo.LastTimestamp == 0 {
		v.WriteBWInfo.LastTimestamp = nowInMS
	} else if (nowInMS - v.WriteBWInfo.LastTimestamp) >= SAVE_STATICS_INTERVAL {
		diffTimestamp := (nowInMS - v.WriteBWInfo.LastTimestamp) / 1000

		v.WriteBWInfo.VideoSpeedInBytesperMS = (v.WriteBWInfo.VideoDatainBytes - v.WriteBWInfo.LastVideoDatainBytes) * 8 / uint64(diffTimestamp) / 1000
		v.WriteBWInfo.AudioSpeedInBytesperMS = (v.WriteBWInfo.AudioDatainBytes - v.WriteBWInfo.LastAudioDatainBytes) * 8 / uint64(diffTimestamp) / 1000

		v.WriteBWInfo.LastVideoDatainBytes = v.WriteBWInfo.VideoDatainBytes
		v.WriteBWInfo.LastAudioDatainBytes = v.WriteBWInfo.AudioDatainBytes
		v.WriteBWInfo.LastTimestamp = nowInMS
	}
}

func (v *VirWriter) Check() {

	var c core.ChunkStream
	for {
		if err := v.conn.Read(&c); err != nil {
			v.Close(err)
			return
		}
	}
}

func (v *VirWriter) DropPacket(pktQue chan *av.Packet, info av.Info) {

	log.Infof("[%v] packet queue max!!!", info)

	v.lastWriteTimer = time.Now()

	if v.mixStream {

		for i := 0; i < maxMixQueueNum-84; i++ {

			tmpPkt, ok := <-pktQue

			//对于音频数据包进行判断
			if ok && tmpPkt.IsAudio {

				//判断是否丢弃音频数据包(当数据包数量达到一定值时，将丢弃音频数据包)
				if len(pktQue) > maxMixQueueNum-2 {
					log.Info("drop audio pkt")
					<-pktQue
				} else {
					pktQue <- tmpPkt
				}
			}

			//对于视频进行数据包判断
			if ok && tmpPkt.IsVideo {

				videoPkt, ok := tmpPkt.Header.(av.VideoPacketHeader)

				//不对SPS和I帧进行丢弃
				if ok && (videoPkt.IsSeq() || videoPkt.IsKeyFrame()) {
					pktQue <- tmpPkt
				}

				//如果视频缓冲太多，则丢弃视频数据（不管是sps或者关键帧）
				if len(pktQue) > maxMixQueueNum-10 {
					log.Info("drop video pkt")
					<-pktQue
				}
			}

		}
		log.Info("packet queue len: ", len(pktQue))

	} else {

		for i := 0; i < maxQueueNum-84; i++ {

			tmpPkt, ok := <-pktQue

			//对于音频数据包进行判断
			if ok && tmpPkt.IsAudio {

				//判断是否丢弃音频数据包(当数据包数量达到一定值时，将丢弃音频数据包)
				if len(pktQue) > maxQueueNum-2 {
					log.Info("drop audio pkt")
					<-pktQue
				} else {
					pktQue <- tmpPkt
				}
			}

			//对于视频进行数据包判断
			if ok && tmpPkt.IsVideo {

				videoPkt, ok := tmpPkt.Header.(av.VideoPacketHeader)

				//不对SPS和I帧进行丢弃
				if ok && (videoPkt.IsSeq() || videoPkt.IsKeyFrame()) {
					pktQue <- tmpPkt
				}

				//如果视频缓冲太多，则丢弃视频数据（不管是sps或者关键帧）
				if len(pktQue) > maxQueueNum-10 {
					log.Info("drop video pkt")
					<-pktQue
				}
			}

		}
		log.Info("packet queue len: ", len(pktQue))
	}
}

//此处需要注意
//Read为读取推流上来的数据
//Write为向客户端推送数据
func (v *VirWriter) Write(p *av.Packet) (err error) {

	err = nil

	if v.closed {
		err = errors.New("VirWriter closed")
		return
	}

	defer func() {
		if e := recover(); e != nil {
			errString := fmt.Sprintf("VirWriter has already been closed:%v", e)
			err = errors.New(errString)
		}
	}()

	//这里可以增加判断，判断Packet包是视频的还是音频的，
	//从而判断是否需要发送给观看用户.
	//以下代码只是将视频进行传输给观看用户

	if v.mixStream {

		//	原来的代码
		if len(v.packetQueue) >= maxMixQueueNum-24 {
			v.DropPacket(v.packetQueue, v.Info())
		} else {
			v.packetQueue <- p
		}

		// if v.limitAudio {

		// 	//被强制不能发布音频
		// 	if p.IsVideo {

		// 		if len(v.packetQueue) >= maxMixQueueNum-24 {
		// 			v.DropPacket(v.packetQueue, v.Info())
		// 		} else {
		// 			v.packetQueue <- p
		// 		}

		// 	}
		// } else {

		// 	//	原来的代码
		// 	if len(v.packetQueue) >= maxMixQueueNum-24 {
		// 		v.DropPacket(v.packetQueue, v.Info())
		// 	} else {
		// 		v.packetQueue <- p
		// 	}
		// }

	} else {

		//	原来的代码
		if len(v.packetQueue) >= maxQueueNum-24 {
			v.DropPacket(v.packetQueue, v.Info())
		} else {
			v.packetQueue <- p
		}

		// if v.limitAudio {

		// 	//被强制不能发布音频
		// 	if p.IsVideo {

		// 		if len(v.packetQueue) >= maxQueueNum-24 {
		// 			v.DropPacket(v.packetQueue, v.Info())
		// 		} else {
		// 			v.packetQueue <- p
		// 		}

		// 	}
		// } else {

		// 	//	原来的代码
		// 	if len(v.packetQueue) >= maxQueueNum-24 {
		// 		v.DropPacket(v.packetQueue, v.Info())
		// 	} else {
		// 		v.packetQueue <- p
		// 	}
		// }

	}

	return
}

func (v *VirWriter) SendPacket() error {

	var cs core.ChunkStream
	for {

		p, ok := <-v.packetQueue

		if ok {
			cs.Data = p.Data
			cs.Length = uint32(len(p.Data))
			cs.StreamID = p.StreamID
			cs.Timestamp = p.TimeStamp
			cs.Timestamp += v.BaseTimeStamp()

			if p.IsVideo {
				cs.TypeID = av.TAG_VIDEO
			} else {
				if p.IsMetadata {
					cs.TypeID = av.TAG_SCRIPTDATAAMF0
				} else {
					cs.TypeID = av.TAG_AUDIO
				}
			}

			v.SaveStatics(p.StreamID, uint64(cs.Length), p.IsVideo)
			v.SetPreTime()
			v.RecTimeStamp(cs.Timestamp, cs.TypeID)

			err := v.conn.Write(cs)
			if err != nil {

				log.Info("(v *VirWriter) SendPacket v.closed = true")
				v.closed = true
				return err
			}

		} else {

			return errors.New("closed")
		}

	}
	return nil
}

func (v *VirWriter) Info() (ret av.Info) {

	ret.UID = v.Uid
	_, _, URL, _ := v.conn.GetInfo()
	ret.URL = URL
	_url, err := url.Parse(URL)
	if err != nil {
		log.Info(err)
	}
	ret.Key = strings.TrimLeft(_url.Path, "/")
	ret.Inter = true
	return
}

func (v *VirWriter) Close(err error) {

	log.Info("VirWriter.player ", v.Info(), "closed: "+err.Error())
	if !v.closed {
		close(v.packetQueue)
	}
	v.closed = true
	v.conn.Close(err)
}

type VirReader struct {
	Uid string
	av.RWBaser
	demuxer         *flv.Demuxer
	conn            StreamReadWriteCloser
	ReadBWInfo      StaticsBW
	limitAudio      bool //是否被限制语音
	lastReaderTimer time.Time
	closed          bool
}

func NewVirReader(conn StreamReadWriteCloser) *VirReader {
	return &VirReader{
		Uid:        uid.NewId(),
		conn:       conn,
		RWBaser:    av.NewRWBaser(time.Second * time.Duration(*writeTimeout)),
		demuxer:    flv.NewDemuxer(),
		ReadBWInfo: StaticsBW{0, "", 0, 0, 0, 0, 0, 0, 0},
		closed:     false,
	}
}

func (v *VirReader) SaveStatics(streamid uint32, length uint64, isVideoFlag bool) {
	nowInMS := int64(time.Now().UnixNano() / 1e6)

	_, _, _, conn := v.conn.GetInfo()
	v.ReadBWInfo.StreamId = streamid
	v.ReadBWInfo.PeerIP = conn.RemoteAddr().String()
	if isVideoFlag {
		v.ReadBWInfo.VideoDatainBytes = v.ReadBWInfo.VideoDatainBytes + length
	} else {
		v.ReadBWInfo.AudioDatainBytes = v.ReadBWInfo.AudioDatainBytes + length
	}

	if v.ReadBWInfo.LastTimestamp == 0 {
		v.ReadBWInfo.LastTimestamp = nowInMS
	} else if (nowInMS - v.ReadBWInfo.LastTimestamp) >= SAVE_STATICS_INTERVAL {
		diffTimestamp := (nowInMS - v.ReadBWInfo.LastTimestamp) / 1000

		//log.Printf("now=%d, last=%d, diff=%d", nowInMS, v.ReadBWInfo.LastTimestamp, diffTimestamp)
		v.ReadBWInfo.VideoSpeedInBytesperMS = (v.ReadBWInfo.VideoDatainBytes - v.ReadBWInfo.LastVideoDatainBytes) * 8 / uint64(diffTimestamp) / 1000
		v.ReadBWInfo.AudioSpeedInBytesperMS = (v.ReadBWInfo.AudioDatainBytes - v.ReadBWInfo.LastAudioDatainBytes) * 8 / uint64(diffTimestamp) / 1000

		v.ReadBWInfo.LastVideoDatainBytes = v.ReadBWInfo.VideoDatainBytes
		v.ReadBWInfo.LastAudioDatainBytes = v.ReadBWInfo.AudioDatainBytes
		v.ReadBWInfo.LastTimestamp = nowInMS
	}
}

func (v *VirReader) Read(p *av.Packet) (err error) {

	defer func() {
		if r := recover(); r != nil {
			log.Error("rtmp read packet panic: ", r)
		}
	}()

	v.SetPreTime()
	cs := &core.ChunkStream{}

	for {
		err = v.conn.Read(cs)
		if err != nil {
			return err
		}
		if cs.TypeID == av.TAG_AUDIO ||
			cs.TypeID == av.TAG_VIDEO ||
			cs.TypeID == av.TAG_SCRIPTDATAAMF0 ||
			cs.TypeID == av.TAG_SCRIPTDATAAMF3 {
			break
		}
	}
	v.lastReaderTimer = time.Now()

	p.IsAudio = cs.TypeID == av.TAG_AUDIO
	p.IsVideo = cs.TypeID == av.TAG_VIDEO
	p.IsMetadata = cs.TypeID == av.TAG_SCRIPTDATAAMF0 || cs.TypeID == av.TAG_SCRIPTDATAAMF3
	p.StreamID = cs.StreamID
	p.Data = cs.Data
	p.TimeStamp = cs.Timestamp

	v.SaveStatics(p.StreamID, uint64(len(p.Data)), p.IsVideo)
	v.demuxer.DemuxH(p)

	return err
}

func (v *VirReader) Info() (ret av.Info) {
	ret.UID = v.Uid
	_, _, URL, _ := v.conn.GetInfo()
	ret.URL = URL
	_url, err := url.Parse(URL)
	if err != nil {
		log.Error(err)
	}
	ret.Key = strings.TrimLeft(_url.Path, "/")
	return
}

func (v *VirReader) Close(err error) {
	log.Info("VirReader.publisher ", v.Info(), "closed: "+err.Error())

	v.closed = true
	v.conn.Close(err)

}
