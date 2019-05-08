package main

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}

	var in = make(chan interface{}, MaxInputDataLen)
	for _, jobV := range jobs {
		wg.Add(1)
		var out = make(chan interface{}, MaxInputDataLen)

		go func(jobV job, in, out chan interface{}) {
			defer func() {
				close(out)
				wg.Done()
			}()
			jobV(in, out)
		}(jobV, in, out)

		in = out
	}

	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)

		dataString := fmt.Sprintf("%v", data)
		//fmt.Printf("%s SingleHash data %s\n", dataString, dataString)
		md5result := DataSignerMd5(dataString)
		//fmt.Printf("%s SingleHash md5(data) %s\n", dataString, md5result)

		go func(md5result, data string) {
			defer wg.Done()
			SingleHashCount(md5result, data, out)
		}(md5result, dataString)
	}
	wg.Wait()
}

func SingleHashCount(md5result, data string, out chan interface{}) {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	var crcmd5result, crc32result string
	go func () {
		defer wg.Done()
		crcmd5result = DataSignerCrc32(md5result)
		//fmt.Printf("%s SingleHash crc32(md5(data)) %s\n", data, crcmd5result)
	}()

	go func () {
		defer wg.Done()
		crc32result = DataSignerCrc32(data)
		//fmt.Printf("%s SingleHash crc32(data) %s\n", data, crc32result)
	}()

	wg.Wait()

	result := crc32result + "~" + crcmd5result
	//fmt.Printf("%s SingleHash result %s\n", data, result)
	out <- result
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		dataString := fmt.Sprintf("%v", data)
		go func(data string) {
			defer wg.Done()
			MultiHashCount(data, out)
		}(dataString)
	}
	wg.Wait()
}

func MultiHashCount(data string, out chan interface{}) {
	resultArray := make([]string, 6)

	result := ""
	wg := &sync.WaitGroup{}
	wg.Add(6)
	for i := 0; i < 6; i++ {
		go func(th int, data string) {
			defer wg.Done()
			MultiHashStep(th, data, resultArray)
		}(i, data)
	}
	wg.Wait()

	for _, item := range resultArray {
		result += item
	}

	//fmt.Printf("%s MyltiHash result %s\n", data, result)
	out <- result
}

func MultiHashStep(th int, data string, result []string) {
	stepResult := DataSignerCrc32(strconv.Itoa(th) + data)
	//fmt.Printf("%s MyltiHash :crc32(th+data) %d %s\n", data, th, stepResult)
	result[th] = stepResult
}

func CombineResults(in, out chan interface{}) {
	arrayData := make([]string, 0)
	result := ""

	for item := range in {
		arrayData = append(arrayData, item.(string))
	}

	sort.Strings(arrayData)

	for _, data := range arrayData {
		if result != "" {
			result += "_"
		}
		result += data
	}

	out <- result
}

func main() {

}