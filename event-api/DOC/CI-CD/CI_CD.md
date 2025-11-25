# CI/CD Documentation

## Обзор

Проект использует GitHub Actions для автоматизации процессов тестирования,
сборки и деплоя.

## Workflows

### 1. CI Pipeline (`.github/workflows/ci.yml`)

Основной CI/CD pipeline, запускается при:

- Push в ветки `master`, `main`, `develop`
- Pull Request в эти ветки

#### Jobs

##### 1.1. Lint (Проверка кода)

- ✅ golangci-lint с расширенными правилами
- ✅ Проверка форматирования (gofmt)
- ✅ Go vet для статического анализа
- ✅ Кэширование Go модулей для ускорения

##### 1.2. Test (Тестирование)

- ✅ Запуск PostgreSQL 17 и Redis 7 через GitHub Services
- ✅ Выполнение всех тестов с флагами `-race` и `-cover`
- ✅ Генерация отчёта о покрытии кода
- ✅ Загрузка coverage в Codecov (опционально)
- ✅ Артефакты с отчётом о покрытии

##### 1.3. Build (Сборка Docker образа)

- ✅ Сборка Docker образа приложения
- ✅ Кэширование Docker layers для ускорения
- ✅ Запускается только при push (не для PR)

##### 1.4. Security (Сканирование безопасности)

- ✅ Trivy для сканирования уязвимостей в файловой системе
- ✅ GoSec для анализа безопасности Go кода
- ✅ Загрузка результатов в GitHub Security

##### 1.5. Deploy (Деплой на Production)

- ✅ Запускается только для `master`/`main` ветки
- ✅ Требует ручного подтверждения (GitHub Environment)
- 🔧 Настраивается под ваш метод деплоя

### 2. Staging Pipeline (`.github/workflows/staging.yml`)

Деплой на staging окружение:

- Запускается при push в ветку `develop`
- Можно запустить вручную через `workflow_dispatch`
- Быстрые тесты перед деплоем
- Автоматический деплой на staging сервер

### 3. Release Pipeline (`.github/workflows/release.yml`)

Создание релизов с тегами версий:

- Запускается при push тега формата `v*.*.*` (например, `v1.0.0`)
- Сборка бинарников для всех платформ:
  - Linux AMD64/ARM64
  - macOS AMD64/ARM64
  - Windows AMD64
- Создание checksums для проверки целостности
- Сборка и публикация Docker образа с версией
- Автоматическая генерация changelog из commit messages
- Создание GitHub Release с бинарниками и changelog

## Настройка CI/CD

### Шаг 1: Базовая настройка

CI/CD уже настроен и будет работать автоматически после push в GitHub. Базовый
функционал (lint, test, build) работает без дополнительных настроек.

### Шаг 2: Настройка Secrets

Перейдите в **Settings → Secrets and variables → Actions** вашего GitHub
репозитория и добавьте:

#### Для Docker Hub (опционально)

````text
DOCKER_USERNAME - ваш Docker Hub username
DOCKER_PASSWORD - ваш Docker Hub password или access token
```bash
#### Для SSH деплоя (опционально)

```text
SSH_HOST - IP или домен вашего сервера
SSH_USERNAME - username для SSH
SSH_PRIVATE_KEY - приватный SSH ключ
SSH_PORT - порт SSH (по умолчанию 22)
````

#### Для staging (опционально)

````text
STAGING_SSH_HOST
STAGING_SSH_USERNAME
STAGING_SSH_PRIVATE_KEY
STAGING_SSH_PORT
```bash
### Шаг 3: Настройка Environments

1. Перейдите в **Settings → Environments**
2. Создайте окружения:
   - `production` - для production деплоя
   - `staging` - для staging деплоя

3. Для production рекомендуется включить:
   - **Required reviewers** - требовать подтверждение от команды
   - **Wait timer** - задержка перед деплоем
   - **Deployment branches** - только `main`/`master`

### Шаг 4: Codecov (опционально)

Для отслеживания покрытия кода:

1. Зарегистрируйтесь на [https://codecov.io](dasda)
2. Подключите ваш GitHub репозиторий
3. Добавьте токен в GitHub Secrets:

   ```text
   CODECOV_TOKEN - токен из Codecov
````

### Шаг 5: Настройка деплоя

#### Вариант А: Docker Hub + SSH

Раскомментируйте в `.github/workflows/ci.yml` (job `deploy`):

```yaml
## Логин в Docker Hub
## Логин в Docker Hub
- name: Login to Docker Hub
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKER_USERNAME }}
    password: ${{ secrets.DOCKER_PASSWORD }}

## Сборка и push образа
## Сборка и push образа
- name: Build and push Docker image
  uses: docker/build-push-action@v6
  with:
    context: .
    push: true
    tags: |
      ${{ secrets.DOCKER_USERNAME }}/${{ env.DOCKER_IMAGE }}:latest
      ${{ secrets.DOCKER_USERNAME }}/${{ env.DOCKER_IMAGE }}:${{ github.sha }}

## SSH деплой
## SSH деплой
- name: Deploy to server via SSH
  uses: appleboy/ssh-action@v1.0.0
  with:
    host: ${{ secrets.SSH_HOST }}
    username: ${{ secrets.SSH_USERNAME }}
    key: ${{ secrets.SSH_PRIVATE_KEY }}
    port: ${{ secrets.SSH_PORT }}
    script: |
      cd /path/to/app
      docker-compose pull
      docker-compose up -d
      docker-compose ps
```

#### Вариант Б: GitHub Container Registry

````yaml
- name: Login to GitHub Container Registry
  uses: docker/login-action@v3
  with:
    registry: ghcr.io
    username: ${{ github.actor }}
    password: ${{ secrets.GITHUB_TOKEN }}

- name: Build and push Docker image
  uses: docker/build-push-action@v6
  with:
    context: .
    push: true
    tags: |
      ghcr.io/${{ github.repository }}:latest
      ghcr.io/${{ github.repository }}:${{ github.sha }}
```bash
#### Вариант В: Cloud Providers

- **AWS ECS/EKS**: Используйте `aws-actions/amazon-ecs-deploy-task-definition`
- **Google Cloud Run**: Используйте `google-github-actions/deploy-cloudrun`
- **Azure**: Используйте `azure/webapps-deploy`
- **DigitalOcean**: Используйте `digitalocean/action-doctl`

### Шаг 6: Настройка линтера

Файл `.golangci.yml` уже создан с оптимальными правилами. Вы можете настроить
его под свои нужды:

```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    # ... добавьте или удалите линтеры
````

Запуск локально:

````bash
## Установка golangci-lint
## Установка golangci-lint
brew install golangci-lint  # macOS
## или
## или
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

## Запуск
## Запуск
golangci-lint run
```bash
## Локальные скрипты

### Deploy Script (`scripts/deploy.sh`)

Деплой приложения на сервер:

```bash
## Production деплой
## Production деплой
./scripts/deploy.sh production

## Staging деплой
## Staging деплой
./scripts/deploy.sh staging
````

Скрипт выполняет:

1. ✅ Проверку Git статуса
2. ✅ Создание бэкапа БД
3. ✅ Pull последних изменений
4. ✅ Сборку Docker образа
5. ✅ Перезапуск контейнеров
6. ✅ Health check приложения
7. ✅ Очистку старых образов

### Backup Script (`scripts/backup.sh`)

Создание бэкапа базы данных:

````bash
## Бэкап дефолтной БД (event_api)
## Бэкап дефолтной БД (event_api)
./scripts/backup.sh

## Бэкап конкретной БД
## Бэкап конкретной БД
./scripts/backup.sh my_database
```bash
Особенности:

- Сжатие через gzip
- Автоматическое удаление старых бэкапов (>7 дней)
- Сохранение в директорию `./backups/`

### Restore Script (`scripts/restore.sh`)

Восстановление из бэкапа:

```bash
## Восстановление
## Восстановление
./scripts/restore.sh backups/event_api_backup_20250104_120000.sql.gz

## В конкретную БД
## В конкретную БД
./scripts/restore.sh backups/backup.sql.gz my_database
````

⚠️ **Внимание**: Это перезапишет текущую базу данных!

## Production Deployment

### Docker Compose Production

Файл `docker-compose.prod.yml` настроен для production:

````bash
## Запуск всех сервисов
## Запуск всех сервисов
docker-compose -f docker-compose.prod.yml up -d

## С Nginx reverse proxy
## С Nginx reverse proxy
docker-compose -f docker-compose.prod.yml --profile with-nginx up -d

## Остановка
## Остановка
docker-compose -f docker-compose.prod.yml down

## Логи
## Логи
docker-compose -f docker-compose.prod.yml logs -f app
```bash
### Переменные окружения для Production

Создайте `.env.production`:

```env
## Server
## Server
PORT=8080
HOST=0.0.0.0
ENV=production

## Database
## Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=your_db_user
DB_PASSWORD=strong_password_here
DB_NAME=event_api
DB_SSLMODE=require

## Redis
## Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=strong_redis_password
REDIS_DB=0

## JWT
## JWT
JWT_SECRET=very_long_random_secret_key_change_this_in_production
JWT_EXPIRATION=24h

## Email
## Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_password
SMTP_FROM=noreply@yourdomain.com

## SMS (выберите провайдер)
## SMS (выберите провайдер)
SMS_PROVIDER=smsru  # или smsc, twilio
SMS_API_KEY=your_api_key
SMS_API_TOKEN=your_api_token
SMS_FROM=YourApp

## CORS
## CORS
CORS_ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
````

### Nginx Reverse Proxy

Файл `nginx.conf` настроен с:

- Rate limiting (10 req/s)
- Gzip compression
- Security headers
- Health check endpoint без rate limit
- SSL/TLS готово (закомментировано)

Для включения SSL:

1. Получите сертификаты (Let's Encrypt, Cloudflare, etc.)
2. Поместите их в `./ssl/`
3. Раскомментируйте HTTPS секцию в `nginx.conf`
4. Запустите с профилем: `docker-compose --profile with-nginx up -d`

## Release Process

### Создание релиза

1. Убедитесь, что все изменения в `master`/`main`
2. Создайте и push тег версии:

````bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```bash
1. GitHub Actions автоматически:
   - Соберёт бинарники для всех платформ
   - Создаст checksums
   - Соберёт Docker образ с версией
   - Сгенерирует changelog
   - Создаст GitHub Release

### Semantic Versioning

Используйте семантическое версионирование:

- `v1.0.0` - Major версия (breaking changes)
- `v1.1.0` - Minor версия (новые фичи)
- `v1.1.1` - Patch версия (bug fixes)

### Метки для Changelog

Используйте метки (labels) в Pull Requests для автоматического changelog:

- `feature`, `enhancement` → 🚀 Features
- `bug`, `fix` → 🐛 Bug Fixes
- `documentation`, `docs` → 📝 Documentation
- `maintenance`, `refactor`, `chore` → 🔧 Maintenance
- `security` → 🔒 Security

Пример:

```bash
## В PR добавьте метку "feature"
## В PR добавьте метку "feature"
## В changelog появится под секцией "🚀 Features"
## В changelog появится под секцией "🚀 Features"
````

## Мониторинг и Логи

### Просмотр логов

````bash
## Application logs
## Application logs
docker-compose -f docker-compose.prod.yml logs -f app

## PostgreSQL logs
## PostgreSQL logs
docker-compose -f docker-compose.prod.yml logs -f postgres

## Redis logs
## Redis logs
docker-compose -f docker-compose.prod.yml logs -f redis

## Nginx logs
## Nginx logs
docker-compose -f docker-compose.prod.yml logs -f nginx

## Все логи
## Все логи
docker-compose -f docker-compose.prod.yml logs -f
```bash
### Health Checks

Все сервисы имеют health checks:

```bash
## Application
## Application
curl http://localhost:8080/v1/api/health

## Проверка всех контейнеров
## Проверка всех контейнеров
docker-compose -f docker-compose.prod.yml ps
````

### Prometheus Metrics (TODO)

Добавить `/metrics` endpoint для мониторинга:

- Request rate
- Response time
- Error rate
- Database connections
- Redis connections

## Troubleshooting

### CI Fails на этапе Test

**Проблема**: Тесты падают в CI, но проходят локально

**Решение**:

1. Проверьте версию Go в `.github/workflows/ci.yml`
2. Убедитесь, что PostgreSQL/Redis запущены корректно
3. Проверьте переменные окружения в step "Create test .env file"

### Docker Build Fails

**Проблема**: Сборка Docker образа падает

**Решение**:

1. Проверьте `Dockerfile` - версия Go должна быть 1.23+
2. Убедитесь, что все зависимости в `go.mod`
3. Локально: `docker build -t event-api:test .`

### Deploy Fails

**Проблема**: Деплой не запускается

**Решение**:

1. Проверьте Secrets в GitHub Settings
2. Убедитесь, что Environment настроен правильно
3. Проверьте, что ветка `master` или `main`

### Security Scan Fails

**Проблема**: Trivy или GoSec находят уязвимости

**Решение**:

1. Обновите зависимости: `go get -u && go mod tidy`
2. Проверьте отчёт в GitHub Security tab
3. Исправьте критичные уязвимости перед мержем

## Best Practices

### 1. Ветки и PR

- `master`/`main` - production код
- `develop` - staging/development
- `feature/*` - новые фичи
- `bugfix/*` - исправления
- `hotfix/*` - срочные исправления для production

### 2. Commit Messages

Используйте conventional commits:

````text
feat: добавлена отправка SMS через Twilio
fix: исправлена утечка памяти в Redis клиенте
docs: обновлена документация по CI/CD
chore: обновлены зависимости
test: добавлены тесты для AuthService
```bash
### 3. Pull Requests

- Используйте описательные названия
- Добавляйте labels (feature, bug, etc.)
- Заполняйте описание с контекстом
- Ждите прохождения CI перед мержем
- Делайте code review

### 4. Production Deployment

- Всегда делайте бэкап перед деплоем
- Используйте blue-green или canary deployment
- Мониторьте логи после деплоя
- Имейте план rollback
- Тестируйте на staging перед production

### 5. Security

- Регулярно обновляйте зависимости
- Проверяйте Security tab в GitHub
- Используйте secrets для конфиденциальных данных
- Включите 2FA для Docker Hub и GitHub
- Регулярно ротируйте secrets

## Полезные команды

```bash
## Локальный запуск CI тестов
## Локальный запуск CI тестов
make test

## С coverage
## С coverage
make test-coverage

## Lint локально
## Lint локально
golangci-lint run

## Сборка
## Сборка
make build

## Production деплой
## Production деплой
./scripts/deploy.sh production

## Бэкап БД
## Бэкап БД
./scripts/backup.sh

## Восстановление БД
## Восстановление БД
./scripts/restore.sh backups/latest.sql.gz

## Проверка Docker образа
## Проверка Docker образа
docker build -t event-api:test .
docker run --rm event-api:test

## Очистка
## Очистка
docker system prune -a --volumes
````

## Roadmap

- [ ] Интеграция с Prometheus/Grafana для метрик
- [ ] Automated load testing
- [ ] Canary deployments
- [ ] Kubernetes deployment manifests
- [ ] Terraform для инфраструктуры
- [ ] E2E тесты в CI
- [ ] Performance benchmarks в CI
- [ ] Automated security patches
- [ ] Multi-region deployment
- [ ] Database migration rollback support

## Ресурсы

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Docker Documentation](https://docs.docker.com/)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
