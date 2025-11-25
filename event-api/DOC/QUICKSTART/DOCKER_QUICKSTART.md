# 🚀 Быстрый старт с Docker Compose

## Первый запуск

### 1️⃣ Запустить PostgreSQL

````bash
make docker-up
```bash
Это запустит:

- ✅ PostgreSQL на `localhost:5432`
- ✅ pgAdmin на `http://localhost:5050`

### 2️⃣ Запустить приложение

```bash
make run
````

Приложение:

- Подключится к БД
- Автоматически выполнит миграции
- Запустится на порту `8080`

---

## Альтернатива: Одной командой

````bash
make dev
```bash
Эта команда:

1. Запустит PostgreSQL
2. Подождёт 3 секунды
3. Запустит приложение

---

## Управление БД

```bash
## Просмотр логов PostgreSQL
## Просмотр логов PostgreSQL
make docker-logs

## Перезапуск PostgreSQL
## Перезапуск PostgreSQL
make docker-restart

## Остановка PostgreSQL
## Остановка PostgreSQL
make docker-down

## Полная очистка (удаление всех данных)
## Полная очистка (удаление всех данных)
make docker-clean
````

---

## Что происходит при первом запуске?

1. **Docker создаёт контейнер** `event_api_postgres`
2. **PostgreSQL инициализируется** с параметрами из `.env`:
   - User: `devuser`
   - Password: `devpass`
   - Database: `event_api`
3. **Выполняется** `scripts/init.sql`:
   - Устанавливаются расширения (uuid-ossp, pgcrypto)
   - Устанавливается timezone UTC
4. **Приложение подключается** к БД
5. **Миграции применяются** автоматически:
   - Создаётся таблица `schema_migrations`
   - Создаётся таблица `users`
6. **Сервер запускается** на порту 8080

---

## Переменные окружения (.env)

````env
## База данных
## База данных
DB_HOST=localhost      # Адрес БД
DB_PORT=5432          # Порт БД
DB_USER=devuser       # Пользователь БД (автоматически создаётся)
DB_PASSWORD=devpass   # Пароль БД
DB_NAME=event_api     # Название БД (автоматически создаётся)
DB_SSLMODE=disable    # Режим SSL
```bash
---

## Проверка работы

### Проверить здоровье БД

```bash
docker exec event_api_postgres pg_isready -U devuser
````

### Подключиться к БД через psql

````bash
docker exec -it event_api_postgres psql -U devuser -d event_api
```bash
### Проверить API

```bash
curl http://localhost:8080/health
````

---

## Troubleshooting

### Ошибка: "port is already allocated"

Порт 5432 уже занят другим PostgreSQL.

**Решение:**

````bash
## Найти процесс
## Найти процесс
lsof -i :5432

## Остановить старый контейнер
## Остановить старый контейнер
docker stop <container_name>
docker rm <container_name>

## Или остановить локальный PostgreSQL
## Или остановить локальный PostgreSQL
brew services stop postgresql
```bash
### Ошибка: "database does not exist"

БД не создана автоматически.

**Решение:**

```bash
## Полная перезагрузка с очисткой
## Полная перезагрузка с очисткой
make docker-clean
make docker-up
````

### Ошибка подключения приложения

**Решение:**

````bash
## Проверить что PostgreSQL запущен
## Проверить что PostgreSQL запущен
docker ps | grep postgres

## Проверить логи
## Проверить логи
make docker-logs

## Проверить .env файл
## Проверить .env файл
cat .env | grep DB_
```bash
---

## Команды Makefile

```bash
make docker-up         # Запустить PostgreSQL и pgAdmin
make docker-down       # Остановить контейнеры
make docker-logs       # Показать логи PostgreSQL
make docker-restart    # Перезапустить PostgreSQL
make docker-clean      # Удалить всё (с данными!)
make dev               # Запустить БД + приложение
make run               # Запустить только приложение
make build             # Скомпилировать приложение
````

---

## Готово! 🎉

Теперь ваше приложение полностью готово к работе с автоматической инициализацией
БД при первом запуске.
