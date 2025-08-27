# 📊 Go Metrics Service

Высокопроизводительный клиент-сервер для сбора и хранения метрик на Go.  
Состоит из **Агента** (сбор и отправка метрик) и **Сервера** (приём, хранение, выдача).

## 🚀 Возможности

### Сервер
- Приём и хранение метрик типов: `gauge` и `counter`.
- Поддерживаемые протоколы/эндпоинты:
  - **JSON**
    - `POST /update` — одна метрика
    - `POST /updates` — батч метрик
    - `POST /value` — получить метрику по JSON-запросу
  - **Текст/URL**
    - `POST /update/{type}/{name}/{value}`
    - `GET /value/{type}/{name}`
- **Хранилища**:
  - **In-Memory** (по умолчанию)
  - **Файловое сохранение** с периодической записью и восстановлением при старте
  - **PostgreSQL** (через `DATABASE_DSN`; миграции в `migrations/`)
- **Сжатие и подписи**:
  - Автоматическая **распаковка входящего gzip** (если `Content-Encoding: gzip`)
  - **Gzip-ответ** при `Accept-Encoding: gzip`
  - **HMAC-SHA256** подпись запросов/ответов (заголовок `HashSHA256`) — включается ключом `KEY`
- **Логирование** (zap), роутер — **chi**
- **Ретраи** с настраиваемыми задержками для некоторых операций (см. `internal/retry`)

### Агент
- Сбор метрик:
  - `runtime.MemStats` (Alloc, TotalAlloc, NumGC) и `RandomValue`
  - **Системные метрики** через `gopsutil`:  
    `TotalMemory`, `FreeMemory`, `CPUutilization{N}` (по числу логических CPU)
- Отправка:
  - Периодический сбор (`poll-interval`) и периодическая отправка (`report-interval`)
  - **Batched** отправка на `/updates` (gzip + HMAC по ключу)
  - **HTTPS/HTTP** — агент работает поверх любого транспорта; TLS обеспечивается окружением/проксей
  - **Ретраи** с экспоненциальной/ступенчатой задержкой (см. `internal/retry`)
- **Ограничение параллелизма исходящих запросов**:  
  Worker-Pool с верхним лимитом воркеров (**флаг `-l`**, переменная `RATE_LIMIT`)
- Кастомные хедеры:
  - `Content-Encoding: gzip` для тела запроса
  - `HashSHA256` при включённом ключе (`KEY`)

## 🧱 Архитектура и структура

```
cmd/
  agent/                # агент: сбор, батчинг, воркеры, флаги
    main.go
    flags.go
    batcher.go
    worker_pool.go
    collector_sys.go
  server/               # сервер: роуты, флаги, middleware
    main.go
    flags.go
    gzip_middleware.go
internal/
  cryptohelpers/        # HMAC: Sign / Compare
  handler/              # JSON-ответ с подписью (WriteSignedJSONResponse), batch-handlers
  logger/               # zap + HTTP логирование
  middleware/           # ValidateHashSHA256 (проверка подписи запроса)
  models/               # модель Metrics (gauge|counter)
  pgerrors/             # классификация ошибок PostgreSQL
  repository/           # Storage интерфейс и реализации:
                        #   - MemStorage (файл/restore/periodic store)
                        #   - PostgresStorage (pgx, миграции)
  retry/                # retry helper с задержками и контекстом
migrations/
  000001_init.up.sql    # gauge_metrics(name, value), counter_metrics(name, value)
  000001_init.down.sql
```

## 🔐 Безопасность и целостность
- **HMAC-SHA256**:
  - Агент подписывает «сырые» данные **до** сжатия; сервер проверяет заголовок `HashSHA256`
  - Сервер также подписывает JSON-ответы (при наличии ключа)
- **Gzip**:
  - Сервер автоматически распаковывает gzip-тела запросов
  - Выдаёт gzip-ответы, если клиент прислал `Accept-Encoding: gzip`

## ⚙️ Конфигурация

### Сервер
Флаги (и соответствующие переменные окружения):
- `-a` / `ADDRESS` — адрес сервера, напр. `:8080` или `0.0.0.0:8080`
- `-i` / `STORE_INTERVAL` — период сохранения на диск (секунды), `0` — запись на каждый апдейт
- `-f` / `FILE_STORAGE_PATH` — путь к файлу хранилища, напр. `./storage.json`
- `-r` / `RESTORE` — восстанавливать состояние из файла при старте (`true|false`)
- `-d` / `DATABASE_DSN` — строка подключения к PostgreSQL
- `-k` / `KEY` — ключ HMAC-SHA256 для подписей

Примеры:
```bash
go run ./cmd/server   -a ":8080"   -i 300   -f "./storage.json"   -r true   -k "supersecret"

# Через окружение:
ADDRESS=:8080 STORE_INTERVAL=300 FILE_STORAGE_PATH=./storage.json RESTORE=true KEY=supersecret go run ./cmd/server
```

### Агент
Флаги (и переменные окружения):
- `-a` / `ADDRESS` — адрес сервера, напр. `http://localhost:8080`
- `-p` / `POLL_INTERVAL` — период сбора метрик (секунды)
- `-r` / `REPORT_INTERVAL` — период отправки батча (секунды)
- `-k` / `KEY` — ключ HMAC-SHA256
- `-l` / `RATE_LIMIT` — **максимум параллельных исходящих запросов** (worker pool)

Примеры:
```bash
go run ./cmd/agent   -a "http://localhost:8080"   -p 2   -r 10   -l 4   -k "supersecret"

# Через окружение:
ADDRESS=http://localhost:8080 POLL_INTERVAL=2 REPORT_INTERVAL=10 RATE_LIMIT=4 KEY=supersecret go run ./cmd/agent
```

## 🗄️ PostgreSQL

### Миграции
```sql
CREATE TABLE IF NOT EXISTS gauge_metrics (
  name  TEXT PRIMARY KEY,
  value DOUBLE PRECISION NOT NULL
);

CREATE TABLE IF NOT EXISTS counter_metrics (
  name  TEXT PRIMARY KEY,
  value BIGINT NOT NULL
);
```

### Включение Postgres-хранилища
Достаточно задать `DATABASE_DSN`, например:
```bash
DATABASE_DSN=postgres://user:pass@localhost:5432/metrics?sslmode=disable
```
При наличии DSN сервер будет использовать `PostgresStorage`; иначе — файловое или in-memory хранилище (в зависимости от `FILE_STORAGE_PATH`/`STORE_INTERVAL`/`RESTORE`).

## 🧪 Тесты
- `cmd/agent/agent_test.go` — проверка отправки gzip+JSON, заголовков, корректности сериализации
- `cmd/server/server_test.go` — проверка распаковки, парсинга и корректности обработки `/update`, `/updates`
- Используется `testify/assert`

Запуск:
```bash
go test ./...
```

## 📡 Примеры запросов

### JSON: одна метрика
```bash
curl -X POST http://localhost:8080/update   -H "Content-Type: application/json"   -d '{"id":"Alloc","type":"gauge","value":12345.67}'
```

### JSON: батч метрик (gzip)
```bash
printf '[{"id":"Alloc","type":"gauge","value":1.23},{"id":"PollCount","type":"counter","delta":5}]' | gzip | curl -X POST http://localhost:8080/updates     -H "Content-Type: application/json"     -H "Content-Encoding: gzip"     --data-binary @-
```

### Текстовый эндпоинт
```bash
curl -X POST http://localhost:8080/update/gauge/Alloc/123.45
curl "http://localhost:8080/value/gauge/Alloc"
```

## 🛠️ Технологии
- Go, **chi** (HTTP), **zap** (логирование), **resty** (клиент)
- **gopsutil** (CPU/Mem)
- **pgx** (PostgreSQL), миграции (SQL-файлы)
- **testify/assert** (тесты)
- Вспомогательные пакеты: собственные `cryptohelpers`, `retry`, `middleware`

## 🗺️ Дорожная карта
- Расширение схемы БД, оптимизации запросов
- gRPC / OpenAPI контракты
- Экспорт/скрейп для Prometheus, дашборды Grafana
- Веб-интерфейс

## 👤 Автор
**Владимир Курепин** 
[GitHub](https://github.com/KurepinVladimir)
