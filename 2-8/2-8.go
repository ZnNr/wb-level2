package main

import (
	"fmt"
	"github.com/beevik/ntp"
	"log"
	"os"
)

/*
Получение точного времени (NTP)
Создать программу, печатающую точное текущее время с использованием NTP-сервера.

Реализовать проект как модуль Go.

Использовать библиотеку ntp для получения времени.

Программа должна выводить текущее время, полученное через NTP (Network Time Protocol).

Необходимо обрабатывать ошибки библиотеки: в случае ошибки вывести её текст в STDERR и вернуть ненулевой код выхода.

Код должен проходить проверки (vet и golint), т.е. быть написан идиоматически корректно.
*/
// Package main provides a command-line tool to fetch and display current time from NTP server.

const defaultNTPServer = "pool.ntp.org"

// getNTPTime fetches current time from the specified NTP server.
func getNTPTime(server string) (string, error) {
	time, err := ntp.Time(server)
	if err != nil {
		return "", fmt.Errorf("failed to get time from NTP server %q: %w", server, err)
	}
	return time.Format("2006-01-02 15:04:05.000 MST"), nil
}

func main() {
	server := defaultNTPServer
	if len(os.Args) > 1 {
		server = os.Args[1]
	}

	timeStr, err := getNTPTime(server)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatalf("Error: %v", err)
	}

	fmt.Println(timeStr)
}
