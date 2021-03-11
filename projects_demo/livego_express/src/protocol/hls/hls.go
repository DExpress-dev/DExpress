package hls

import (
	"av"
	"concurrent-map"
	"errors"
	"fmt"
	log "logging"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	duration = 3000
)

var (
	ErrNoPublisher         = errors.New("No publisher")
	ErrInvalidReq          = errors.New("invalid req url path")
	ErrNoSupportVideoCodec = errors.New("no support video codec")
	ErrNoSupportAudioCodec = errors.New("no support audio codec")
)

var crossdomainxml = []byte(`<?xml version="1.0" ?>
<cross-domain-policy>
	<allow-access-from domain="*" />
	<allow-http-request-headers-from domain="*" headers="*"/>
</cross-domain-policy>`)

type Server struct {
	listener net.Listener
	conns    cmap.ConcurrentMap
}

func NewServer() *Server {
	ret := &Server{
		conns: cmap.New(),
	}
	go ret.checkStop()
	return ret
}

func (server *Server) Serve(listener net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		server.handle(w, r)
	})
	server.listener = listener
	http.Serve(listener, mux)
	return nil
}

func (server *Server) GetListener() net.Listener {
	return server.listener
}

func (server *Server) GetWriter(info av.Info) av.WriteCloser {
	var s *Source
	ok := server.conns.Has(info.Key)
	if !ok {
		log.Info("new hls source")
		s = NewSource(info)
		server.conns.Set(info.Key, s)
	} else {
		v, _ := server.conns.Get(info.Key)
		s = v.(*Source)
	}
	return s
}

func (server *Server) getConn(key string) *Source {
	v, ok := server.conns.Get(key)
	if !ok {
		return nil
	}
	return v.(*Source)
}

func (server *Server) checkStop() {
	for {
		<-time.After(5 * time.Second)
		for item := range server.conns.IterBuffered() {
			v := item.Val.(*Source)
			if !v.Alive() {
				log.Info("check stop and remove: ", v.Info())
				server.conns.Remove(item.Key)
			}
		}
	}
}

func (server *Server) handle(w http.ResponseWriter, r *http.Request) {
	if path.Base(r.URL.Path) == "crossdomain.xml" {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(crossdomainxml)
		return
	}
	switch path.Ext(r.URL.Path) {
	case ".m3u8":
		key, _ := server.parseM3u8(r.URL.Path)
		conn := server.getConn(key)
		if conn == nil {
			//log.Error("m3u8 url", r.URL.Path, "key", key, "connection do not exist.")
			http.Error(w, ErrNoPublisher.Error(), http.StatusForbidden)
			return
		}
		tsCache := conn.GetCacheInc()
		if tsCache == nil {
			//log.Error("url", r.URL.Path, "key", key, "has no tsCache")
			http.Error(w, ErrNoPublisher.Error(), http.StatusForbidden)
			return
		}
		body, err := tsCache.GenM3U8PlayList()
		if err != nil {
			log.Error("GenM3U8PlayList error: ", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Write(body)
	case ".ts":
		key, _ := server.parseTs(r.URL.Path)
		conn := server.getConn(key)
		if conn == nil {
			log.Error(".ts url", r.URL.Path, "key", key, "connection do not exist.")
			http.Error(w, ErrNoPublisher.Error(), http.StatusForbidden)
			return
		}
		tsCache := conn.GetCacheInc()
		item, err := tsCache.GetItem(r.URL.Path)
		if err != nil {
			log.Error("GetItem error: ", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "video/mp2ts")
		w.Header().Set("Content-Length", strconv.Itoa(len(item.Data)))
		w.Write(item.Data)
	}
}

func (server *Server) parseM3u8(pathstr string) (key string, err error) {
	pathstr = strings.ToLower(pathstr)
	pathstr = strings.TrimLeft(pathstr, "/")

	index := strings.LastIndex(pathstr, ".m3u8")
	if index < 0 {
		errString := fmt.Sprintf("path(%s) has no .m3u8", pathstr)
		return "", errors.New(errString)
	}
	key = pathstr[0:index]

	return
}

func (server *Server) parseTs(pathstr string) (key string, err error) {
	pathstr = strings.ToLower(pathstr)
	pathstr = strings.TrimLeft(pathstr, "/")

	index := strings.LastIndex(pathstr, ".ts")
	if index < 0 {
		errString := fmt.Sprintf("path(%s) has no .ts", pathstr)
		return "", errors.New(errString)
	}
	pathstr = pathstr[0:index]

	index = strings.LastIndex(pathstr, "/")
	if index < 0 {
		errString := fmt.Sprintf("path(%s) has no /", pathstr)
		return "", errors.New(errString)
	}

	key = pathstr[0:index]
	return
}
