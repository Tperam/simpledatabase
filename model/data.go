package model

import (
	"fmt"
	"strconv"
	"strings"
)

// Data 存储数据结构
type Data struct {
	Num  int
	Name string
	Sex  string
}

// Serialize 从字符串转换成数据
func (d *Data) Serialize() string {
	return fmt.Sprintf("%d,%s,%s\n", d.Num, d.Name, d.Sex)
}

// UnSerialize 从数据转换成字符串
func UnSerialize(line string) *Data {
	d := &Data{}
	strArr := strings.Split(line, ",")
	d.Num, _ = strconv.Atoi(strArr[0])
	d.Name = strArr[1]
	d.Sex = strArr[2][:len(strArr[2])-1]
	return d
}
