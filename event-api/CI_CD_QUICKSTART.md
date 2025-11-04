# CI/CD Quick Start Guide

Быстрое руководство по настройке CI/CD для проекта Event API.

## 🚀 Быстрый старт

### 1. GitHub Actions включен автоматически

После push в GitHub, CI/CD workflows будут запускаться автоматически:

- ✅ **Push в `master`/`main`** → Полный CI/CD + деплой на production
- ✅ **Push в `develop`** → Staging деплой
- ✅ **Pull Request** → Lint + Test + Build
- ✅ **Tag `v*.*.*`** → Release с бинарниками

### 2. Настройка Secrets (5 минут)

Перейдите в **Settings → Secrets and variables → Actions** и добавьте:

#### Минимальные (для Docker Hub):
```
DOCKER_USERNAME = your_dockerhub_username
DOCKER_PASSWORD = your_dockerhub_password_or_token
```

#### Для SSH деплоя:
```
SSH_HOST = your.server.com
SSH_USERNAME = deploy
SSH_PRIVATE_KEY = -----BEGIN OPENSSH PRIVATE KEY-----...
SSH_PORT = 22
```

### 3. Раскомментируйте деплой

В файле `.github/workflows/ci.yml` в job `deploy`:

```yaml
# Было (закомментировано):
# - name: Login to Docker Hub
#   uses: docker/login-action@v3

# Стало (раскомментируйте):
- name: Login to Docker Hub
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKER_USERNAME }}
    password: ${{ secrets.DOCKER_PASSWORD }}
```

То же самое для других шагов деплоя.

### 4. Локальное тестирование

Проверьте, что всё работает локально:

```bash
# Запустить проверки как в CI
make ci-test

# Или по отдельности
make lint
make test
make build
```

### 5. Первый деплой

```bash
# Commit и push
git add .
git commit -m "feat: настроен CI/CD"
git push origin master

# Смотрите прогресс в GitHub Actions tab
```

## 📋 Что включено

### Workflows

1. **CI Pipeline** (`.github/workflows/ci.yml`)
   - Lint код
   - Запуск тестов с PostgreSQL и Redis
   - Сборка Docker образа
   - Security scan
   - Деплой на production

2. **Staging Pipeline** (`.github/workflows/staging.yml`)
   - Быстрые тесты
   - Деплой на staging сервер

3. **Release Pipeline** (`.github/workflows/release.yml`)
   - Сборка для Linux, macOS, Windows
   - Docker образ с версией
   - Автоматический changelog
   - GitHub Release

### Локальные скрипты

- `scripts/deploy.sh` - Деплой на сервер
- `scripts/backup.sh` - Бэкап базы данных
- `scripts/restore.sh` - Восстановление из бэкапа

### Конфигурация

- `.golangci.yml` - Настройки линтера
- `docker-compose.prod.yml` - Production стек
- `nginx.conf` - Nginx reverse proxy
- `.env.*.example` - Примеры env файлов

## 🎯 Типичные сценарии

### Новая фича

```bash
git checkout -b feature/new-feature
# ... делаете изменения ...
git commit -m "feat: добавлена новая фича"
git push origin feature/new-feature
# Создайте PR → CI проверит автоматически
```

### Hotfix для production

```bash
git checkout -b hotfix/critical-bug
# ... исправляете ...
git commit -m "fix: критическая ошибка исправлена"
git push origin hotfix/critical-bug
# PR → мерж в master → автодеплой
```

### Релиз

```bash
# Убедитесь что всё в master
git checkout master
git pull

# Создайте тег
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Release создастся автоматически
```

### Staging тест

```bash
git checkout develop
git merge feature/my-feature
git push origin develop
# Автоматически задеплоится на staging
```

## 🔧 Кастомизация

### Изменить версию Go

В `.github/workflows/ci.yml`:
```yaml
env:
  GO_VERSION: '1.25.0'  # ← измените здесь
```

### Добавить новый шаг в CI

В `.github/workflows/ci.yml` добавьте новый step:
```yaml
- name: My custom step
  run: |
    echo "Doing something"
    make my-custom-command
```

### Изменить метод деплоя

Замените SSH деплой на свой метод в job `deploy`:
- Kubernetes: используйте `kubectl` или Helm
- AWS: используйте `aws-actions`
- GCP: используйте `google-github-actions`

## 🐛 Troubleshooting

### CI падает на тестах

```bash
# Запустите локально
make ci-test

# Если локально проходит, проверьте:
# - Версия Go в CI
# - PostgreSQL/Redis версии
# - Переменные окружения
```

### Деплой не запускается

Проверьте:
1. Secrets добавлены в GitHub
2. Environment "production" создан
3. Код в ветке `master` или `main`
4. Шаги деплоя раскомментированы

### Docker build fails

```bash
# Проверьте локально
docker build -t event-api:test .

# Если локально работает:
# - Проверьте .dockerignore
# - Убедитесь что go.mod/go.sum committed
```

## 📚 Дополнительные ресурсы

- [Полная документация CI/CD](./CI_CD.md)
- [SMS Service документация](./SMS_SERVICE.md)
- [Redis миграция](./docs/REDIS_MIGRATION_SUMMARY.md)

## ✅ Checklist перед production

- [ ] Secrets добавлены в GitHub
- [ ] Production environment настроен
- [ ] `.env.production` файл создан на сервере
- [ ] Docker Hub / Container Registry настроен
- [ ] SSH доступ к production серверу работает
- [ ] Бэкап базы данных настроен
- [ ] Мониторинг настроен
- [ ] SSL сертификаты установлены
- [ ] Nginx reverse proxy настроен
- [ ] Тесты проходят локально
- [ ] Staging деплой успешен

## 🆘 Помощь

Если что-то не работает:

1. Проверьте логи в GitHub Actions tab
2. Запустите `make ci-test` локально
3. Проверьте [CI_CD.md](./CI_CD.md) для деталей
4. Проверьте GitHub Actions логи для конкретной ошибки

## 🎉 Готово!

Теперь у вас настроен полноценный CI/CD pipeline с автоматическим:
- ✅ Тестированием
- ✅ Линтингом
- ✅ Security сканированием
- ✅ Деплоем на staging и production
- ✅ Релизами с бинарниками

Каждый push проверяется, каждый релиз автоматизирован! 🚀
