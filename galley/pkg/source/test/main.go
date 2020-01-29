package main

import (
	"fmt"
	"istio.io/istio/galley/pkg/meshconfig"
	kubeMeta "istio.io/istio/galley/pkg/metadata/kube"
	"istio.io/istio/galley/pkg/runtime"
	"istio.io/istio/galley/pkg/runtime/resource"
	"istio.io/istio/galley/pkg/source/fs"
	"istio.io/istio/galley/pkg/source/kube/dynamic/converter"
	"istio.io/istio/galley/pkg/testing/events"
	"log"
	"istio.io/pkg/appsignals"
	"os"
	"syscall"
)
var (
	cfg = &converter.Config{Mesh: meshconfig.NewInMemory()}
)

func startOrFail(s runtime.Source) chan resource.Event {
	ch := make(chan resource.Event, 1024)
	err := s.Start(events.ChannelHandler(ch))
	if err != nil {
		log.Fatalf("Start error:%v\n", err)
	}
	return ch
}


func newOrFail(dir string) runtime.Source {
	s, err := fs.New(dir, kubeMeta.Types, cfg)
	if err != nil {
		log.Fatalf("Unexpected error found: %v", err)
	}

	if s == nil {
		log.Fatal("expected non-nil source")
	}
	return s
}

func receive(ch chan resource.Event) {
	for {
		e, _ := <- ch
		fmt.Printf("receive event:%v, entry:%v\n", e, e.Entry)
	}
}

func main()  {
	dir := "./fs"
	shutdown := make(chan os.Signal, 1)
	// 监控文件变化
	appsignals.FileTrigger(dir, syscall.SIGUSR1, shutdown)
	// 创建source
	s := newOrFail(dir)
	// 启动source
	ch := startOrFail(s)
	// 下游接收到的事件
	receive(ch)
}