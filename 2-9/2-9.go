package main

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

/*
Написать функцию Go, осуществляющую примитивную распаковку строки, содержащей повторяющиеся символы/руны.

Примеры работы функции:

Вход: "a4bc2d5e"
Выход: "aaaabccddddde"

Вход: "abcd"
Выход: "abcd" (нет цифр — ничего не меняется)

Вход: "45"
Выход: "" (некорректная строка, т.к. в строке только цифры — функция должна вернуть ошибку)

Вход: ""
Выход: "" (пустая строка -> пустая строка)

Дополнительное задание
Поддерживать escape-последовательности вида \:

Вход: "qwe\4\5"
Выход: "qwe45" (4 и 5 не трактуются как числа, т.к. экранированы)

Вход: "qwe\45"
Выход: "qwe44444" (\4 экранирует 4, поэтому распаковывается только 5)

Требования к реализации
Функция должна корректно обрабатывать ошибочные случаи (возвращать ошибку, например, через error), и проходить unit-тесты.

Код должен быть статически анализируем (vet, golint).
*/

// Unpack распаковывает строку, содержащую повторяющиеся символы/руны.
// Поддерживает escape-последовательности вида \.
// Возвращает распакованную строку и ошибку в случае некорректного ввода.
func Unpack(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	var builder strings.Builder
	runes := []rune(s)
	i := 0
	escaped := false

	for i < len(runes) {
		r := runes[i]
		i++

		if escaped {
			// Экранированный символ — просто добавляем его
			builder.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			// Начало escape-последовательности
			if i >= len(runes) {
				// Обратный слэш в конце строки — ошибка
				return "", errors.New("trailing backslash")
			}
			escaped = true
			continue
		}

		// Обычный символ
		if i < len(runes) && unicode.IsDigit(runes[i]) {
			// Следующий символ — цифра, читаем всё число
			start := i
			for i < len(runes) && unicode.IsDigit(runes[i]) {
				i++
			}
			numStr := string(runes[start:i])
			count, err := strconv.Atoi(numStr)
			if err != nil {
				// Не должно произойти, но на всякий случай
				return "", errors.New("invalid number")
			}
			if count == 0 {
				// Пропускаем символ, если указано 0 повторений
				continue
			}
			for j := 0; j < count; j++ {
				builder.WriteRune(r)
			}
		} else {
			// Следующий символ — не цифра, просто добавляем текущий символ один раз
			builder.WriteRune(r)
		}
	}

	// Проверка: если строка заканчивается на обратный слэш
	if escaped {
		return "", errors.New("trailing backslash")
	}

	result := builder.String()

	// Дополнительная проверка: если результат пустой, но вход не был пустым,
	// и строка не содержит ни одного валидного символа (например, "45"),
	// то это ошибка.
	if result == "" && len(s) > 0 {
		// Проверим, состоит ли исходная строка только из цифр и/или слэшей без валидных символов
		hasValidChar := false
		escapedCheck := false
		for _, r := range s {
			if escapedCheck {
				// Экранированный символ считается валидным
				hasValidChar = true
				escapedCheck = false
				continue
			}
			if r == '\\' {
				escapedCheck = true
				continue
			}
			if !unicode.IsDigit(r) {
				hasValidChar = true
				break
			}
		}
		if !hasValidChar {
			return "", errors.New("string contains only digits or invalid sequences")
		}
	}

	return result, nil
}

func main() {
	// тесты функции и примеры
	examples := []string{
		"a4bc2d5e",
		"abcd",
		"45",
		"",
		"qwe\\4\\5",
		"qwe\\45",
		"\\3\\2",
		"3",
		"\\",
	}

	for _, ex := range examples {
		res, err := Unpack(ex)
		if err != nil {
			println("Input:", ex, "=> Error:", err.Error())
		} else {
			println("Input:", ex, "=> Output:", res)
		}
	}
}
