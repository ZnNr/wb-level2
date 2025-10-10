package main

/*
Что выведет программа?

Объяснить вывод программы.

package main

type customError struct {
  msg string
}

func (e *customError) Error() string {
  return e.msg
}

func test() *customError {
  // ... do something
  return nil
}

func main() {
  var err error
  err = test()
  if err != nil {
    println("error")
    return
  }
  println("ok")
}
*/

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}

func test() *customError {
	// ... do something
	return nil
}

func main() {
	var err error
	err = test() //код демонстрирует тонкую, но критическую ошибку в Go, связанную со сравнением интерфейсов и nil
	if err != nil {
		println("error")
		return
	}
	println("ok")
}

//Ожидание: "Возвращается nil, значит, err == nil ожидание: вывод: ok"

//Реальность: Программа выведет: error

/*
Функция test() возвращает *customError — указатель на структуру.
Даже если он nil, тип у значения всё ещё есть: *customError.
Когда присваивается это значение переменной типа error (интерфейс):

var err error
err = test() // test() возвращает (*customError)(nil)

Переменная err становится интерфейсом, который содержит:
Тип: *customError
Значение: nil

Такой интерфейс НЕ равен nil, потому что его тип не нулевой!
*/
