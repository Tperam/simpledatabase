package main

import (
	"regexp"
	"simpledatabase/sortmethod"
)

func main() {
	esi := sortmethod.NewExternalSortInfo()
	esi.SrcDir = "./data"
	esi.TmpDir = "./tmp"
	esi.TargetFile = "./data/finaldata.dat"
	re, _ := regexp.Compile("data\\d+\\.txt$")
	esi.Run(re)
}
