package main

import (
	"github.com/RichardKnop/machinery/v1/tasks"
	"machinery-demo/server"
)

// 创建一个任务
func sendTask() {
	s, _ := server.StartServer()

	s.SendTask(&tasks.Signature{
		Name: "push",
		Args: []tasks.Arg{
			{Name: "pushToken", Type: "string", Value: "acc01xxxcfu1"},
			{Name: "pushContent", Type: "string", Value: "hello,world"},
		},
	})
}

//// 并发创建多个任务
func sendConcurrencyTask() {

	s, _ := server.StartServer()

	group, _ := tasks.NewGroup(
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "acc01xxxcfu1"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "x00c1acufl1h"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "c1s20cgalkj"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
	)

	s.SendGroup(group, len(group.Tasks))
}

// 执行回调任务
func sendChordTask() {
	s, _ := server.StartServer()

	group, _ := tasks.NewGroup(
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "acc01xxxcfu1"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "x00c1acufl1h"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
		&tasks.Signature{
			Name: "push",
			Args: []tasks.Arg{
				{Name: "pushToken", Type: "string", Value: "c1s20cgalkj"},
				{Name: "pushContent", Type: "string", Value: "hello,world"},
			},
		},
	)

	chord, _ := tasks.NewChord(group, &tasks.Signature{
		Name: "afterPush",
	})

	s.SendChord(chord, len(group.Tasks))
}

// 执行回调任务
func sendChainTask() {
	s, _ := server.StartServer()

	chain, _ := tasks.NewChain(
		&tasks.Signature{
			Name: "audit",
			Args: []tasks.Arg{
				{Name: "business_id", Type: "int", Value: 1},
			},
		},
		&tasks.Signature{
			Name: "business",
			Args: []tasks.Arg{
				{Name: "business_id", Type: "int", Value: 1},
			},
		},
	)

	s.SendChain(chain)
}

func main() {
	sendChainTask()
}
