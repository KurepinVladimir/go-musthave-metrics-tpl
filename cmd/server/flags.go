package main

import (
	"flag"
	"log"
	"os"
	"strconv"
)

// неэкспортированная переменная flagRunAddr содержит адрес и порт для запуска сервера
var flagRunAddr string
var flagStoreInterval int64
var flagFileStoragePath string
var flagRestore bool

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")
	//flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")

	flag.Int64Var(&flagStoreInterval, "i", 300, "storage interval in seconds")
	flag.StringVar(&flagFileStoragePath, "f", "/tmp/metrics-db.json", "file storage path")
	flag.BoolVar(&flagRestore, "r", false, "restore from file storage")

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	// проверка на неизвестные аргументы
	if len(flag.Args()) > 0 {
		log.Fatalf("Неизвестные аргументы: %v", flag.Args())
	}

	// Переделать на "github.com/caarlos0/env/v6"

	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr

		// if !strings.HasPrefix(flagRunAddr, "http://") && !strings.HasPrefix(flagRunAddr, "https://") {
		// 	flagRunAddr = "http://" + flagRunAddr
		// }

	}

	// if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
	// 	if val, err := strconv.ParseInt(envStoreInterval, 10, 64); err == nil {
	// 		flagStoreInterval = val
	// 	} else {
	// 		log.Fatalf("Ошибка преобразования STORE_INTERVAL в int64: %v", err)
	// 	}
	// }

	// if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
	// 	flagFileStoragePath = envFileStoragePath
	// }
	// if envRestore := os.Getenv("RESTORE"); envRestore != "" {
	// 	// convert string to bool
	// 	if val, err := strconv.ParseBool(envRestore); err == nil {
	// 		flagRestore = val
	// 	} else {
	// 		log.Fatalf("Ошибка преобразования RESTORE в bool: %v", err)
	// 	}
	// }

	if env := os.Getenv("STORE_INTERVAL"); env != "" {
		if v, err := strconv.ParseInt(env, 10, 64); err == nil {
			flagStoreInterval = v
		}
	}
	if env := os.Getenv("FILE_STORAGE_PATH"); env != "" {
		flagFileStoragePath = env
	}
	if env := os.Getenv("RESTORE"); env != "" {
		if v, err := strconv.ParseBool(env); err == nil {
			flagRestore = v
		}
	}

}
