package main

import (
	"flag"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
)

// неэкспортированная переменная flagRunAddr содержит адрес и порт для запуска сервера
var flagRunAddr string
var flagStoreInterval int64
var flagFileStoragePath string
var flagRestore bool

type Config struct {
	RunAddr         string `env:"ADDRESS"`
	StoreInterval   int64  `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
}

// parseFlags обрабатывает аргументы командной строки
func parseFlags() {
	// регистрируем переменную flagRunAddr как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")
	flag.Int64Var(&flagStoreInterval, "i", 300, "storage interval in seconds")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&flagRestore, "r", false, "restore from file storage")

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	// проверка на неизвестные аргументы
	if len(flag.Args()) > 0 {
		log.Fatalf("Неизвестные аргументы: %v", flag.Args())
	}

	// читаем переменные окружения и заполняем структуру Config
	// если переменные окружения не заданы, то будут использованы значения по умолчанию
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("Ошибка парсинга переменных окружения: %v", err)
	}

	if cfg.RunAddr != "" {
		flagRunAddr = cfg.RunAddr
	}

	// Устанавливаем значение только если переменная окружения была явно задана
	if _, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		flagStoreInterval = cfg.StoreInterval
	}

	if cfg.FileStoragePath != "" {
		flagFileStoragePath = cfg.FileStoragePath
	}

	// Устанавливаем значение только если переменная окружения была явно задана
	if _, ok := os.LookupEnv("RESTORE"); ok {
		flagRestore = cfg.Restore
	}

}
