package main

import "machinery-demo/server"

func worker() error {
	consumerTag := "machinery_worker"
	s, err := server.StartServer()
	if err != nil {
		return err
	}

	worker := s.NewWorker(consumerTag, 0)

	return worker.Launch()
}

func main() {
	worker()
}
