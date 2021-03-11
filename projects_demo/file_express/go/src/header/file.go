package header

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

//文件是否存在
func FileExists(path string) (bool, error) {

	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//文件夹是否存在
func DirExists(path string) (bool, error) {

	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

//得到目录下所有文件（递归）
func GetDirAllFile(path string) []string {

	var fileArray []string

	filepath.Walk(path, func(fpath string, info os.FileInfo, err error) error {

		if !info.IsDir() {
			path := fpath
			fileArray = append(fileArray, path)
		}
		return nil
	})
	return fileArray
}

func GetFileCreateTime(path string) int64 {

	osType := runtime.GOOS
	finfo, _ := os.Stat(path)
	if osType == "linux" {

		stat_t := finfo.Sys().(*syscall.Stat_t)
		tCreate := int64(stat_t.Ctim.Sec)
		return tCreate
	}
	return time.Now().Unix()
}

//得到目录下所有文件，并按照创建时间排序
func GetDirAllFileSortCreateTime(path string) []string {

	//得到所有文件
	fileArray := GetDirAllFile(path)

	//得到所有文件的创建时间
	var timeArray []int64
	for _, v := range fileArray {

		timeArray = append(timeArray, GetFileCreateTime(v))
	}

	//根据创建时间进行排序
	for i := 0; i < len(timeArray)-1; i++ {

		//遍历i位以后的所有元素，如果比i位元素小，就和i位元素互换位置
		for j := i + 1; j < len(timeArray); j++ {

			if timeArray[j] < timeArray[i] {

				timeArray[i], timeArray[j] = timeArray[j], timeArray[i]
				fileArray[i], fileArray[j] = fileArray[j], fileArray[i]
			}
		}
	}

	return fileArray
}
