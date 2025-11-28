# Задание: Интеграция проверки статуса запроса на повышение роли

## Описание

Реализовать функциональность отслеживания и уведомления пользователя о статусе его запроса на повышение роли в приложении.

## Требования

### 1. API Интеграция

#### Базовые функции в `src/services/api.js`

```javascript
// Уже реализовано:
-requestRole(role, reason) - // POST /v1/api/users/request-role
  getRoleRequestStatus(role) - // GET /v1/api/users/request-role/status?role={role}
  getAllRoleRequests(); // GET /v1/api/users/request-role/all
```

**Требуется:**

- Добавить кэширование результатов на 30 секунд
- Добавить обработку ошибок сети с retry механизмом (до 3 попыток)
- Добавить timeout на 10 секунд для каждого запроса

### 2. Компоненты

#### 2.1 `RoleRequestStatusBadge` (новый компонент)

**Назначение:** Компактный индикатор статуса в заголовке или меню

**Props:**

```javascript
{
  status: 'pending' | 'approved' | 'rejected',
  size: 'small' | 'medium' | 'large',  // default: 'medium'
  showLabel: boolean,                   // default: true
  onPress: () => void                   // optional
}
```

**Функциональность:**

- Отображать иконку и текст статуса в соответствии со статусом
- Цвета:
  - `pending`: #FFC107 (желтый)
  - `approved`: #4CAF50 (зеленый)
  - `rejected`: #F44336 (красный)
- Поддерживать нажатие для открытия детальной информации
- Анимировать изменение статуса (pulse для pending)

#### 2.2 `RoleRequestNotification` (новый компонент)

**Назначение:** Toast/Alert уведомление при изменении статуса

**Используемые API:**

- `Alert` из react-native
- Push notifications (опционально)

**Функциональность:**

- Проверять статус запроса каждые 30 секунд
- При изменении статуса показывать соответствующее уведомление
- Различные сообщения для одобрения/отклонения:
  - **Одобрено:** "Ваш запрос на роль {role} одобрен! ✅"
  - **Отклонено:** "Ваш запрос на роль {role} отклонен. Причина: {reason}"
- Сохранять историю проверок в AsyncStorage

#### 2.3 Обновить `RoleRequestStatus` компонент

**Текущие функции:**

- Отображение истории запросов
- Pull-to-refresh механизм

**Требуется добавить:**

- Живое обновление (auto-refresh каждые 30 сек для pending запросов)
- Индикатор "Проверяется статус..." при загрузке
- Раскрытие/свертывание детальной информации (accordion)
- Кнопка "Скопировать ID запроса" для support
- Обработка случаев:
  - Запрос одобрен → показать поздравление
  - Запрос отклонен → показать причину и кнопку "Попробовать снова"
- Сортировка: Сначала pending, потом approved, потом rejected

#### 2.4 Интеграция в `HomeScreen`

**Требуется:**

- Добавить компонент `RoleRequestStatusBadge` в заголовок (если есть активный pending запрос)
- При нажатии на badge переходить на экран `RoleRequestStatus`
- Показывать количество pending запросов (если > 0)

#### 2.5 Обновить `RoleRequestForm`

**Текущие функции:**

- Форма для отправки запроса на роль

**Требуется добавить:**

- После успешной отправки показывать:
  - "✅ Запрос отправлен успешно!"
  - "Вы получите уведомление при решении"
  - Опцию "Проверить статус" которая переходит на экран статуса

### 3. Состояние и Кэширование

#### Context/State Management

Создать `RoleRequestContext` (в `src/context/RoleRequestContext.js`):

```javascript
{
  // Состояние
  allRequests: [],
  currentRequest: null,
  isLoading: boolean,
  lastUpdated: timestamp,

  // Методы
  fetchRoleRequests: async () => void,
  getRoleRequestStatus: async (role: string) => RoleRequestStatus,
  refreshRequests: async () => void,

  // Настройки
  autoRefreshInterval: 30000,  // 30 сек
  cacheExpiry: 30000,          // 30 сек
}
```

#### AsyncStorage для кэширования

- Ключи:
  - `@roleRequests` - список всех запросов
  - `@roleRequestStatus_{role}` - статус конкретной роли
  - `@roleRequestsLastUpdate` - время последнего обновления

### 4. Обработка ошибок

#### Сценарии

1. **Сетевая ошибка:**
   - Показать Alert с опцией повтора
   - Использовать кэшированные данные, если доступны
   - Максимум 3 попытки retry

2. **Запрос не найден:**
   - "Запрос на повышение роли не найден"
   - Предложить создать новый

3. **Тайм-аут (>10 сек):**
   - "Слишком долго загружается. Проверьте интернет"
   - Опция повтора

4. **Неавторизованный доступ:**
   - Перенаправить на экран логина

### 5. UX/UI Требования

#### Визуальные элементы

- Скелетон/Shimmer загрузка для списков
- Плавные переходы при смене статуса
- Индикаторы разных статусов:
  - Pending: ⏳ желтый цвет, пульсирующая анимация
  - Approved: ✅ зеленый цвет, checkmark анимация
  - Rejected: ❌ красный цвет

#### Интерактивность

- Swipe to refresh на экране статуса
- Long-press для копирования ID запроса
- Быстрый доступ к форме запроса на роль из экрана статуса
- Кнопка "Обновить" для принудительного обновления статуса

### 6. Логирование и Аналитика

Добавить логирование:

```javascript
// Логировать событие просмотра статуса
logEvent("view_role_request_status", {
  role: "creator",
  status: "pending",
  daysWaiting: 5,
});

// Логировать изменение статуса
logEvent("role_request_status_changed", {
  role: "creator",
  fromStatus: "pending",
  toStatus: "approved",
  responseTime: 86400, // секунды
});

// Логировать повторный запрос после отклонения
logEvent("role_request_retry", {
  role: "creator",
  previousReason: "Need more experience",
});
```

### 7. Тестирование

#### Unit Tests

- Проверка правильности парсинга статусов
- Проверка логики кэширования
- Проверка обработки ошибок сети

#### Integration Tests

- Полный цикл: запрос → ожидание → изменение статуса
- Проверка уведомлений при изменении статуса
- Проверка retry механизма

#### Manual Testing

- Протестировать на медленном интернете (3G)
- Протестировать при отсутствии интернета (offline mode)
- Протестировать в фоновом режиме (app background)

## Файлы для создания/изменения

### Создать:

```
src/
├── components/
│   ├── RoleRequestStatusBadge.js
│   └── RoleRequestNotification.js
├── context/
│   └── RoleRequestContext.js
├── hooks/
│   ├── useRoleRequestStatus.js
│   └── useRoleRequestAutoRefresh.js
└── utils/
    └── roleRequestCache.js
```

### Изменить:

```
src/
├── components/
│   ├── RoleRequestStatus.js (обновить)
│   ├── RoleRequestForm.js (обновить)
├── screens/
│   └── HomeScreen.js (обновить)
└── services/
    └── api.js (обновить)
```

## API Endpoints Используемые

```
GET  /v1/api/users/request-role/status?role={role}
GET  /v1/api/users/request-role/all
POST /v1/api/users/request-role
```

## Примеры использования

### В компоненте:

```javascript
import { useRoleRequestStatus } from "../hooks/useRoleRequestStatus";

function MyComponent() {
  const { status, isLoading, error, refetch } = useRoleRequestStatus("creator");

  return (
    <View>
      {isLoading && <ActivityIndicator />}
      {status && <RoleRequestStatusBadge status={status.status} />}
      {error && <Button title="Повторить" onPress={refetch} />}
    </View>
  );
}
```

### С Context:

```javascript
import { useRoleRequest } from '../context/RoleRequestContext';

function MyComponent() {
  const { allRequests, refreshRequests } = useRoleRequest();

  useEffect(() => {
    refreshRequests();
  }, []);

  return <FlatList data={allRequests} renderItem={...} />;
}
```

## Критерии приемки

- [ ] Все компоненты созданы и работают без ошибок
- [ ] Статус обновляется автоматически каждые 30 сек
- [ ] Уведомления показываются при изменении статуса
- [ ] Кэширование работает корректно
- [ ] Retry механизм срабатывает при сетевых ошибках
- [ ] UI отзывчив и не лагирует
- [ ] Обработаны все edge cases
- [ ] Добавлены unit и integration тесты
- [ ] Логирование событий работает
- [ ] Документация актуальна

## Приоритет

🔴 **Высокий:**

- Проверка статуса каждые 30 сек
- Уведомления при изменении статуса
- Кэширование результатов

🟡 **Средний:**

- RoleRequestStatusBadge компонент
- Retry механизм
- Скелетон загрузка

🟢 **Низкий:**

- Аналитика
- Упреждающие уведомления о сроках
- Детальная статистика времени ожидания
