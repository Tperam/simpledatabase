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
	return d
}

// ToString 转换成string
func (d *Data) ToString() string {
	return fmt.Sprintf("%d,%s,%s", d.Num, d.Name, d.Sex)
}
