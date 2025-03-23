package tasks

import (
	"log"
	"time"
)

type TaskFunc func() error

func task1Func() error {
	log.Println("Executing task 1 specific logic...")
	time.Sleep(70 * time.Second)
	log.Println("Task 1 completed")
	return nil
}

func task2Func() error {
	log.Println("Executing task 2 specific logic...")
	time.Sleep(3 * time.Second)
	log.Println("Task 2 completed")
	return nil
}

func task3Func() error {
	log.Println("Executing task 3 specific logic...")
	time.Sleep(4 * time.Second)
	log.Println("Task 3 completed")
	return nil
}
