package rtmprelay

import (
	"configure"
	"errors"
	"fmt"
	log "logging"
	"reflect"
)

/*
   "static_pull":[{"type":"http-flv",
                   "source":"http://pull99.a8.com/live/1500365043587794.flv",
                   "app":"live",
                   "stream":"1500365043587794"},
                   {"type":"rtmp",
                   "source":"rtmp://pull99.a8.com/live/1500365043587794",
                   "app":"live",
                   "stream":"1500365043587794"}
                 ],
*/

const (
	RtmpType    = 1
	HttpflvType = 2
)

type staticPullInfo struct {
	Streamtype int
	SourceUrl  string
	Appname    string
	Streamname string
	PullObj    interface{}
}

type StaticPullManager struct {
	pullinfos   []staticPullInfo
	IsStartFlag bool
}

func NewStaticPullManager(listenPort int, pullinfos []configure.StaticPullInfo) *StaticPullManager {
	if pullinfos == nil || len(pullinfos) == 0 {
		return nil
	}

	pullmanager := &StaticPullManager{}

	for _, pullinfo := range pullinfos {
		staticpull := staticPullInfo{}

		staticpull.SourceUrl = pullinfo.Source
		staticpull.Appname = pullinfo.App
		staticpull.Streamname = pullinfo.Stream
		rtmpurl := fmt.Sprintf("rtmp://127.0.0.1:%d/%s/%s",
			listenPort, staticpull.Appname, staticpull.Streamname)
		if pullinfo.Type == "rtmp" {
			staticpull.Streamtype = RtmpType

			var liveId string
			var pushid string
			staticpull.PullObj = NewRtmpRelay(&staticpull.SourceUrl, &rtmpurl, &liveId, &pushid, -1)
		} else if pullinfo.Type == "http-flv" {
			staticpull.Streamtype = HttpflvType
			staticpull.PullObj = NewFlvPull(&staticpull.SourceUrl, &rtmpurl)
		} else {
			log.Errorf("not support type(%d)", pullinfo.Type)
			return nil
		}
		pullmanager.pullinfos = append(pullmanager.pullinfos, staticpull)
	}
	pullmanager.IsStartFlag = false
	return pullmanager
}

func (self *StaticPullManager) Start() error {
	if self.IsStartFlag {
		errString := fmt.Sprintf("StaticPullManager has already started")
		log.Error(errString)
		return errors.New(errString)
	}

	log.Infof("pullinfos=%v", self.pullinfos)
	for _, pullobj := range self.pullinfos {
		if obj, ok := pullobj.PullObj.(*FlvPull); ok {
			err := obj.Start()
			log.Infof("static flv pull start:%s, error=%v", pullobj.SourceUrl, err)
		} else if obj, ok := pullobj.PullObj.(*RtmpRelay); ok {
			err := obj.Start()
			log.Infof("static rtmp pull start:%s, error=%v", pullobj.SourceUrl, err)
		} else {
			log.Errorf("Unknow type=%v", reflect.TypeOf(pullobj.PullObj))
		}
	}
	self.IsStartFlag = true

	return nil
}

func (self *StaticPullManager) Stop() {
	if !self.IsStartFlag {
		log.Error("StaticPullManager has already stoped")
		return
	}
	for _, pullobj := range self.pullinfos {
		if obj, ok := pullobj.PullObj.(FlvPull); ok {
			obj.Stop()
			log.Infof("static flv pull stop:%s", pullobj.SourceUrl)
		} else if obj, ok := pullobj.PullObj.(RtmpRelay); ok {
			obj.Stop()
			log.Infof("static rtmp pull stop:%s", pullobj.SourceUrl)
		}
	}
	self.IsStartFlag = false
}
