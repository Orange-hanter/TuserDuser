# 📋 Roadmap Развития Проекта TuserDuser

## ✅ Завершено

### Backend APIs

- [x] **Event Participants API** (27 Nov 2025)
  - GET `/v1/api/events/{event_id}/participants` - получить список участников
  - Таблица БД `event_registrations` с полями: user_id, public_name, avatar_url, status
  - Оптимизировано: индексы для быстрого поиска
  - Документация: `/DOC/API_Event_Participants.md`
  - Тестирование: `/DOC/API_Event_Participants_Testing.md`
  - ✅ Скомпилировано: 32.8 MB binary

### Quick Wins Completed (1 день)

- [x] **Автоскролл в чате** (30м)
  - Добавлен ref flatListRef для FlatList
  - Auto-scroll при получении новых комментариев
  - Auto-scroll при отправке комментариев
  - onContentSizeChange hook для реального времени
  - File: `admin-panel/src/components/EventCommentChat.js`

- [x] **Фильтры в админ-панели** (1-2ч)
  - Фильтр по статусам событий (pending, approved, etc)
  - Фильтр по типам событий (concert, exhibition, film)
  - FilterBar UI компонент с кнопками
  - Toggle фильтрации и "Clear Filters"
  - getFilteredEvents() функция
  - getUniqueStatuses() и getUniqueTypes() helpers
  - File: `admin-panel/src/screens/PendingEventsScreen.js`

- [x] **Dark Mode** (2-3ч)
  - ThemeContext с light/dark темами
  - Complete color palette (20+ colors)
  - ThemeProvider wrapper в App.js
  - useTheme hook для использования в компонентах
  - Toggle button в HomeScreen (☀️ / 🌙)
  - Files: `admin-panel/src/context/ThemeContext.js`, `admin-panel/App.js`

- [x] **Redis Monitoring (Prometheus)** (3-4ч)
  - RedisMetrics struct с 13+ метриками
  - Command counters (SET, GET, DEL, etc)
  - Command duration histograms
  - Memory usage и key count gauges
  - Connection tracking
  - Discovery queue metrics
  - Error rate tracking
  - Setup documentation с примерами
  - Files: `event-api/internal/metrics/prometheus.go`,
    `event-api/internal/handlers/metrics.go`,
    `event-api/DOC/PROMETHEUS_SETUP.md`

---

## 🎯 Приоритет 1: Критические улучшения (1-2 недели)

### Backend

- [ ] **Оптимизация Discovery Engine**
  - Добавить Redis Streams для аналитики вместо List
  - Кешировать отфильтрованные очереди в Redis
  - Добавить Bloom Filter для быстрой проверки исключённых событий
  - Задача: `internal/discovery/` - O(1) lookup excluded events

- [ ] **Увеличение throughput**
  - Добавить connection pooling для database
  - Добавить batch operations для bulk inserts
  - Оптимизировать N+1 queries
  - Задача: `internal/database/` + `internal/service/`

- [ ] **Мониторинг Redis**
  - Добавить Prometheus metrics для Redis (memory, operations)
  - Настроить alerts на использование памяти > 80%
  - Dashboard в Grafana
  - Задача: `internal/metrics/` + Prometheus integration

### Admin Panel

- [ ] **Улучшение UI чата**
  - Автоскролл к последнему сообщению
  - Индикатор "печатает..." (typing indicator)
  - Кнопка "Request Revision" прямо из чата (без модального окна)
  - Search по комментариям
  - Задача: `admin-panel/src/components/EventCommentChat.js`

- [ ] **Фильтры и сортировка**
  - Фильтр по статусам событий (pending, needs_revision, approved)
  - Фильтр по типам событий (concert, exhibition, film)
  - Сортировка по дате создания/обновления
  - Задача: `admin-panel/src/screens/PendingEventsScreen.js`

- [ ] **Bulk actions**
  - Одобрить/отклонить несколько событий сразу
  - Массовое изменение статуса
  - Экспорт данных в CSV
  - Задача: `admin-panel/src/components/`

---

## 🎯 Приоритет 2: Функциональность (2-3 недели)

### Backend

- [ ] **Real-time уведомления**
  - WebSocket для live updates (когда админ пишет, создатель видит)
  - Использовать Redis Pub/Sub
  - Notifications in admin-panel и creator app
  - Задача: `internal/handlers/websocket.go` + Redis

- [ ] **Full-text search**
  - Поиск по названию события, описанию
  - Elasticsearch или PostgreSQL FTS
  - Индексирование при создании события
  - Задача: `internal/search/`

- [ ] **Event analytics**
  - Трекинг: сколько раз событие было shown/liked/booked
  - Конверсия: like → book
  - Популярные события по категориям
  - Задача: `internal/analytics/` + `internal/models/analytics.go`

- [ ] **Экспорт и импорт**
  - Export events в JSON/CSV
  - Import events из CSV
  - Backup и restore database
  - Задача: `internal/handlers/export.go` + `internal/handlers/import.go`

### Frontend (Admin Panel)

- [ ] **Dashboard с аналитикой**
  - Статистика по событиям (всего, pending, approved)
  - Конверсия like → book
  - Популярные категории
  - Активные пользователи
  - Задача: новый экран `admin-panel/src/screens/AnalyticsScreen.js`

- [ ] **Event editor**
  - Создание/редактирование событий из админ-панели
  - Rich text editor для описания
  - Upload фото/видео
  - Задача: `admin-panel/src/components/EventEditor.js`

- [ ] **User management**
  - Список всех пользователей
  - Бан/разбан пользователя
  - Просмотр истории пользователя
  - Задача: `admin-panel/src/screens/UsersScreen.js`

### Frontend (Creator App)

- [ ] **Creator dashboard**
  - Мои события (созданые, активные, завершённые)
  - Статистика по событиям
  - Уведомления от админа
  - Задача: `admin-panel/src/screens/CreatorDashboard.js`

- [ ] **Event creation form**
  - Форма создания события с валидацией
  - Выбор категории, даты, времени
  - Upload фото
  - Сохранение как draft
  - Задача: новый flow в admin-panel

---

## 🎯 Приоритет 3: Масштабирование (3-4 недели)

### Infrastructure

- [ ] **Redis Sentinel**
  - High Availability для Redis
  - Automatic failover
  - Production-ready setup
  - Задача: `docker-compose.prod.yml` + Sentinel config

- [ ] **PostgreSQL Replication**
  - Master-Slave репликация
  - Read replicas для analytics
  - Backup strategy
  - Задача: `docker-compose.prod.yml`

- [ ] **Load Balancing**
  - Nginx load balancer конфиг
  - Health checks для servers
  - Auto-scaling rules
  - Задача: `nginx/nginx-lb.conf`

- [ ] **CI/CD Improvements**
  - Automated tests на каждый push
  - Code coverage > 80%
  - Automated deployment на staging
  - Задача: `.github/workflows/`

### Performance

- [ ] **Caching Strategy**
  - HTTP caching headers (ETag, Cache-Control)
  - CDN для статики
  - Query result caching
  - Задача: `internal/middleware/cache.go`

- [ ] **Database Optimization**
  - Индексирование на frequently queried columns
  - Query планирование и анализ
  - Архивирование старых данных
  - Задача: `internal/migrations/`

- [ ] **API Rate Limiting**
  - Limit по IP
  - Limit по user
  - Exponential backoff
  - Задача: `internal/middleware/ratelimit.go`

---

## 🎯 Приоритет 4: Качество кода (Ongoing)

### Testing

- [ ] **Unit Tests**
  - Покрытие Discovery engine (goal: 95%)
  - Покрытие Repository слоя
  - Покрытие Service слоя
  - Задача: `*_test.go` во всех пакетах

- [ ] **Integration Tests**
  - Tests с real Redis
  - Tests с real PostgreSQL
  - API endpoint tests
  - Задача: `internal/integration_test.go`

- [ ] **Load Testing**
  - k6 или artillery tests
  - Тестирование на 1000+ concurrent users
  - Bottleneck identification
  - Задача: `load-tests/`

- [ ] **E2E Tests**
  - Cypress/Selenium tests для admin-panel
  - User flow testing
  - Cross-browser testing
  - Задача: `e2e-tests/`

### Documentation

- [ ] **API Documentation**
  - Swagger/OpenAPI cleanup
  - Example requests/responses
  - Error codes documentation
  - Задача: Swagger comments в handlers

- [ ] **Architecture Documentation**
  - C4 диаграммы (уже есть, обновить)
  - Sequence диаграммы для основных flows
  - Data flow diagrams
  - Задача: `Docs/` уточнение

- [ ] **Development Guide**
  - Setup guide для новых разработчиков
  - Common tasks
  - Troubleshooting
  - Задача: `DEVELOPMENT.md`

- [ ] **Deployment Guide**
  - Development, staging, production setup
  - Environment variables
  - Migration steps
  - Задача: `DEPLOYMENT.md`

### Code Quality

- [ ] **Refactoring**
  - Уменьшить cyclomatic complexity в discovery/engine.go
  - Выделить business logic из handlers
  - Уменьшить код дублирование
  - Задача: Code review + metrics

- [ ] **Error Handling**
  - Custom error types
  - Proper error codes и messages
  - Error logging и monitoring
  - Задача: `internal/errors/`

- [ ] **Security**
  - SQL injection prevention audit
  - XSS prevention audit
  - Authentication/authorization audit
  - Задача: Security review всех handlers

---

## 🎯 Приоритет 5: Расширение функциональности (4-6 недель)

### Features

- [ ] **Advanced Filtering**
  - Фильтр по геолокации
  - Фильтр по цене
  - Фильтр по рейтингу
  - Saved filters/bookmarks
  - Задача: `internal/discovery/filters.go`

- [ ] **Recommendations**
  - ML-based event recommendations
  - Collaborative filtering
  - Content-based recommendations
  - Задача: `internal/recommendations/`

- [ ] **Social Features**
  - Шеринг события в социальные сети
  - Комментарии на события
  - Рейтинг события
  - Задача: `internal/social/`

- [ ] **Gamification**
  - Badges за активность
  - Leaderboard лучших посетителей
  - Rewards program
  - Задача: `internal/gamification/`

- [ ] **Integrations**
  - Google Calendar sync
  - Spotify/SoundCloud integration
  - Instagram integration
  - Задача: `internal/integrations/`

---

## 🎯 Приоритет 6: Mobile Apps (3-4 недели)

### React Native Creator App

- [ ] **Event Management**
  - Создание события через mobile
  - Редактирование события
  - Удаление события
  - Задача: новый React Native app

- [ ] **Notifications**
  - Push notifications
  - In-app notifications
  - Notification preferences
  - Задача: React Native Notifications

- [ ] **Analytics**
  - Личная статистика событий
  - График активности
  - Insights
  - Задача: React Native Charts

---

## 📊 Метрики для отслеживания

### Performance

- [ ] API response time < 200ms (p95)
- [ ] Discovery engine query < 100ms
- [ ] Memory usage < 500MB в normal mode
- [ ] Redis memory < 1GB

### Reliability

- [ ] Uptime > 99.9%
- [ ] Error rate < 0.1%
- [ ] Database connection pool efficiency > 95%

### Quality

- [ ] Code coverage > 80%
- [ ] Cyclomatic complexity avg < 10
- [ ] Technical debt < 5%

---

## 🔄 Текущий статус

### Завершено ✅

- [x] Chat между админом и создателем события
- [x] iPhone notch support в чате
- [x] Redis интеграция для Discovery
- [x] API endpoints для комментариев

### В разработке 🚀

- [ ] Оптимизация Discovery queries

### Требует внимания ⚠️

- [ ] Мониторинг Redis
- [ ] Full-text search для событий
- [ ] Analytics dashboard

---

## 📅 Рекомендуемый график

**Week 1-2:** Приоритет 1 (Critical fixes)  
**Week 3-4:** Приоритет 2 (Core features)  
**Week 5-6:** Приоритет 3 (Scaling)  
**Ongoing:** Приоритет 4 (Quality)  
**Month 2:** Приоритет 5 (Advanced features)  
**Month 3:** Приоритет 6 (Mobile)

---

## 🚀 Quick Wins (можно сделать за 1 день)

1. Автоскролл в чате (30 мин)
2. Фильтры в PendingEventsScreen (1-2 часа)
3. Dark mode для admin-panel (2-3 часа)
4. Swagger UI improvements (1-2 часа)
5. Redis monitoring dashboard (3-4 часа)

---

## 💡 Идеи для обсуждения

- Использовать GraphQL вместо REST?
- Microservices architecture?
- Event sourcing pattern?
- CQRS pattern?
- Kubernetes для deployment?
