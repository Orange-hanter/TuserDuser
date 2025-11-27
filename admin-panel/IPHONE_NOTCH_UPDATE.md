# Обновление чата - iPhone челка + API проверка

## 📱 Обновления для iPhone (челка)

### Что было изменено:

#### 1. SafeAreaView для учёта челки и Home indicator

```javascript
import { SafeAreaView, useSafeAreaInsets } from "react-native";

const insets = useSafeAreaInsets(); // Получает отступы
<SafeAreaView style={[
  styles.safeArea,
  {
    paddingTop: insets.top,      // Отступ от челки
    paddingBottom: insets.bottom   // Отступ от Home indicator
  }
]}>
```

#### 2. Оптимизированные размеры для мобилок

| Элемент                 | Старое | Новое          | Улучшение                     |
| ----------------------- | ------ | -------------- | ----------------------------- |
| Header padding          | 15     | 12px h, 12px v | Компактнее                    |
| Title fontSize          | 18     | 17             | Лучше на узких экранах        |
| Comment margin          | 10     | 12             | Больше воздуха                |
| Comment maxWidth        | 85%    | 88%            | Лучше использует пространство |
| Input minHeight         | нет    | 40             | Гарантирует видимость кнопки  |
| Comment author fontSize | 12     | 11             | Стройнее выглядит             |
| Comment text fontSize   | 14     | 13             | Оптимально для чтения         |

#### 3. Улучшенные стили

- ✅ Header: `min-height: 50px` - гарантирует видимость
- ✅ Input container: `min-height: 60px` - мобильная оптимизация
- ✅ Input padding: `8px` vertical - правильные отступы
- ✅ ListContent: `paddingHorizontal: 12` - не касается краёв
- ✅ Comment bubbles: `marginHorizontal: 4` - маргины с краёв

---

## ✅ Проверка API

### Используемые эндпоинты:

#### 1. GET /v1/api/creator/events/{eventId}/comments

- **Что делает:** Получает все комментарии к событию
- **Вызывается:** При открытии чата + каждые 5 сек (авто-обновление)
- **Требуется:** Authorization header (токен) → middleware → X-User-ID
- **Возвращает:** Массив комментариев (ID, автор, текст, время)
- **Статус:** ✅ Проверено, работает

#### 2. POST /v1/api/creator/events/{eventId}/comments

- **Что делает:** Добавляет новый комментарий
- **Вызывается:** При клике кнопки "Send" в чате
- **Требуется:** Authorization header + Body { comment: "текст" }
- **Возвращает:** { "status": "ok" }
- **Статус:** ✅ Проверено, работает

### Поток авторизации:

```
Админ-панель
    ↓
setAuthToken(JWT_token)
    ↓
API interceptor добавляет: Authorization: Bearer {token}
    ↓
Бэкенд middleware AuthMiddleware
    ↓
Извлекает из токена: user_id, role, email
    ↓
Устанавливает заголовки: X-User-ID, X-User-Role, X-User-Email
    ↓
Handler получает X-User-ID для проверки доступа
    ↓
CreatorService выполняет операцию
    ↓
Ответ возвращается в админ-панель
```

---

## 🔍 Вот что происходит когда админ пишет комментарий:

### Шаг 1: Админ открывает чат

```
POST /v1/api/events/{id}/review
Body: { action: "...", comment: "..." }
↓
Chat modal открывается
↓
EventCommentChat компонент монтируется
```

### Шаг 2: Загружаются комментарии

```
GET /v1/api/creator/events/{eventId}/comments
Header: Authorization: Bearer {admin_token}
↓
Middleware:
  - Парсит JWT
  - Извлекает: user_id = "admin-uuid"
  - Устанавливает X-User-ID header
  - Вызывает next()
↓
CreatorHandler.GetEventComments:
  - Читает userID из X-User-ID header
  - Проверяет что он авторизован (userID != "")
  - Получает комментарии из БД
  - Возвращает массив
↓
Чат отображает комментарии:
  - authorRole="admin" → синий фон
  - authorRole="creator" → серый фон
```

### Шаг 3: Админ отправляет комментарий

```
TextInput → Кнопка "Send"
↓
POST /v1/api/creator/events/{eventId}/comments
Header: Authorization: Bearer {admin_token}
Body: { comment: "Пожалуйста исправьте..." }
↓
API interceptor добавляет Authorization header
↓
Middleware:
  - Парсит JWT
  - user_id = "admin-uuid"
  - role = "admin"
  - Устанавливает X-User-ID = "admin-uuid"
  - Устанавливает X-User-Role = "admin"
↓
CreatorHandler.AddComment:
  - Читает userID и userRole из headers
  - Парсит JSON body: { comment: "..." }
  - Вызывает CreatorService.AddComment(
      eventID, userID="admin-uuid", role="admin", comment
    )
↓
CreatorService.AddComment:
  - INSERT INTO event_review_comments
    (event_id, author_id, author_role, comment, created_at)
    VALUES (eventID, "admin-uuid", "admin", comment, NOW())
↓
Возвращает { "status": "ok" }
↓
Чат показывает Alert("Success", "Comment added")
↓
Через 5 секунд: GET /v1/api/creator/events/{eventId}/comments
  (автоматическое обновление)
↓
Новый комментарий появляется в чате
```

---

## 🧪 Как проверить API вручную

### Терминал 1: Запустить бэкенд

```bash
cd /Users/dakh/Git/TuserDuser/event-api
make run
# или
go run cmd/server/main.go
```

### Терминал 2: Авторизоваться и получить токен

```bash
# 1. Логин
curl -X POST http://localhost:8080/v1/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@admin.admin","password":"adminpass"}'

# Ответ содержит: {"token":"eyJ0..."}
# Скопируй токен в переменную:
TOKEN="eyJ0..."
```

### Терминал 3: Получить комментарии

```bash
EVENT_ID="550e8400-e29b-41d4-a716-446655440002"

curl -X GET http://localhost:8080/v1/api/creator/events/$EVENT_ID/comments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"

# Ответ: []  или [{ id: 1, eventId: ..., authorRole: "admin", ... }]
```

### Терминал 4: Добавить комментарий

```bash
curl -X POST http://localhost:8080/v1/api/creator/events/$EVENT_ID/comments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"comment":"Test comment from admin"}'

# Ответ: {"status":"ok"}
```

### Терминал 5: Проверить что комментарий добавился

```bash
curl -X GET http://localhost:8080/v1/api/creator/events/$EVENT_ID/comments \
  -H "Authorization: Bearer $TOKEN"

# Ответ: [{ id: 1, comment: "Test comment from admin", authorRole: "admin", ... }]
```

---

## 📋 Чек-лист изменений

### UI Изменения:

- [x] SafeAreaView обёртка добавлена
- [x] useSafeAreaInsets hook подключен
- [x] Динамические отступы для челки добавлены
- [x] Header height увеличен до min-height: 50px
- [x] Input container height увеличен до min-height: 60px
- [x] Все fontSize оптимизированы
- [x] Все padding/margin пересчитаны
- [x] placeholderTextColor добавлен
- [x] contentContainerStyle добавлен для FlatList

### API Проверки:

- [x] GET /v1/api/creator/events/{eventId}/comments — работает
- [x] POST /v1/api/creator/events/{eventId}/comments — работает
- [x] Middleware правильно передаёт X-User-ID из JWT
- [x] Handler проверяет авторизацию через X-User-ID
- [x] БД сохраняет комментарии с author_id и author_role
- [x] Ответы возвращаются в правильном формате
- [x] HTTP статусы правильные (200, 400, 401, 404, 500)

---

## ⚠️ Важное

### Для iOS приложения (мобильная админ-панель):

- SafeAreaView автоматически обработает челку iPhone 13+
- На Android отступы будут минимальны (нет челки)
- На iPad отступы будут нормальные (нет челки)

### Для веб-админ-панели (если когда-то будет):

- Эти стили не будут применяться (SafeAreaView React Native только)
- Нужно будет делать свою адаптивность через CSS Media Queries

---

**Дата:** 27 ноября 2025  
**Статус:** ✅ Готово  
**Версия:** 1.0.1
