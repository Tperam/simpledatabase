package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"sync"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	var dataLen int64 = 1000000000
	data := make(chan string, 1000)
	var wg sync.WaitGroup

	wg.Add(1)
	go CreateData(&wg, data, dataLen)

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go SaveDataToFile(&wg, data, "./data/data"+strconv.Itoa(i)+".txt")
	}

	wg.Wait()
}

// CreateData 管理创建数据
func CreateData(wg *sync.WaitGroup, data chan<- string, length int64) {
	// 开启两个生产线程
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	wg.Add(1)
	go produceData(cancel1, wg, data, length/2)
	wg.Add(1)
	go produceData(cancel2, wg, data, length/2)

	select {
	case <-ctx1.Done():
		<-ctx2.Done()
		close(data)
	case <-ctx2.Done():
		<-ctx1.Done()
		close(data)
	}
	// 使用 context控制
	wg.Done()
}

// produceData 创建数据
func produceData(cancel context.CancelFunc, wg *sync.WaitGroup, data chan<- string, length int64) {
	var i int64
	for i = 0; i < length; i++ {
		data <- fmt.Sprintf("%d,%s,%s\n", rand.Int(), getName(), getSex())
	}
	cancel()
	wg.Done()
}

func getName() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, 20)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
func getSex() string {
	genderNum := rand.Intn(2)
	if genderNum == 0 {
		return "male"
	}
	return "female"
}

// SaveDataToFile 保存数据到文件
func SaveDataToFile(wg *sync.WaitGroup, data <-chan string, filename string) {
	file, err := os.OpenFile(filename, os.O_WRONLY, 0666)
	if err != nil {
		file, _ = os.Create(filename)
	}
	bw := bufio.NewWriter(file)

	for x := range data {
		bw.WriteString(x)
	}
	bw.Flush()
	wg.Done()
}
