package main

import (
	"fmt"
	"reflect"
	"time"
)

/*
Функция or (объединение done-каналов)
Реализовать функцию, которая будет объединять один или более каналов done (каналов сигнала завершения) в один. Возвращаемый канал должен закрываться, как только закроется любой из исходных каналов.

Сигнатура функции может быть такой:

var or func(channels ...&lt;-chan interface{}) &lt;-chan interface{}
Пример использования функции:

sig := func(after time.Duration) &lt;-chan interface{} {
   c := make(chan interface{})
   go func() {
      defer close(c)
      time.Sleep(after)
   }()
   return c
}

start := time.Now()
&lt;-or(
   sig(2*time.Hour),
   sig(5*time.Minute),
   sig(1*time.Second),
   sig(1*time.Hour),
   sig(1*time.Minute),
)
fmt.Printf("done after %v", time.Since(start))
В этом примере канал, возвращённый or(...), закроется через ~1 секунду, потому что самый короткий канал sig(1*time.Second) закроется первым. Ваша реализация or должна уметь принимать на вход произвольное число каналов и завершаться при сигнале на любом из них.

Подсказка: используйте select в бесконечном цикле для чтения из всех каналов одновременно, либо рекурсивно объединяйте каналы попарно.
*/

// or объединяет несколько done-каналов в один
func or(channels ...<-chan interface{}) <-chan interface{} {
	// Базовые случаи
	switch len(channels) {
	case 0:
		// Если каналов нет - возвращаем уже закрытый канал
		c := make(chan interface{})
		close(c)
		return c
	case 1:
		// Если канал один - возвращаем его как есть
		return channels[0]
	}

	// Создаем канал для результата
	orDone := make(chan interface{})

	// Запускаем горутину для отслеживания всех каналов
	go func() {
		defer close(orDone)

		// Рекурсивно объединяем каналы попарно
		select {
		case <-channels[0]:
		case <-or(append(channels[1:])...):
		}
	}()

	return orDone
}

// Альтернативная реализация с использованием цикла и select
func orSelect(channels ...<-chan interface{}) <-chan interface{} {
	switch len(channels) {
	case 0:
		c := make(chan interface{})
		close(c)
		return c
	case 1:
		return channels[0]
	}

	orDone := make(chan interface{})

	go func() {
		defer close(orDone)

		// Создаем канал для сигнала от любой из горутин
		done := make(chan struct{})

		// Запускаем горутины для каждого канала
		for _, ch := range channels {
			go func(c <-chan interface{}) {
				select {
				case <-c:
					close(done)
				case <-done:
					return
				}
			}(ch)
		}

		// Ждем сигнала закрытия
		<-done
	}()

	return orDone
}

// Оптимизированная версия с использованием одного select в цикле
func orOptimized(channels ...<-chan interface{}) <-chan interface{} {
	// Специальные случаи
	switch len(channels) {
	case 0:
		c := make(chan interface{})
		close(c)
		return c
	case 1:
		return channels[0]
	}

	orDone := make(chan interface{})

	go func() {
		defer close(orDone)

		// Используем select в бесконечном цикле
		for {
			for i := 0; i < len(channels); i++ {
				select {
				case <-channels[i]:
					// Один из каналов закрылся - выходим
					return
				default:
					// Продолжаем проверять
				}
			}
			// Небольшая пауза, чтобы не загружать CPU
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return orDone
}

// Еще одна версия - с использованием reflect.Select (более эффективная для многих каналов)
func orReflect(channels ...<-chan interface{}) <-chan interface{} {
	switch len(channels) {
	case 0:
		c := make(chan interface{})
		close(c)
		return c
	case 1:
		return channels[0]
	}

	orDone := make(chan interface{})

	go func() {
		defer close(orDone)

		// Создаем слайс случаев для select
		cases := make([]reflect.SelectCase, len(channels))
		for i, ch := range channels {
			cases[i] = reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch),
			}
		}

		// Ждем, пока один из каналов не закроется
		reflect.Select(cases)
	}()

	return orDone
}

// sig создает канал, который закрывается через указанное время
func sig(after time.Duration) <-chan interface{} {
	c := make(chan interface{})
	go func() {
		defer close(c)
		time.Sleep(after)
	}()
	return c
}

func main() {
	// Тестируем базовую реализацию
	fmt.Println("Тест 1: Базовая реализация (рекурсивная)")
	start := time.Now()
	<-or(
		sig(2*time.Hour),
		sig(5*time.Minute),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)
	fmt.Printf("done after %v\n\n", time.Since(start))

	// Тестируем реализацию с select
	fmt.Println("Тест 2: Реализация с select")
	start = time.Now()
	<-orSelect(
		sig(2*time.Hour),
		sig(500*time.Millisecond),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)
	fmt.Printf("done after %v\n\n", time.Since(start))

	// Тестируем оптимизированную версию
	fmt.Println("Тест 3: Оптимизированная версия")
	start = time.Now()
	<-orOptimized(
		sig(2*time.Hour),
		sig(200*time.Millisecond),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)
	fmt.Printf("done after %v\n\n", time.Since(start))

	// Тест с нулевым количеством каналов
	fmt.Println("Тест 4: Нулевое количество каналов")
	start = time.Now()
	<-or()
	fmt.Printf("done after %v\n\n", time.Since(start))

	// Тест с одним каналом
	fmt.Println("Тест 5: Один канал")
	start = time.Now()
	<-or(sig(300 * time.Millisecond))
	fmt.Printf("done after %v\n", time.Since(start))
}
