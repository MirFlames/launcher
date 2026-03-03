# Миграции БД (goose)

Миграции выполняются автоматически при старте backend.

## Создание новой миграции (CLI)

```bash
cd backend
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir migrations create add_new_column sql
```

Будет создан файл `*_add_new_column.sql` с секциями `-- +goose Up` и `-- +goose Down`.

## Ручной запуск миграций

```bash
cd backend
goose -dir migrations sqlite3 ./data/sessions.db up
```

## Откат последней миграции

```bash
goose -dir migrations sqlite3 ./data/sessions.db down
```
