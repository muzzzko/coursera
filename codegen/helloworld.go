package main

import "fmt"

type MyStruct struct {
	Name string `my:"some"`
}

// apigen
func (m *MyStruct) HelloWorld() {
	m.Name = "Egor"
	fmt.Printf("Hello %s", m.Name)
}
