package main

/*
Что выведет программа?

Объяснить вывод программы.

	func main() {
	  ch := make(chan int)
	  go func() {
	    for i := 0; i &lt; 10; i++ {
	    ch &lt;- i
	  }
	}()

	  for n := range ch {
	    println(n)
	  }
	}
*/

// исходном задании опечатка &lt; вместо символа  <
/*
0
1
2
3
4
5
6
7
8
9
fatal error: all goroutines are asleep - deadlock!
*/
func main() {
	ch := make(chan int)
	go func() {                   //горутина запускается
		for i := 0; i < 10; i++ { //Горутина отправляет числа 0–9 в канал.
			ch <- i
		}
		close(ch) // <-- ВАЖНО: закрыть канал в исходном коде канал не закрыт
	}() //горутина-отправитель уже завершена
	//Однако: канал не закрыт → for range считает, что могут быть ещё данные
	for n := range ch { //for range ch НЕ ЗАВЕРШИТСЯ, если канал не закрыт!
		println(n)
	}
}
