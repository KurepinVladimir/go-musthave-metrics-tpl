package main

import (
	"flag"
	"log"

	//"os"
	//"strconv"
	"strings"

	//"time"

	"github.com/caarlos0/env/v6"
)

// неэкспортированная переменная flagRunAddr содержит адрес и порт для запроса
var (
	flagRunAddr        string
	flagReportInterval int64
	flagPollInterval   int64
)

// type Config struct {
// 	RunAddr        string        `env:"ADDRESS"`
// 	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
// 	PollInterval   time.Duration `env:"POLL_INTERVAL"`
// }

type Config struct {
	RunAddr        string `env:"ADDRESS"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	PollInterval   int    `env:"POLL_INTERVAL"`
}

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

	if !strings.HasPrefix(flagRunAddr, "http://") && !strings.HasPrefix(flagRunAddr, "https://") {
		flagRunAddr = "http://" + flagRunAddr
	}

	// проверка на неизвестные аргументы
	if len(flag.Args()) > 0 {
		log.Fatalf("Неизвестные аргументы: %v", flag.Args())
	}

	// читаем переменные окружения и заполняем структуру Config
	// если переменные окружения не заданы, то будут использованы значения по умолчанию
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	if envRunAddr := cfg.RunAddr; envRunAddr != "" {

		flagRunAddr = envRunAddr

		if !strings.HasPrefix(flagRunAddr, "http://") && !strings.HasPrefix(flagRunAddr, "https://") {
			flagRunAddr = "http://" + flagRunAddr
		}
	}

	if envReportInterval := cfg.ReportInterval; envReportInterval != 0 {
		//flagReportInterval = int64(envReportInterval.Seconds())
		flagReportInterval = int64(envReportInterval)
	}

	if envPollInterval := cfg.PollInterval; envPollInterval != 0 {
		//flagPollInterval = int64(envPollInterval.Seconds())
		flagPollInterval = int64(envPollInterval)
	}

}
