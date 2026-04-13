# Task Service

Сервис для управления задачами с HTTP API на Go.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Причина в том, что SQL-файл из `migrations/0001_create_tasks.up.sql` монтируется в `docker-entrypoint-initdb.d` и применяется только при инициализации пустого data volume.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

Базовый префикс API:

```text
/api/v1
```

Основные маршруты:

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

# Периодические задачи — расширение API трекера задач

## Задача

Расширить существующий CRUD API трекера задач медицинской информационной системы возможностью создавать периодические задачи и генерировать их экземпляры по расписанию.

## Что было до

REST API на Go (Clean Architecture, PostgreSQL) с базовым CRUD: создание, чтение, обновление, удаление задач.

## Что реализовано

**Периодические задачи** — врач может создать задачу-шаблон с настройками повторения. По вызову эндпоинта генерации на заданную дату система создаёт конкретные экземпляры из подходящих шаблонов.

Поддерживаемые типы периодичности:

| Тип | Описание | Пример |
|-----|----------|--------|
| `daily` | Каждые N дней от даты создания | `{"type":"daily","interval":2}` — через день |
| `monthly` | Фиксированный день месяца | `{"type":"monthly","day_of_month":15}` |
| `specific_dates` | Конкретный список дат | `{"type":"specific_dates","dates":["2026-04-15","2026-05-01"]}` |
| `even_odd` | Чётные/нечётные дни месяца | `{"type":"even_odd","even_odd":"even"}` |

## Новый эндпоинт

```
POST /api/v1/tasks/generate?date=2026-04-14
```

Находит все задачи-шаблоны, проверяет совпадение с переданной датой, создаёт экземпляры со статусом `new` и ссылкой на шаблон через `periodic_source_id`.

## Изменённые файлы

```
migrations/0002_add_task_periodicity.up.sql       — миграция: колонки periodicity (JSONB) и periodic_source_id (FK)
docker-compose.yml                                 — монтирование новой миграции
internal/domain/task/task.go                       — типы Periodicity, валидация, логика MatchesDate
internal/repository/postgres/task_repository.go    — расширение SQL-запросов, ListPeriodicTemplates, CreateBatch
internal/usecase/task/ports.go                     — расширение интерфейсов и DTO
internal/usecase/task/service.go                   — валидация периодичности, метод GenerateForDate
internal/transport/http/handlers/dto.go            — periodicityDTO, конвертеры DTO <-> domain
internal/transport/http/handlers/task_handler.go   — хендлер GenerateForDate, прокидка периодичности в Create/Update
internal/transport/http/router.go                  — маршрут POST /tasks/generate
internal/transport/http/docs/openapi.json          — схема Periodicity, обновлённые запросы/ответы, новый эндпоинт
```

## Архитектурные решения

- **Периодичность хранится в JSONB** прямо в таблице `tasks` — проект небольшой, отдельная таблица избыточна
- **Периодическая задача = шаблон**, экземпляры ссылаются на него через `periodic_source_id`
- **Генерация — через API-вызов**, без фоновых процессов — упрощает инфраструктуру
- **Массовая вставка** (`CreateBatch`) — один SQL-запрос вместо N отдельных INSERT
- **Обратная совместимость** — существующие задачи без периодичности работают как раньше, новые поля nullable

## Как проверить

```bash
docker compose down -v && docker compose up --build
```

```bash
# Создать периодическую задачу
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Обзвон пациентов","periodicity":{"type":"daily","interval":1}}'

# Сгенерировать экземпляры на дату
curl -X POST "http://localhost:8080/api/v1/tasks/generate?date=2026-04-14"

# Проверить результат
curl http://localhost:8080/api/v1/tasks

# Swagger UI
# http://localhost:8080/swagger/
```
