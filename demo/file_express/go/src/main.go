package main

import (
	"business"
	"express"
	"flag"
	"fmt"
	"header"
	"os"
	"path/filepath"
	"time"

	log4plus "log4go"
)

//版本号
var (
	ver     string = "1.0.17"
	exeName string = "Load so Library"
	pidFile string = ""
)

type Flags struct {
	Help    bool
	Version bool
}

var web *business.WebManager

func (f *Flags) Init() {
	flag.BoolVar(&f.Help, "h", false, "help")
	flag.BoolVar(&f.Version, "V", false, "show version")
}

func (f *Flags) Check() (needReturn bool) {
	flag.Parse()

	if f.Help {
		flag.Usage()
		needReturn = true
	} else if f.Version {
		verString := exeName + " Version: " + ver + "\r\n"
		fmt.Println(verString)
		needReturn = true
	}

	return needReturn
}

var flags *Flags = &Flags{}

func init() {
	flags.Init()
	exeName = getExeName()
	pidFile = GetCurrentDirectory() + "/" + exeName + ".pid"
}

func getExeName() string {
	ret := ""
	ex, err := os.Executable()
	if err == nil {
		ret = filepath.Base(ex)
	}
	return ret
}

func setLog() {
	logJson := "log.json"
	set := false
	if bExist := business.PathExist(logJson); bExist {
		if err := log4plus.SetupLogWithConf(logJson); err == nil {
			set = true
		}
	}

	if !set {
		fileWriter := log4plus.NewFileWriter()
		exeName := getExeName()
		fileWriter.SetPathPattern("./log/" + exeName + "-%Y%M%D.log")
		log4plus.Register(fileWriter)
		log4plus.SetLevel(log4plus.DEBUG)
	}
}

func writePid() {
	SaveFile(fmt.Sprintf("%d", os.Getpid()), pidFile)
}

func main() {

	needReturn := flags.Check()
	if needReturn {
		return
	}

	setLog()
	defer log4plus.Close()
	log4plus.Info("%s Version=%s", getExeName(), ver)

	writePid()
	defer os.Remove(pidFile)
	defer log4plus.Close()

	fileArray1 := header.GetDirAllFile("/root/projects/dexpress/src")
	for _, v := range fileArray1 {

		log4plus.Info("file---->>>> %s", v)

	}

	express := express.New("log", "/root/projects/dexpress/src", "/root/projects/dexpress/src")
	express.SendFile("0.0.0.0", "127.0.0.1", 41002, false, "/root/udpping.py", "20201229/base")

	for {
		time.Sleep(time.Duration(1) * time.Second)
	}

}
