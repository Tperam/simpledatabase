package sortmethod

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"runtime"
	"simpledatabase/algorithm"
	"simpledatabase/model"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 内排序
//
//	- 读取数据后我们需要将其反序列化成`model.data`类型,我们将这一步拆分。使io一直在运行不被其他占用。
//  - 读取到一定内存量时，我们对其进行排序。再将数据写出到不同的文件
//  - 继续开始读取内存，直到将当前文件整个排序完成

// InnerSortInfo 内排序结构体
// 通过MaxMemorySize 管理内存
// 通过ReadData 获取数据
type InnerSortInfo struct {
	MaxMemorySize   uint64
	MemStats        *runtime.MemStats
	Lock            *sync.RWMutex
	TargetDir       string
	data            []*model.Data
	wg              *sync.WaitGroup
	finshAppendData bool
}

// 单位换算
const (
	B  = 1
	KB = 1024
	MB = KB * 1024
	GB = MB * 1024
	TB = GB * 1024
)

// NewInnerSortInfo 创建一个InnerSortInfo
func NewInnerSortInfo() *InnerSortInfo {
	return &InnerSortInfo{
		MaxMemorySize: 100 * MB,
		MemStats:      &runtime.MemStats{},
		TargetDir:     "./data",
		Lock:          &sync.RWMutex{},
		wg:            &sync.WaitGroup{},
	}
}

// Run 用于将所有数据内排序
func (isi *InnerSortInfo) Run(srcDir string, dataFileRe *regexp.Regexp) {
	serializeData := make(chan string, 1000)
	UnserializeData := make(chan *model.Data, 1000)
	isi.wg.Add(4)
	go isi.readData(serializeData, srcDir, dataFileRe)
	go isi.handleUnserialize(serializeData, UnserializeData)
	go isi.appendData(UnserializeData)
	go isi.memoryManage()
	isi.wg.Wait()
}

// SetMaxMemorySize 设置最大内存
// 当前自定义中 B = bytes
// eg "100KB" = 100 * 1024
func (isi *InnerSortInfo) SetMaxMemorySize(size string) bool {

	size = strings.ToLower(strings.TrimSpace(size))

	// 切割字符串，获取单位前的数字
	numStr := size[:len(size)-2]
	// 转换成int64类型
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		fmt.Println("输入有误,请匹配当前表达式 ^\\d+[TGMK]{0,1}B$")
		return false
	}
	// 切割单位
	unit := size[len(size)-2:]

	switch unit {
	case "tb":
		isi.MaxMemorySize = num * TB
	case "gb":
		isi.MaxMemorySize = num * GB
	case "mb":
		isi.MaxMemorySize = num * MB
	case "kb":
		isi.MaxMemorySize = num * KB
	case "b":
		isi.MaxMemorySize = num * B
	default:
		fmt.Println("使用了未定义的类型,请以 kb, mb, gb, tb, b结尾")
		return false
	}

	return true
}

// readData 读取data数据
// data 输出管道，用于往内存中添加数据
// srcDir 文件路径
func (isi *InnerSortInfo) readData(data chan<- string, srcDir string, re *regexp.Regexp) {
	fiArr, _ := ioutil.ReadDir(srcDir)
	for _, fi := range fiArr {

		if !re.MatchString(fi.Name()) {
			continue
		}

		file, err := os.OpenFile(path.Join(srcDir, fi.Name()), os.O_RDONLY, 0666)
		if err != nil {
			isi.wg.Done()
			return
		}

		br := bufio.NewReader(file)

		for {
			line, err := br.ReadString('\n')
			if err == io.EOF {
				break
			}
			data <- line
		}
	}
	close(data)
	isi.wg.Done()
}

// handleUnserialize 管理反序列化线程
func (isi *InnerSortInfo) handleUnserialize(data <-chan string, unserializeData chan<- *model.Data) {
	// 开启两个反序列化线程
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())

	isi.wg.Add(1)
	isi.unserializeData(cancel1, data, unserializeData)
	isi.wg.Add(1)
	isi.unserializeData(cancel2, data, unserializeData)

	<-ctx1.Done()
	<-ctx2.Done()
	close(unserializeData)
	isi.wg.Done()

}

// unserializeData 反序列化操作
func (isi *InnerSortInfo) unserializeData(cancel context.CancelFunc, data <-chan string, unserializeData chan<- *model.Data) {
	for x := range data {
		isi.Lock.RLock()
		unserializeData <- model.UnSerialize(x)
		isi.Lock.RUnlock()
	}
	cancel()
	isi.wg.Done()
}

// 添加data到结构体中
func (isi *InnerSortInfo) appendData(unserializeData <-chan *model.Data) {
	for x := range unserializeData {
		isi.data = append(isi.data, x)
	}
	isi.finshAppendData = true
	isi.wg.Done()
}

// saveCurrentDataToFile 将当前排序结果保存到文件
func (isi *InnerSortInfo) saveCurrentDataToFile(filename string) {

	length := len(isi.data)
	// 排序
	algorithm.QuickSort(isi.data, 0, length-1)
	// 排序后io写出
	file, err := os.OpenFile(filename, os.O_WRONLY, 0666)
	if err != nil {
		file, _ = os.Create(filename)
	}

	bw := bufio.NewWriter(file)

	for i := 0; i < length; i++ {
		bw.WriteString(isi.data[i].Serialize())
	}
	bw.Flush()

}

// memoryManage 内存管理
// 原理，每秒读取一次内存信息
// 在读取数据时进行判断。超出则停止读取，直到下一次可运行
func (isi *InnerSortInfo) memoryManage() {
	var i int
	for !isi.finshAppendData {
		runtime.ReadMemStats(isi.MemStats)

		if isi.MemStats.Alloc > isi.MaxMemorySize {
			isi.Lock.Lock()
			isi.saveCurrentDataToFile(path.Join(isi.TargetDir, "data"+strconv.Itoa(i)+".txt"))
			isi.data = make([]*model.Data, 0, len(isi.data))
			isi.Lock.Unlock()
			i++
		}

		time.Sleep(1 * time.Second)
	}
	isi.saveCurrentDataToFile(path.Join(isi.TargetDir, "data"+strconv.Itoa(i)+".txt"))
	isi.wg.Done()
}
