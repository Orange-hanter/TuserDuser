# CI/CD Setup Checklist

Используйте этот чеклист для настройки CI/CD после интеграции.

## 📋 Pre-deployment Checklist

### Phase 1: Базовая проверка (5 минут)

- [ ] Все файлы закоммичены в Git
- [ ] `.env` файл в `.gitignore` (не коммитить!)
- [ ] Проект собирается локально: `make build`
- [ ] Тесты проходят локально: `make test`
- [ ] Линтер проходит локально: `make lint` (требует установки golangci-lint)

### Phase 2: GitHub Setup (10 минут)

#### Repository Settings

- [ ] Push код в GitHub репозиторий
- [ ] Вкладка **Actions** включена (Settings → Actions → General → Allow all actions)

#### Secrets Configuration

Перейдите в **Settings → Secrets and variables → Actions**

**Минимальные (для базового CI):**

- [ ] Ничего не требуется! Lint, Test, Build работают без secrets

**Для Docker Hub (опционально):**

- [ ] `DOCKER_USERNAME` - ваш Docker Hub username
- [ ] `DOCKER_PASSWORD` - Docker Hub password или access token

**Для Production Deploy:**

- [ ] `SSH_HOST` - IP или домен production сервера
- [ ] `SSH_USERNAME` - username для SSH
- [ ] `SSH_PRIVATE_KEY` - приватный SSH ключ
- [ ] `SSH_PORT` - порт SSH (обычно 22)

**Для Staging Deploy:**

- [ ] `STAGING_SSH_HOST`
- [ ] `STAGING_SSH_USERNAME`
- [ ] `STAGING_SSH_PRIVATE_KEY`
- [ ] `STAGING_SSH_PORT`

**Опционально:**

- [ ] `CODECOV_TOKEN` - для отслеживания coverage

#### Environments Configuration

Перейдите в **Settings → Environments**

**Production Environment:**

- [ ] Создан environment с именем `production`
- [ ] Добавлены reviewers (рекомендуется)
- [ ] Настроен deployment branch: только `master` или `main`
- [ ] Wait timer настроен (опционально)

**Staging Environment:**

- [ ] Создан environment с именем `staging`
- [ ] Настроен deployment branch: только `develop`

### Phase 3: Workflow Configuration (15 минут)

#### `.github/workflows/ci.yml`

**Для Docker Hub deploy:**

- [ ] Раскомментирован блок "Login to Docker Hub"
- [ ] Раскомментирован блок "Build and push Docker image"
- [ ] Проверен `DOCKER_IMAGE` name в env (или используйте ваш)

**Для SSH deploy:**

- [ ] Раскомментирован блок "Deploy to server via SSH"
- [ ] Обновлен путь `/path/to/app` на реальный путь на сервере
- [ ] Проверены команды деплоя

**Или альтернативный метод:**

- [ ] Kubernetes deployment настроен
- [ ] Cloud provider (AWS/GCP/Azure) настроен
- [ ] Другой метод деплоя настроен

#### `.github/workflows/staging.yml`

- [ ] Раскомментирован deploy блок (если используете staging)
- [ ] Обновлены пути и команды для staging сервера

#### `.github/workflows/release.yml`

**Для Docker Hub:**

- [ ] Раскомментирован блок "Build and push Docker image"
- [ ] Проверен username/repository name

### Phase 4: Server Preparation (20 минут)

#### Production Server

- [ ] Docker установлен
- [ ] Docker Compose установлен
- [ ] SSH доступ работает
- [ ] Публичный SSH ключ добавлен в `~/.ssh/authorized_keys`
- [ ] Пользователь имеет права на Docker (или добавлен в docker group)

**Создайте структуру на сервере:**

```bash
mkdir -p /opt/event-api
cd /opt/event-api
```

- [ ] Директория создана: `/opt/event-api` (или ваш путь)
- [ ] Скопирован файл `.env.production` → `.env`
- [ ] Все переменные в `.env` настроены
- [ ] `docker-compose.prod.yml` скопирован на сервер
- [ ] PostgreSQL volume создан
- [ ] Redis volume создан

#### Staging Server (если используется)

- [ ] Аналогичная настройка как для production
- [ ] Используется `.env.staging`
- [ ] Отдельная база данных

### Phase 5: First Deployment Test (10 минут)

#### Local Testing

```bash
# Запустите проверки локально
make ci-test
make check
make build

# Если есть Docker
make docker-build
```

- [ ] `make ci-test` проходит
- [ ] `make check` проходит (если установлен golangci-lint)
- [ ] `make build` проходит
- [ ] Docker образ собирается (если используете)

#### GitHub Actions Test

```bash
git add .
git commit -m "feat: добавлен CI/CD"
git push origin master
```

- [ ] Push выполнен успешно
- [ ] GitHub Actions workflow запустился
- [ ] Lint job прошел успешно
- [ ] Test job прошел успешно
- [ ] Build job прошел успешно
- [ ] Security job прошел успешно

**Если deploy настроен:**

- [ ] Deploy job запустился
- [ ] Ждет подтверждения (если настроены reviewers)
- [ ] Deploy выполнен успешно
- [ ] Приложение доступно на production URL

### Phase 6: Verification (10 минут)

#### Application Health

- [ ] Health endpoint работает: `curl http://your-server:8080/v1/api/health`
- [ ] Swagger доступен: `http://your-server:8080/swagger/index.html`
- [ ] API endpoints отвечают
- [ ] Database подключена
- [ ] Redis подключен

#### Logs Check

```bash
# На сервере
docker-compose -f docker-compose.prod.yml logs -f app

# Или через Makefile
make prod-logs
```

- [ ] Логи показывают успешный запуск
- [ ] Нет критичных ошибок
- [ ] Database migrations применены
- [ ] Redis соединение установлено

#### Services Status

```bash
docker-compose -f docker-compose.prod.yml ps
```

- [ ] PostgreSQL container running и healthy
- [ ] Redis container running и healthy
- [ ] Application container running и healthy
- [ ] Nginx running (если используется)

### Phase 7: Staging Test (15 минут)

Если используете staging:

```bash
git checkout develop
git merge master
git push origin develop
```

- [ ] Staging workflow запустился
- [ ] Staging deploy выполнен
- [ ] Staging приложение работает
- [ ] Можно тестировать новые фичи на staging

### Phase 8: Release Test (10 минут)

Протестируйте release workflow:

```bash
git checkout master
git tag -a v0.1.0 -m "Initial release with CI/CD"
git push origin v0.1.0
```

- [ ] Release workflow запустился
- [ ] Binaries созданы для всех платформ
- [ ] Checksums сгенерированы
- [ ] Docker image создан с тегом версии
- [ ] Changelog сгенерирован
- [ ] GitHub Release создан
- [ ] Assets загружены в Release

### Phase 9: Backup & Recovery Test (10 минут)

#### Database Backup

```bash
make backup
```

- [ ] Backup создан в `./backups/`
- [ ] Backup сжат (gzip)
- [ ] Размер backup разумный

#### Restore Test (на staging!)

```bash
# НЕ НА PRODUCTION!
make restore FILE=backups/your_backup.sql.gz
```

- [ ] Restore выполнен успешно
- [ ] Данные восстановлены
- [ ] Приложение работает после restore

### Phase 10: Monitoring Setup (опционально, 30+ минут)

- [ ] Prometheus настроен (если используете)
- [ ] Grafana настроена (если используете)
- [ ] Alerting настроен
- [ ] Slack/Discord notifications настроены
- [ ] Uptime monitoring (UptimeRobot, etc.)

## ⚠️ Common Issues

### Issue: Tests fail in CI but pass locally

**Check:**

- [ ] Go version совпадает (local vs CI)
- [ ] PostgreSQL version совпадает
- [ ] Redis version совпадает
- [ ] Environment variables правильные

### Issue: Docker build fails

**Check:**

- [ ] `.dockerignore` настроен правильно
- [ ] `go.mod` и `go.sum` committed
- [ ] Dockerfile Go version актуальна

### Issue: Deploy fails

**Check:**

- [ ] SSH ключ добавлен правильно
- [ ] SSH доступ работает: `ssh user@server`
- [ ] Docker установлен на сервере
- [ ] Пути в deploy script правильные

### Issue: Application crashes after deploy

**Check:**

- [ ] `.env` файл существует на сервере
- [ ] Все переменные окружения заполнены
- [ ] Database доступна
- [ ] Redis доступен
- [ ] Логи: `make prod-logs`

## ✅ Success Criteria

Ваш CI/CD полностью настроен когда:

- ✅ Push в master → автоматический production deploy
- ✅ Push в develop → автоматический staging deploy
- ✅ Tag v*.*.\* → автоматический release с binaries
- ✅ Pull Requests → автоматические тесты
- ✅ Security scanning работает
- ✅ Health checks проходят
- ✅ Backups работают
- ✅ Rollback процедура протестирована
- ✅ Документация актуальна
- ✅ Команда знает как использовать CI/CD

## 📚 Next Steps

После успешной настройки:

1. **Документация:**
   - [ ] Обновите team wiki с ссылками на workflows
   - [ ] Создайте runbook для troubleshooting
   - [ ] Задокументируйте deployment процесс

2. **Optimization:**
   - [ ] Настройте cache для Docker layers
   - [ ] Настройте parallel jobs где возможно
   - [ ] Оптимизируйте test suite

3. **Monitoring:**
   - [ ] Добавьте Prometheus metrics
   - [ ] Настройте Grafana dashboards
   - [ ] Настройте alerting

4. **Security:**
   - [ ] Регулярно обновляйте зависимости
   - [ ] Ротируйте secrets каждые 3-6 месяцев
   - [ ] Review security scan результаты

## 🆘 Need Help?

- 📖 [CI/CD Full Documentation](./CI_CD.md)
- 🚀 [CI/CD Quick Start](./CI_CD_QUICKSTART.md)
- 🔐 [GitHub Secrets Setup](./GITHUB_SECRETS_SETUP.md)
- 💬 Ask in team chat
- 🐛 Check GitHub Actions logs
