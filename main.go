package main

import (
	"fmt"
	"sync"

	"net/http"

	_ "net/http/pprof"

	"time"

	"github.com/hashicorp/consul/watch"
	"mosn.io/holmes"
)

func init() {
	//memory
	http.HandleFunc("/make1gb", make1gbslice)

	//deadlock
	http.HandleFunc("/lockorder1", lockorder1)
	http.HandleFunc("/lockorder2", lockorder2)
	http.HandleFunc("/req", req)

	//chanblock
	http.HandleFunc("/chanblock", channelBlock)

	//goroutine leaks
	http.HandleFunc("/leak", leak)

	//large memory allocation caused by business logic
	http.HandleFunc("/alloc", alloc)

	//cpu outage, deadloop
	http.HandleFunc("/cpuex", cpuex)

	http.HandleFunc("/deadloop", deadloop)

	go http.ListenAndServe(":10003", nil)
}

func main() {
	h, _ := holmes.New(
		holmes.WithCollectInterval("2s"),
		holmes.WithDumpPath("/tmp"),
		// holmes.WithLogger(holmes.NewFileLog("/tmp/holmes.log", mlog.INFO)),
		holmes.WithTextDump(),
		// holmes.WithCGroup(true), // set cgroup to true
		holmes.WithCPUDump(10, 25, 80, time.Minute),
		holmes.WithMemDump(3, 25, 80, time.Minute),
		holmes.WithGCHeapDump(10, 20, 40, time.Minute),
		holmes.WithGoroutineDump(10, 25, 2000, 10000, time.Minute),
	)

	h.EnableCPUDump().EnableMemDump().EnableGCHeapDump().EnableGoroutineDump().Start()

	time.Sleep(time.Hour)
}

func make1gbslice(wr http.ResponseWriter, req *http.Request) {
	var a = make([]byte, 1073741824)
	_ = a
}

var l1 sync.Mutex
var l2 sync.Mutex

func req(wr http.ResponseWriter, req *http.Request) {
	l1.Lock()
	defer l1.Unlock()
}

func lockorder1(wr http.ResponseWriter, req *http.Request) {
	l1.Lock()
	defer l1.Lock()

	time.Sleep(time.Minute)

	l2.Lock()
	defer l2.Lock()

}

func lockorder2(wr http.ResponseWriter, req *http.Request) {
	l2.Lock()
	defer l2.Lock()

	time.Sleep(time.Minute)

	l1.Lock()
	defer l1.Unlock()

}

var nilCh chan int

func channelBlock(wr http.ResponseWriter, req *http.Request) {
	nilCh <- 1
}

func leak(wr http.ResponseWriter, req *http.Request) {
	taskChan := make(chan int)
	consumer := func() {
		for task := range taskChan {
			_ = task
		}
	}

	producer := func() {
		for i := 0; i < 10; i++ {
			taskChan <- i
		}
		//forget to close taskChan
	}

	go consumer()
	go producer()
}

func alloc(wr http.ResponseWriter, req *http.Request) {
	var m = make(map[string]string, 102400)
	for i := 0; i < 1000; i++ {
		m[fmt.Sprint(i)] = fmt.Sprint(i)
	}
	_ = m
}

func cpuex(wr http.ResponseWriter, req *http.Request) {
	go func() {
		for {
			// time.Sleep(time.Millisecond)
		}
	}()
}

func deadloop(wr http.ResponseWriter, req *http.Request) {
	for i := 0; i < 4; i++ {
		for {
			time.Sleep(time.Millisecond)
		}
	}
}

func test() {
	params := make(map[string]string)
	params["type"] = ""
	params["name"] = ""
	plan, err := watch.Parse(params)
	if err != nil {
		panic(err)
	}
	if plan == nil {
		panic("plan is nil")
	}
	plan.Token = ""
}
