package model

import (
	"bytes"
	"fmt"
	"strconv"
	"unsafe"
)

// Data 存储数据结构
type Data struct {
	Num        int
	Name       string
	Sex        string
	sourceData []byte
}

const delim = ','

// Serialize 从字符串转换成数据
//
func (d *Data) Serialize() []byte {
	return d.sourceData
}

// UnSerialize 从数据转换成字符串
func UnSerialize(line []byte) *Data {

	d := &Data{}
	// 转换 string(num) -> num
	numIndex := bytes.IndexByte(line, delim)
	copyNumByte := make([]byte, numIndex)
	copy(copyNumByte, line)
	numStr := *(*string)(unsafe.Pointer(&copyNumByte))
	num, _ := strconv.Atoi(numStr)
	d.Num = num

	// 获取name和sex
	nameIndex := bytes.IndexByte(line[numIndex+1:], delim)
	nameByte := line[numIndex+1 : nameIndex]
	d.Name = *(*string)(unsafe.Pointer(&nameByte))

	sexByte := line[nameIndex+1:]
	d.Sex = *(*string)(unsafe.Pointer(&sexByte))
	d.sourceData = line
	return d
}

// func UnSerialize(line []byte) *Data {
// 	delimIndexArr := make([]int, 0, 2)
// 	for i, x := range line {
// 		if x == delim {
// 			delimIndexArr = append(delimIndexArr, i)
// 		}
// 	}

// 	d := &Data{}
// 	// 转换 string(num) -> num
// 	copyNumByte := make([]byte, delimIndexArr[0])
// 	copy(copyNumByte, line[:delimIndexArr[0]])
// 	numStr := *(*string)(unsafe.Pointer(&copyNumByte))
// 	num, _ := strconv.Atoi(numStr)
// 	d.Num = num
// 	// 获取name和sex
// 	nameByte := line[delimIndexArr[0]+1 : delimIndexArr[1]]
// 	d.Name = *(*string)(unsafe.Pointer(&nameByte))

// 	sexByte := line[delimIndexArr[1]+1:]
// 	d.Sex = *(*string)(unsafe.Pointer(&sexByte))
// 	d.sourceData = line
// 	return d
// }

// ToString 转换成string
func (d *Data) ToString() string {
	return fmt.Sprintf("%d,%s,%s", d.Num, d.Name, d.Sex)
}
