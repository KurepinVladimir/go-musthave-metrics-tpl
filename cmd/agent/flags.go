package main

import (
	"flag"
	"log"
)

// неэкспортированная переменная flagRunAddr содержит адрес и порт для запроса
var (
	flagRunAddr        string
	flagReportInterval int64
	flagPollInterval   int64
)

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {

	// регистрируем переменную flagRunAddr как аргумент -a со значением :8080 по умолчанию
	// Флаг -a=<ЗНАЧЕНИЕ> отвечает за адрес эндпоинта HTTP-сервера (по умолчанию localhost:8080).
	//flag.StringVar(&flagRunAddr, "a", ":8080", "address and port")
	//flag.StringVar(&flagRunAddr, "a", "http://localhost:8080/update", "address and port")
	//flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port")
	//flag.StringVar(&flagRunAddr, "a", ":8080", "address and port")
	flag.StringVar(&flagRunAddr, "a", "http://localhost:8080", "address and port")

	// Флаг -r=<ЗНАЧЕНИЕ> позволяет переопределять reportInterval — частоту отправки метрик на сервер (по умолчанию 10 секунд).
	flag.Int64Var(&flagReportInterval, "r", 10, "report interval in seconds")

	// Флаг -p=<ЗНАЧЕНИЕ> позволяет переопределять pollInterval — частоту опроса метрик из пакета runtime (по умолчанию 2 секунды).
	flag.Int64Var(&flagPollInterval, "p", 2, "poll interval in seconds")

	// парсим переданные аргументы в зарегистрированные переменные
	flag.Parse()

	// проверка на неизвестные аргументы
	if len(flag.Args()) > 0 {
		log.Fatalf("Неизвестные аргументы: %v", flag.Args())
	}
}
