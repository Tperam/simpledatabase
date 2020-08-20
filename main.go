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

	ish := sortmethod.NewInnerSortHandler()
	ish.SetMaxMemorySize("1GB")

	re, _ := regexp.Compile("^data\\d+\\.txt$")
	ish.Run("./", re)
}
