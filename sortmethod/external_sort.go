package sortmethod

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"simpledatabase/model"
	"strconv"
	"sync"
	"time"
)

// ExternalSortInfo 外部排序信息
type ExternalSortInfo struct {
	// 开启最大 io 数量
	// 几路归并
	MaxIONum uint8

	// 需要归并排序的文件夹
	SrcDir string

	// 缓存文件路径
	TmpDir string

	// 最终输出文件的文件
	TargetFile string

	// 缓冲数据长度
	DataCacheLen int

	//
	wg *sync.WaitGroup

	// 是否需要下一次合并
	isNeedNextMerge bool
}

// NewExternalSortInfo 创建一个ExternalSortInfo，附带默认配置
func NewExternalSortInfo() *ExternalSortInfo {
	return &ExternalSortInfo{
		MaxIONum:     9,
		SrcDir:       "./data",
		TmpDir:       "./tmp",
		TargetFile:   "./data/finalData.dat",
		DataCacheLen: 2,
	}
}

// Run 开始外排序
func (esi *ExternalSortInfo) Run(re *regexp.Regexp) {
	fiArr, _ := ioutil.ReadDir(esi.SrcDir)
	// 控制开启多少个io
	var fileNameArr []string
	for _, fi := range fiArr {
		if re.MatchString(fi.Name()) {
			fileNameArr = append(fileNameArr, path.Join(esi.SrcDir, fi.Name()))
		}
	}
	isNeedNextMerge, fileNameArr := esi.singleFileMerge(fileNameArr)
	for isNeedNextMerge {
		fmt.Println()
		isNeedNextMerge, fileNameArr = esi.singleFileMerge(fileNameArr)
	}
}

// readData 读取数据
// 1. 保证从`io`中读取的同时，也在进行反序列化操作（将等待io时间利用到反序列化操作的时间上）
// 2. 为每个`io`设置缓存，当data[min]被取出时，立刻有数据填充进去
func (esi *ExternalSortInfo) singleFileMerge(fileNameArr []string) (bool, []string) {
	var outputFileNameArr []string
	if uint8(len(fileNameArr)) < esi.MaxIONum {
		esi.isNeedNextMerge = false
	}
	for i := 0; i < len(fileNameArr); {
		var fsArr []*os.File
		var unserializeData []chan *model.Data
		// 读取数据方式
		for j := uint8(0); j < esi.MaxIONum && i < len(fileNameArr); j++ {
			fsArr[j], _ = os.OpenFile(fileNameArr[i], os.O_RDONLY, 0666)
			serializeData := make(chan string, esi.DataCacheLen)
			unserializeData = append(unserializeData, make(chan *model.Data, esi.DataCacheLen))
			esi.wg.Add(2)
			// 创建一个 bufio.Reader
			// 创建一个带缓存的 channel
			go esi.readData(bufio.NewReader(fsArr[j]), serializeData)
			go esi.unserializeData(serializeData, unserializeData[j])
			i++
		}
		// 用于存放从io中取出的数据
		var data []*model.Data
		// 有效下标
		var availableDataIndex []uint8
		// 读取一遍
		for j := uint8(0); j < uint8(len(unserializeData)); j++ {
			singleData, ok := <-unserializeData[j]
			if !ok {
				continue
			}
			data[j] = singleData
			availableDataIndex = append(availableDataIndex, j)
		}
		var outputFile *os.File
		// 开始排序，并写出数据
		if esi.isNeedNextMerge {
			// 使用 time.Now().UnixNano() 假设生成唯一数
			id := strconv.FormatInt(time.Now().UnixNano(), 10)

			outputFileName := path.Join(esi.TmpDir, "dataExternal"+id+".txt")
			outputFileNameArr = append(outputFileNameArr, outputFileName)
			fi, err := os.OpenFile(outputFileName, os.O_WRONLY, 0666)
			if err != nil {
				fi, _ = os.Create(outputFileName)
			}
			outputFile = fi
		} else {
			// 如果当前文件数小于最大io限制，则直接输出完整数据
			fi, err := os.OpenFile(esi.TargetFile, os.O_WRONLY, 0666)
			if err != nil {
				fi, _ = os.Create(esi.TargetFile)
			}
			outputFile = fi
		}

		bw := bufio.NewWriter(outputFile)
		for len(availableDataIndex) > 0 {
			// 记录 数组有效索引
			min := 0
			for j := 1; j < len(availableDataIndex); j++ {
				if data[availableDataIndex[min]].Num > data[availableDataIndex[j]].Num {
					min = j
				}
			}
			bw.WriteString(data[availableDataIndex[min]].Serialize())
			// 从channel中取值替换当前映射数组的值
			singleData, ok := <-unserializeData[availableDataIndex[min]]
			// 当前设计，当无法获取出数据之后。我们将有效数组下标中min置为空
			if !ok {
				// 无法取出数据
				// 删除当前下标
				availableDataIndex = append(availableDataIndex[:min], availableDataIndex[min+1:]...)
			} else {

				data[availableDataIndex[min]] = singleData
			}

		}
		// 等待执行完成
		esi.wg.Wait()
		bw.Flush()
		outputFile.Close()
		// 关闭文件io 防止泄露
		for _, fs := range fsArr {
			fs.Close()
		}
	}
	return esi.isNeedNextMerge, fileNameArr
}

// readData 读取数据
func (esi *ExternalSortInfo) readData(br *bufio.Reader, data chan<- string) {
	line, err := br.ReadString('\n')
	if err == io.EOF {
		close(data)
	}
	data <- line

	esi.wg.Done()
}

func (esi *ExternalSortInfo) unserializeData(serializeData <-chan string, unserializeData chan<- *model.Data) {
	for x := range serializeData {
		unserializeData <- model.UnSerialize(x)
	}
	close(unserializeData)
	esi.wg.Done()
}