package main

import (
	"common/compile"
	cmap "concurrent-map"
	"configure"
	"flag"
	"fmt"
	log "logging"
	"net"
	"protocol/hls"
	"protocol/httpflv"
	"protocol/httpopera"
	"protocol/rtmp"
	"protocol/rtmp/rtmprelay"
	"time"
)

var (
	version    = "v2.2.2"
	baseConfig = flag.String(
		"cfgfile",     //参数
		"livego.json", //参数默认值
		"live configure filename")

	liveRoomConfig = flag.String(
		"room",      //参数
		"room.json", //参数默认值
		"直播房间配置信息")

	loglevel = flag.String("loglevel", "debug", "log level")
	logfile  = flag.String("logfile", "livego.log", "log file path")
)

var StaticPulMgr *rtmprelay.StaticPullManager
var checkVer *bool

func init() {

	checkVer = flag.Bool("V", false, "is ok")

	flag.Parse()
	log.SetOutputByName(*logfile)
	log.SetRotateByDay()
	log.SetLevelByString(*loglevel)
}

func PushStatic() {
	time.Sleep(time.Second * 5)
	var pullArray []configure.StaticPullInfo

	pullArray, bRet := configure.GetStaticPullList()

	log.Infof("startStaticPull: pullArray=%v, ret=%v", pullArray, bRet)
	if bRet && pullArray != nil && len(pullArray) > 0 {
		StaticPulMgr = rtmprelay.NewStaticPullManager(configure.GetListenPort(), pullArray)
		if StaticPulMgr != nil {
			StaticPulMgr.Start()
		}
	}
}

func stopStaticPull() {
	if StaticPulMgr != nil {
		StaticPulMgr.Stop()
	}
}

func startHls() (*hls.Server, net.Listener) {
	hlsaddr := fmt.Sprintf(":%d", configure.GetHlsPort())
	hlsListen, err := net.Listen("tcp", hlsaddr)
	if err != nil {
		log.Error(err)
	}

	hlsServer := hls.NewServer()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("HLS server panic: ", r)
			}
		}()
		log.Info("HLS listen On", hlsaddr)
		hlsServer.Serve(hlsListen)
	}()
	return hlsServer, hlsListen
}

func startRtmp(stream *rtmp.RtmpStream, hlsServer *hls.Server) {
	rtmpAddr := fmt.Sprintf(":%d", configure.GetListenPort())

	rtmpListen, err := net.Listen("tcp", rtmpAddr)
	if err != nil {
		log.Fatal(err)
	}

	var rtmpServer *rtmp.Server

	if hlsServer == nil {
		rtmpServer = rtmp.NewRtmpServer(stream, nil)
		log.Infof("hls server disable....")
	} else {
		rtmpServer = rtmp.NewRtmpServer(stream, hlsServer)
		log.Infof("hls server enable....")
	}

	defer func() {
		if r := recover(); r != nil {
			log.Error("RTMP server panic: ", r)
		}
	}()
	log.Info("RTMP Listen On", rtmpAddr)
	rtmpServer.Serve(rtmpListen)
}

func startHTTPFlv(stream *rtmp.RtmpStream, l net.Listener) net.Listener {
	var flvListen net.Listener
	var err error

	httpFlvAddr := fmt.Sprintf(":%d", configure.GetHttpFlvPort())
	if l == nil {
		log.Info("new flv listen...")
		flvListen, err = net.Listen("tcp", httpFlvAddr)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		flvListen = l
	}

	hdlServer := httpflv.NewServer(stream)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Fatal("HTTP-FLV server panic: ", r)
			}
		}()
		log.Info("HTTP-FLV listen On", httpFlvAddr)
		hdlServer.Serve(flvListen)
	}()
	return flvListen
}

func startHTTPOpera(stream *rtmp.RtmpStream, l net.Listener) net.Listener {
	var opListen net.Listener
	var err error

	operaAddr := fmt.Sprintf(":%d", configure.GetHttpOperPort())
	if l == nil {
		opListen, err = net.Listen("tcp", operaAddr)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		opListen = l
	}

	rtmpAddr := fmt.Sprintf(":%d", configure.GetListenPort())
	opServer := httpopera.NewServer(stream, rtmpAddr)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("HTTP-Operation server panic: ", r)
			}
		}()
		log.Info("HTTP-Operation listen On", operaAddr)
		opServer.Serve(opListen)
	}()

	return opListen
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("livego panic: ", r)
			time.Sleep(1 * time.Second)
		}
	}()

	if *checkVer {
		verStr := "ver: " + version + "\r\n"
		fmt.Println(verStr + compile.BuildTime())
		return
	}

	log.Info(`
     _     _            ____       
    | |   (_)_   _____ / ___| ___  
    | |   | \ \ / / _ \ |  _ / _ \ 
    | |___| |\ V /  __/ |_| | (_) |
    |_____|_| \_/ \___|\____|\___/ 
    
	Version: %s Map Hash Count: %d
	
	`, version, cmap.SHARD_COUNT)

	//加载基础配置信息
	log.Info("---->>>> LoadConfig")
	if err := configure.LoadConfig(*baseConfig); err != nil {

		log.Error("<<<< LoadConfig Error")
		return
	}

	//加载直播间配置信息
	log.Info("---->>>> LoadRtmpConfig")
	if err := configure.LoadRtmpConfig(*liveRoomConfig); err != nil {

		log.Error("<<<< LoadRtmpConfig Error")
		return
	}

	//创建Rtmp流管理
	log.Info("---->>>> NewRtmpStream")
	var hlsServer *hls.Server
	stream := rtmp.NewRtmpStream()

	//启动统计
	log.Info("---->>>> PushStatic")
	go PushStatic()
	defer stopStaticPull()

	log.Info("---->>>> Check Hls")
	if configure.IsHlsEnable() {

		log.Info("---->>>> Start Hls")
		hlsServer, _ = startHls()
	}

	log.Info("---->>>> Check Flv")
	if configure.IsHttpFlvEnable() {
		if configure.GetHlsPort() == configure.GetHttpFlvPort() {

			log.Error("Check Flv Failed Hls Port = Flv Port")
			return
		} else {

			log.Info("---->>>> Start Flv")
			startHTTPFlv(stream, nil)
		}
	}

	//启动Http操作
	log.Info("---->>>> Check Oper")
	if configure.IsHttpOperEnable() {

		if configure.IsHlsEnable() && configure.GetHlsPort() == configure.GetHttpOperPort() {

			log.Error("Check Oper Failed Oper Port = Hls Port")
			return
		} else if configure.IsHttpFlvEnable() && configure.GetHttpFlvPort() == configure.GetHttpOperPort() {

			log.Error("Check Oper Failed Oper Port = Flv Port")
		} else {

			log.Info("---->>>> Start Oper")
			startHTTPOpera(stream, nil)
		}
	}

	//启动Rtmp
	log.Info("---->>>> Check Rtmp")
	if configure.IsHlsEnable() {

		log.Info("---->>>> Start Rtmp")
		startRtmp(stream, hlsServer)
	} else {

		log.Info("---->>>> Start Rtmp")
		startRtmp(stream, nil)
	}
}
