package main

import (
	"fmt"
	"mr/mapreduce"
	"os"
	"strings"
	"strconv"
	"unicode"
)

// The mapping function is called once for each piece of the input.
// In this framework, the key is the name of the file that is being processed,
// and the value is the file's contents. The return value should be a slice of
// key/value pairs, each represented by a mapreduce.KeyValue.
func mapF(document string, value string) (res []mapreduce.KeyValue) {
	// TODO:
	//create a map containing the keys i.e. words and their respective counts
	wc := make(map[string] int)
	
	//function used to split words accordingly
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	
	//split file contents into words and update the count of each unique word in the map
	for _, word := range strings.FieldsFunc(value, f) {
		wc[word]++ 	//increment the count of each unique word
	}

	//store the results of mapF in this variable containing a slice of keyValue pairs
	var result[] mapreduce.KeyValue
	
	//create a slice of all Key-Value pairs and append them to result
	for key, value := range wc {
		var temp mapreduce.KeyValue
		temp.Key = key
		temp.Value = strconv.Itoa(value) 
		result = append(result, temp)
	}
	
	return result
}

// The reduce function is called once for each key generated by Map, with a
// list of that key's string value (merged across all inputs). The return value
// should be a single output value for that key.
func reduceF(key string, values []string) string {
	// TODO:
	//keep track of the total count of key
	var totalCount int
	
	//add counts from different files
	for _, value := range values {
		intValue, _ := strconv.Atoi(value)
		totalCount += intValue
	}
	
	//return the total count
	return strconv.Itoa(totalCount)
}

// Can be run in 3 ways:
// 1) Sequential (e.g., go run wc.go master sequential x1.txt .. xN.txt)
// 2) Master (e.g., go run wc.go master localhost:7777 x1.txt .. xN.txt)
// 3) Worker (e.g., go run wc.go worker localhost:7777 localhost:7778 &)
// known issue: both 2) and 3) work on OSX but not Ubuntu 16.04
func main() {
	if len(os.Args) < 4 {
		fmt.Printf("%s: see usage comments in file\n", os.Args[0])
	} else if os.Args[1] == "master" {
		var mr *mapreduce.Master
		if os.Args[2] == "sequential" {
			mr = mapreduce.Sequential("wcseq", os.Args[3:], 3, mapF, reduceF)
		} else {
			mr = mapreduce.Distributed("wcseq", os.Args[3:], 3, os.Args[2])
		}
		mr.Wait()
	} else {
		mapreduce.RunWorker(os.Args[2], os.Args[3], mapF, reduceF, 100)
	}
}