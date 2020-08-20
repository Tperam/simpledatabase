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

	esi := sortmethod.NewExternalSortInfo()
	esi.SrcDir = "./data"
	esi.TmpDir = "./tmp"
	esi.TargetFile = "./finaldata.dat"
	re, _ := regexp.Compile("data\\d+\\.txt$")
	esi.Run(re)
}
