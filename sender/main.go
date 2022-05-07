package main

import (
	"fmt"
	"github.com/RichardKnop/machinery/v1/log"
	"github.com/RichardKnop/machinery/v1/tasks"
	"machinery-demo/server"
	"time"
)

// 创建一个任务
func sendTask() {
	s, err := server.StartServer()
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	task := server.CreatePushTask("zhangsan", "hello, world")

	for i := 0; i < 10; i++ {
		asyncResult, err := s.SendTask(task)
		if err != nil {
			log.ERROR.Printf("%v\n", err)
		}
		results, err := asyncResult.Get(time.Duration(time.Millisecond * 5))
		if err != nil {
			log.ERROR.Printf("%v\n", err)
		}

		fmt.Printf("%v\n", tasks.HumanReadableResults(results))
	}
}

// 并发创建多个任务
func sendParallelTask() {
	s, err := server.StartServer()
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	group, err := tasks.NewGroup(
		server.CreatePushTask("zhangsan", "hell world"),
		server.CreatePushTask("lisi", "hell world"),
		server.CreatePushTask("wangwu", "hell world"),
	)
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	asyncResults, err := s.SendGroup(group, 3)
	if err != nil {
		log.ERROR.Printf("%v", err)
	}

	for _, results := range asyncResults {
		result, err := results.Get(time.Duration(time.Millisecond * 5))
		if err != nil {
			log.ERROR.Printf("%v", err)
		}

		log.INFO.Printf("%v", tasks.HumanReadableResults(result))
	}
}

// 并发执行多个任务且完成后执行回调操作
func sendChordTask() {
	s, err := server.StartServer()
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	group, err := tasks.NewGroup(
		server.CreateAuditTask(1, 123),
		server.CreateAuditTask(2, 123),
		server.CreateAuditTask(3, 123),
	)
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	chord, err := tasks.NewChord(group, server.CreateAuditCallback())

	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	chordAsyncResult, err := s.SendChord(chord, len(group.Tasks))
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	results, err := chordAsyncResult.Get(time.Duration(time.Millisecond * 5))
	if err != nil {
		log.ERROR.Printf("%v\n", err)
	}

	log.INFO.Printf("%v", tasks.HumanReadableResults(results))
}

// 串行执行多个任务且完成后执行回调操作
func sendChainTask() {
	s, err := server.StartServer()

	chain, err := tasks.NewChain(
		server.CreateAuditTask(1, 123),
		server.CreatePushTask("userId1's Token", "there is a new audit"),
		server.CreateAuditCallback(),
	)

	if err != nil {
		log.ERROR.Printf("%v", err)
		return
	}

	chainAsyncResult, err := s.SendChain(chain)
	if err != nil {
		log.ERROR.Printf("%v", err)
		return
	}

	result, err := chainAsyncResult.Get(time.Duration(time.Millisecond * 5))
	if err != nil {
		log.ERROR.Printf("%v", err)
		return
	}

	log.INFO.Printf("%v", tasks.HumanReadableResults(result))
}

// 创建延时任务
func sendDelayTask() {
	s, err := server.StartServer()
	if err != nil {
		log.ERROR.Printf("%v", err)
	}

	task := server.CreatePushTask("user's token", "this is a delay message")
	eta := time.Now().UTC().Add(time.Second * 5)
	task.ETA = &eta

	asyncResult, _ := s.SendTask(task)

	result, _ := asyncResult.Get(time.Millisecond * 5)
	log.INFO.Printf("%v", tasks.HumanReadableResults(result))
}

func main() {
	sendTask()
	//sendParallelTask()
	//sendChordTask()
	//sendChainTask()
	//sendDelayTask()
}
