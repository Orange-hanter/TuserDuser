Ниже — **информационная модель (информационная схема / ER-диаграмма в текстовом виде)**, составленная на основе всех пользовательских историй. Модель охватывает **сущности, их атрибуты и связи**, необходимые для реализации платформы мероприятий с ролями: **Автор, Пользователь, Админ, Поддержка + Групповые функции, заказы, медиатека, уведомления**.

Данный документ сгенерирован и отвечает целям создания общего представления как комплексная система должна выглядеть. Этот документ используется как шаблон.
---

# 🧩 Информационная модель платформы мероприятий

---

## 📂 Основные сущности

---

### 1. `User` (Пользователь / Автор / Админ / Поддержка)

> **Описание:** Единая сущность для всех ролей. Роль определяется через атрибут `role` или отдельную таблицу ролей, если нужны разрешения.

**Атрибуты:**
- `user_id` (PK) — уникальный ID
- `email` — email (уникальный)
- `phone` — телефон
- `full_name` — ФИО
- `birth_date` — дата рождения → для расчёта возраста
- `gender` — пол (м/ж/другое)
- `avatar_url` — ссылка на аватар
- `role` — enum: `user`, `author`, `admin`, `support`
- `created_at` — дата регистрации
- `last_login` — последний вход
- `is_banned` — флаг бана (boolean)
- `ban_reason` — причина бана (текст)
- `ban_until` — до какого числа (если временный бан)
- `privacy_settings` — JSON: какие данные показывать (телефон, email, возраст и т.д.)
- `onboarding_completed` — прошёл ли онбординг
- `referral_code` — реферальный код пользователя
- `referred_by` — кто пригласил (FK → `user_id`)

**Связи:**
- → `Event` (как автор)
- → `EventParticipant` (как участник)
- → `Group` (как создатель или участник)
- → `Review` (как автор отзыва)
- → `SupportTicket` (как инициатор)
- → `Notification` (получатель)
- → `Media` (загрузил фото/видео)
- → `Order` (сделал заказ)
- → `Bookmark` (добавил в избранное)

---

### 2. `Event` (Мероприятие)

> **Описание:** Основная сущность — событие, созданное автором.

**Атрибуты:**
- `event_id` (PK)
- `author_id` (FK → `User.user_id`)
- `title` — заголовок
- `description` — описание
- `cover_image_url` — обложка
- `location_name` — название места (напр., “Кафе ‘Лето’”)
- `location_lat`, `location_lng` — координаты
- `address` — полный адрес
- `start_datetime`, `end_datetime` — начало и окончание
- `age_restriction` — минимальный возраст (int)
- `is_paid` — boolean
- `price_min`, `price_max` — диапазон цен (если есть)
- `external_registration_url` — ссылка на внешнюю регистрацию
- `status` — enum: `draft`, `pending_moderation`, `approved`, `rejected`, `cancelled`, `completed`
- `rejection_reason` — причина отклонения (если rejected)
- `created_at`, `updated_at`
- `participant_limit` — лимит участников (опционально)
- `supports_group_orders` — поддерживает групповые заказы (boolean)
- `supports_partner_search` — можно искать партнёра (boolean)
- `moderation_notes` — заметки модератора (JSON)

**Связи:**
- ← `User` (автор)
- → `EventTag` (теги мероприятия)
- → `EventParticipant` (участники)
- → `EventFAQ` (вопросы и ответы)
- → `Review` (отзывы)
- → `Media` (медиатека мероприятия)
- → `Notification` (уведомления, связанные с событием)
- → `Order` (заказы, если мероприятие в заведении)
- → `SupportTicket` (жалобы/вопросы по мероприятию)

---

### 3. `EventTag` (Тег мероприятия)

> **Описание:** Метки для категоризации и поиска (напр., “Кино”, “Спорт”, “Бесплатно”).

**Атрибуты:**
- `tag_id` (PK)
- `name` — название тега (уникальное)
- `is_moderated` — прошёл ли модерацию (если создавался пользователем)
- `created_by` (FK → `User.user_id`) — кто создал тег (если пользовательский)
- `created_at`

**Связи:**
- → `EventTagMapping` (связь с мероприятиями)

---

### 4. `EventTagMapping` (Связь мероприятие–тег)

> **Описание:** Многие-ко-многим. Одно мероприятие может иметь несколько тегов.

**Атрибуты:**
- `event_id` (FK)
- `tag_id` (FK)
- (PK: составной — `event_id + tag_id`)

---

### 5. `EventParticipant` (Участник мероприятия)

> **Описание:** Связь между пользователем и мероприятием. Может быть индивидуальным или групповым.

**Атрибуты:**
- `participant_id` (PK)
- `event_id` (FK)
- `user_id` (FK)
- `group_id` (FK → `Group.group_id`, опционально)
- `joined_at` — дата присоединения
- `status` — enum: `confirmed`, `cancelled`, `no_show`
- `has_made_order` — сделал ли заказ (boolean)
- `order_id` (FK → `Order.order_id`, опционально)
- `notifications_enabled` — получает ли уведомления (boolean)

**Связи:**
- ← `User`, ← `Event`, ← `Group`, ← `Order`

---

### 6. `Group` (Группа участников)

> **Описание:** Пользователи могут создавать группы для совместного участия.

**Атрибуты:**
- `group_id` (PK)
- `name` — название группы (напр., “Компания Ивана”)
- `creator_id` (FK → `User.user_id`)
- `event_id` (FK → `Event.event_id`)
- `created_at`
- `member_count` — количество участников (можно вычислять, но для производительности — хранить)
- `is_active` — активна ли группа

**Связи:**
- → `GroupMember` (участники группы)
- ← `EventParticipant` (если группа участвует в мероприятии)

---

### 7. `GroupMember` (Участник группы)

> **Описание:** Связь пользователей с группой.

**Атрибуты:**
- `group_id` (FK)
- `user_id` (FK)
- `joined_at`
- `role_in_group` — enum: `member`, `admin` (если нужно управление внутри группы)
- (PK: `group_id + user_id`)

---

### 8. `EventFAQ` (Вопросы и ответы по мероприятию)

> **Описание:** Раздел Q&A на карточке мероприятия.

**Атрибуты:**
- `faq_id` (PK)
- `event_id` (FK)
- `user_id` (FK → кто задал вопрос)
- `question_text`
- `answer_text` (может быть null, пока не ответил автор)
- `answered_by` (FK → `User.user_id`, кто ответил — автор или модератор)
- `answered_at`
- `is_pinned` — закреплён ли вопрос
- `created_at`, `updated_at`

---

### 9. `Review` (Отзыв о мероприятии)

> **Описание:** Оставляется после завершения события.

**Атрибуты:**
- `review_id` (PK)
- `event_id` (FK)
- `user_id` (FK)
- `rating` — int от 1 до 5
- `comment` — текст отзыва
- `media_url` — фото/видео к отзыву (опционально)
- `created_at`
- `is_moderated` — прошёл ли модерацию
- `moderation_status` — enum: `pending`, `approved`, `rejected`
- `rejection_reason` — если отклонён

**Связи:**
- ← `User`, ← `Event`, ← `Media`

---

### 10. `Media` (Медиа: фото/видео)

> **Описание:** Фото и видео с мероприятий — как от автора, так и от участников.

**Атрибуты:**
- `media_id` (PK)
- `uploader_id` (FK → `User.user_id`)
- `event_id` (FK → `Event.event_id`, опционально)
- `review_id` (FK → `Review.review_id`, опционально)
- `url` — ссылка на файл
- `type` — enum: `photo`, `video`
- `uploaded_at`
- `is_public` — можно ли показывать в общей медиатеке
- `is_moderated` — прошла ли модерацию
- `moderation_status` — enum: `pending`, `approved`, `rejected`
- `moderation_notes` — комментарий модератора

**Связи:**
- ← `User`, ← `Event`, ← `Review`

---

### 11. `Bookmark` (Избранное)

> **Описание:** Пользователь добавляет мероприятие в избранное.

**Атрибуты:**
- `user_id` (FK)
- `event_id` (FK)
- `bookmarked_at`
- (PK: `user_id + event_id`)

---

### 12. `Order` (Заказ в заведении)

> **Описание:** Заказ, сделанный через мероприятие (если поддерживается).

**Атрибуты:**
- `order_id` (PK)
- `event_id` (FK)
- `user_id` (FK → кто сделал заказ)
- `group_id` (FK → если от группы)
- `items` — JSON: список блюд/напитков с количеством и ценой
- `total_amount` — сумма заказа
- `status` — enum: `pending`, `confirmed`, `cancelled`, `completed`
- `created_at`, `updated_at`
- `restaurant_notes` — комментарии для заведения (напр., “без лука”)

**Связи:**
- ← `User`, ← `Event`, ← `Group`, ← `EventParticipant`

---

### 13. `Notification` (Уведомление)

> **Описание:** Push/email уведомления пользователям.

**Атрибуты:**
- `notification_id` (PK)
- `user_id` (FK)
- `event_id` (FK, опционально)
- `group_id` (FK, опционально)
- `type` — enum: `event_reminder`, `event_changed`, `event_cancelled`, `support_reply`, `moderation_status`, `group_invite`, `partner_found`
- `title`, `body` — заголовок и текст
- `sent_at`
- `read_at` — когда прочитано (null, если не прочитано)
- `channel` — enum: `push`, `email`, `in_app`
- `priority` — enum: `low`, `normal`, `high`, `urgent`
- `deep_link` — ссылка для перехода в приложение

---

### 14. `SupportTicket` (Обращение в поддержку)

> **Описание:** Заявка пользователя в службу поддержки.

**Атрибуты:**
- `ticket_id` (PK)
- `user_id` (FK)
- `assigned_to` (FK → `User.user_id`, кто обрабатывает — админ/поддержка)
- `subject` — тема
- `description` — описание проблемы
- `status` — enum: `new`, `in_progress`, `resolved`, `closed`
- `category` — enum: `technical`, `event_related`, `billing`, `complaint`, `other`
- `created_at`, `updated_at`, `resolved_at`
- `screenshot_url` — прикреплённый скриншот
- `internal_notes` — заметки для поддержки

**Связи:**
- ← `User` (инициатор и исполнитель)
- → `SupportMessage` (переписка)

---

### 15. `SupportMessage` (Сообщение в тикете поддержки)

> **Описание:** Диалог внутри обращения.

**Атрибуты:**
- `message_id` (PK)
- `ticket_id` (FK)
- `sender_id` (FK → `User.user_id`)
- `message_text`
- `sent_at`
- `is_internal` — заметка для команды (не видна пользователю)

---

### 16. `AdminActionLog` (Лог действий админа)

> **Описание:** Аудит действий модераторов и админов.

**Атрибуты:**
- `log_id` (PK)
- `admin_id` (FK → `User.user_id`)
- `action_type` — enum: `event_approve`, `event_reject`, `user_ban`, `media_remove`, `ticket_resolve` и т.д.
- `target_id` — ID целевого объекта (event_id, user_id, media_id и т.д.)
- `target_type` — enum: `event`, `user`, `media`, `ticket`
- `details` — JSON: дополнительные данные (причина, старое/новое значение)
- `timestamp`

---

## 🔄 Основные связи (вкратце)

```
User → создает → Event
User → участвует → Event (через EventParticipant)
User → входит в → Group (через GroupMember)
User → оставляет → Review
User → загружает → Media
User → добавляет → Bookmark
User → создаёт → SupportTicket
User → получает → Notification

Event → имеет → EventTag (через EventTagMapping)
Event → имеет → EventFAQ
Event → имеет → Review
Event → имеет → Media
Event → имеет → Order (если поддерживается)
Event → генерирует → Notification

Group → содержит → User (через GroupMember)
Group → участвует → Event (через EventParticipant)

SupportTicket → содержит → SupportMessage

Admin → логирует → AdminActionLog
```

---

## 🧠 Дополнительные рекомендации

### 🔐 Безопасность и приватность
- Хранить согласия на обработку данных (GDPR/152-ФЗ).
- Отдельная таблица `UserConsent`: тип согласия, дата, версия политики.

### 📈 Аналитика
- Таблица `EventAnalytics`: просмотры карточки, клики на “Присоединиться”, отписки — для авторов и админов.
- Таблица `UserBehaviorLog`: действия пользователя (для рекомендаций и персонализации).

### 🌐 Локализация (если нужно)
- Таблицы `EventTranslation`, `TagTranslation` — для мультиязычности.

---

## 🖼️ Визуализация

Если нужно — могу подготовить:
- **ER-диаграмму** в формате Mermaid, PlantUML или описать для Figma/dbdiagram.io.
- **Схему в JSON** для импорта в инструменты проектирования БД.
- **Словарь данных** с описанием каждого поля.

---

✅ Эта информационная модель:
- Покрывает **все пользовательские истории**.
- Поддерживает **роли, группы, заказы, медиа, отзывы, уведомления, поддержку**.
- Гибкая для **расширения** (аналитика, локализация, интеграции).
- Готова к **реализации в любой СУБД** (PostgreSQL, MySQL и др.).

---

Нужна визуальная схема или экспорт в конкретный формат? Готов сделать — просто укажи, какой 👇