package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"regexp"
	"simpledatabase/sortmethod"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	isi := sortmethod.NewInnerSortInfo()
	isi.SetMaxMemorySize("1GB")
	isi.TargetDir = "./tempdata/"
	re, _ := regexp.Compile("^data\\d+\\.txt$")
	isi.Run("./", re)
}
