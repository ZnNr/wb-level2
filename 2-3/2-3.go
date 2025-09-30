package main

/*
Что выведет программа?

Объяснить внутреннее устройство интерфейсов и их отличие от пустых интерфейсов.
*/
import (
	"fmt"
	"os"
)

func Foo() error {
	var err *os.PathError = nil //cоздается переменная err типа *os.PathError
	//err — это nil-указатель конкретного типа (*os.PathError).
	return err
}

func main() {
	err := Foo()
	fmt.Println(err)        //<nil>
	fmt.Println(err == nil) // false
}

/*
<nil>
false
*/
