# GitHub Secrets Setup Guide

Пошаговая инструкция по настройке GitHub Secrets для CI/CD.

## Где добавить Secrets

1. Откройте ваш репозиторий на GitHub
2. Перейдите в **Settings** (верхняя панель)
3. В левом меню выберите **Secrets and variables** → **Actions**
4. Нажмите **New repository secret**

## Обязательные Secrets

### Для Docker Hub

Если вы хотите публиковать Docker образы в Docker Hub:

#### DOCKER_USERNAME

````text
Value: ваш_username_на_dockerhub
Example: johndoe
```bash
#### DOCKER_PASSWORD

```text
Value: ваш_пароль_или_access_token
Рекомендация: Используйте Access Token вместо пароля
Как получить: Docker Hub → Account Settings → Security → New Access Token
````

### Для SSH деплоя

Если вы деплоите на свой сервер через SSH:

#### SSH_HOST

````text
Value: IP_адрес_или_домен_вашего_сервера
Example: 123.45.67.89 или server.example.com
```bash
#### SSH_USERNAME

```text
Value: имя_пользователя_на_сервере
Example: deploy или ubuntu
````

#### SSH_PRIVATE_KEY

````text
Value: приватный SSH ключ (весь текст)
Example:
-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAACmFlczI1Ni1jdHIAAAAG...
...весь ключ...
-----END OPENSSH PRIVATE KEY-----

Как получить:
1. Сгенерируйте новую SSH пару ключей:
   ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/deploy_key

2. Добавьте публичный ключ на сервер:
   ssh-copy-id -i ~/.ssh/deploy_key.pub user@server.com

3. Скопируйте приватный ключ:
   cat ~/.ssh/deploy_key

4. Вставьте весь текст (включая BEGIN и END строки) в Secret
```bash
**SSH_PORT** (опционально)

```text
Value: порт SSH (по умолчанию 22)
Example: 22 или 2222
````

## Опциональные Secrets

### Для Codecov (отслеживание покрытия кода)

#### CODECOV_TOKEN

````text
Value: токен из codecov.io
Как получить:
1. Зарегистрируйтесь на https://codecov.io
2. Подключите ваш GitHub репозиторий
3. Скопируйте Upload Token
```bash
### Для Staging окружения

#### STAGING_SSH_HOST

```text
Value: IP_или_домен_staging_сервера
````

#### STAGING_SSH_USERNAME

````text
Value: имя_пользователя
```bash
#### STAGING_SSH_PRIVATE_KEY

```text
Value: приватный SSH ключ для staging
````

#### STAGING_SSH_PORT

````text
Value: порт SSH (по умолчанию 22)
```bash
### Для уведомлений в Slack/Discord (опционально)

#### SLACK_WEBHOOK_URL

```text
Value: webhook URL из Slack
Как получить: Slack → Apps → Incoming Webhooks
````

#### DISCORD_WEBHOOK_URL

````text
Value: webhook URL из Discord
Как получить: Discord → Server Settings → Integrations → Webhooks
```bash
## Настройка Environments

### Production Environment

1. Перейдите в **Settings** → **Environments**
2. Нажмите **New environment**
3. Имя: `production`
4. Настройте:
   - ✅ **Required reviewers** - добавьте членов команды для подтверждения
   - ✅ **Wait timer** - задержка перед деплоем (опционально)
   - ✅ **Deployment branches** - только `main` или `master`

### Staging Environment

Повторите для staging окружения:

1. Имя: `staging`
2. Deployment branches: только `develop`

## Проверка настройки

После добавления всех secrets:

1. Сделайте тестовый commit и push:

```bash
git add .
git commit -m "test: проверка CI/CD"
git push origin master
````

1. Перейдите во вкладку **Actions** в GitHub
2. Проверьте что workflow запустился
3. Если есть ошибки, проверьте логи

## Безопасность

### ✅ Хорошие практики

1. **Используйте Access Tokens вместо паролей**
   - Docker Hub: используйте Access Token
   - GitHub: используйте Personal Access Token с минимальными правами

2. **Ротируйте secrets регулярно**
   - Меняйте ключи каждые 3-6 месяцев
   - После ухода члена команды

3. **Минимальные права доступа**
   - SSH ключи: только для деплоя, не root
   - Docker Hub: только push права

4. **Отдельные ключи для разных окружений**
   - Production и Staging должны иметь разные ключи

### ❌ Чего НЕ делать

1. ❌ Не коммитьте secrets в код
2. ❌ Не используйте одинаковые ключи везде
3. ❌ Не давайте root доступ через SSH ключи
4. ❌ Не используйте слабые пароли

## Тестирование без Production

Если вы хотите протестировать CI без настройки деплоя:

1. Оставьте только базовые secrets (например, DOCKER_USERNAME/PASSWORD)
2. В `.github/workflows/ci.yml` закомментируйте job `deploy`
3. CI будет выполнять: lint → test → build → security scan
4. Когда будете готовы к деплою, раскомментируйте и добавьте SSH secrets

## Troubleshooting

### Ошибка: "Secret not found"

**Проблема**: Workflow не может найти secret
**Решение**:

1. Проверьте правильность имени secret
2. Убедитесь что secret добавлен в репозиторий, а не в организацию
3. Проверьте что secret не истёк (для токенов с временем жизни)

### Ошибка: "Permission denied" при SSH

**Проблема**: Не может подключиться к серверу
**Решение**:

1. Проверьте что публичный ключ добавлен на сервер
2. Проверьте права на ~/.ssh/authorized_keys (должно быть 600)
3. Попробуйте подключиться вручную с этим ключом

### Ошибка: Docker login failed

**Проблема**: Не может залогиниться в Docker Hub
**Решение**:

1. Проверьте правильность username
2. Создайте новый Access Token в Docker Hub
3. Убедитесь что копировали токен полностью

## Готово

После настройки всех необходимых secrets ваш CI/CD pipeline готов к работе! 🎉

Дополнительная помощь:

- [GitHub Secrets Documentation](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [GitHub Environments Documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment)
