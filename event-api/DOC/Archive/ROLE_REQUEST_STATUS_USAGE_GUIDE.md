# Инструкция по использованию компонентов отслеживания статуса роли

## Быстрый старт

### 1. Hook `useRoleRequestStatus`

Использование в компоненте для получения статуса:

```javascript
import { useRoleRequestStatus } from "../hooks/useRoleRequestStatus";

function MyComponent() {
  const { status, isLoading, error, refetch, lastUpdated } =
    useRoleRequestStatus("creator", {
      refreshInterval: 30000, // обновлять каждые 30 сек
      autoRefresh: true, // автоматическое обновление
    });

  if (isLoading) return <ActivityIndicator />;
  if (error) return <Text>Ошибка: {error}</Text>;

  return (
    <View>
      <Text>Статус: {status?.status}</Text>
      <Button title="Обновить" onPress={refetch} />
    </View>
  );
}
```

**Опции:**

- `refreshInterval` (number, default: 30000) - интервал обновления в миллисекундах
- `autoRefresh` (boolean, default: true) - автоматическое обновление

**Возвращает:**

- `status` - объект статуса запроса
- `isLoading` - загружается ли
- `error` - ошибка (если есть)
- `refetch` - функция для принудительного обновления
- `lastUpdated` - время последнего обновления

### 2. Компонент `RoleRequestStatusBadge`

Компактный индикатор статуса:

```javascript
import RoleRequestStatusBadge from "../components/RoleRequestStatusBadge";

function MyComponent({ navigation }) {
  return (
    <RoleRequestStatusBadge
      status="pending" // 'pending' | 'approved' | 'rejected'
      size="medium" // 'small' | 'medium' | 'large'
      showLabel={true} // показывать текст
      onPress={() => navigation.navigate("RoleRequestStatus")}
    />
  );
}
```

**Props:**

- `status` - текущий статус ('pending', 'approved', 'rejected')
- `size` - размер badge ('small', 'medium', 'large')
- `showLabel` - показывать текст статуса
- `onPress` - callback при нажатии

**Особенности:**

- Пульсирующая анимация для pending статуса
- Автоматическое изменение цвета в зависимости от статуса
- Поддержка нажатия для навигации

### 3. Компонент `RoleRequestStatus`

Экран с полной информацией о запросах:

```javascript
<RoleRequestStatus role="creator" navigation={navigation} />
```

**Props:**

- `role` - фильтр по роли (опционально)
- `navigation` - объект навигации

**Функциональность:**

- ✅ Автоматическое обновление каждые 30 сек
- ✅ Pull-to-refresh
- ✅ Accordion (раскрытие/свертывание)
- ✅ Копирование ID запроса (long-press)
- ✅ Кнопка "Попробовать снова" для отклоненных
- ✅ Сортировка (pending → approved → rejected)

### 4. Компонент `RoleRequestNotification`

Автоматические уведомления об изменениях:

```javascript
// Использовать в App.js
import RoleRequestNotification from "./src/components/RoleRequestNotification";

function App() {
  return (
    <View>
      <AppNavigator />
      <RoleRequestNotification /> {/* Компонент без видимого вывода */}
    </View>
  );
}
```

**Особенности:**

- 🔔 Отслеживает изменение статуса каждые 30 сек
- 📲 Показывает Alert при одобрении/отклонении
- 💾 Сохраняет историю в AsyncStorage
- 🔄 Работает в фоновом режиме

### 5. Интеграция в HomeScreen

Добавить badge в заголовок:

```javascript
import { useRoleRequestStatus } from "../hooks/useRoleRequestStatus";
import RoleRequestStatusBadge from "../components/RoleRequestStatusBadge";

function HomeScreen({ navigation }) {
  const { status } = useRoleRequestStatus("creator");

  return (
    <View>
      {status?.status === "pending" && (
        <RoleRequestStatusBadge
          status={status.status}
          size="small"
          onPress={() => navigation.navigate("RoleRequestStatus")}
        />
      )}
      {/* остальное содержимое */}
    </View>
  );
}
```

## Примеры использования

### Пример 1: Проверка статуса при открытии приложения

```javascript
useEffect(() => {
  const unsubscribe = navigation.addListener("focus", () => {
    refetch(); // Обновить статус при переходе на экран
  });
  return unsubscribe;
}, [navigation, refetch]);
```

### Пример 2: Retry логика

```javascript
const handleRetry = async () => {
  try {
    // Отправить новый запрос
    await requestRole("creator", "Новая попытка");
    // Очистить старый статус и обновить
    setSelectedRequest(null);
    await refetch();
  } catch (error) {
    Alert.alert("Ошибка", error.message);
  }
};
```

### Пример 3: Отображение времени ожидания

```javascript
function getWaitingTime(createdAt) {
  const now = new Date();
  const created = new Date(createdAt);
  const diffMs = now - created;
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
  return `${diffDays} дня${diffDays !== 1 ? "й" : ""}`;
}

// В компоненте:
<Text>Ожидание: {getWaitingTime(request.created_at)}</Text>;
```

## Обработка ошибок

### Сетевая ошибка

```javascript
if (error) {
  return (
    <View style={{ alignItems: "center", padding: 20 }}>
      <Text style={{ color: "red" }}>❌ {error}</Text>
      <Button title="Повторить" onPress={refetch} color="#007AFF" />
    </View>
  );
}
```

### Retry с экспоненциальной задержкой

```javascript
const retryWithBackoff = async (fn, maxRetries = 3) => {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      const delay = Math.pow(2, i) * 1000; // 1s, 2s, 4s
      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }
};
```

## Оптимизация производительности

### Кэширование результатов

```javascript
import AsyncStorage from "@react-native-async-storage/async-storage";

const getCachedStatus = async (role) => {
  try {
    const cached = await AsyncStorage.getItem(`@roleStatus_${role}`);
    if (cached) {
      const { data, timestamp } = JSON.parse(cached);
      const isExpired = Date.now() - timestamp > 30000; // 30 сек
      if (!isExpired) return data;
    }
  } catch (error) {
    console.error("Cache error:", error);
  }
  return null;
};
```

### Уменьшение количества запросов

```javascript
const { status, refetch } = useRoleRequestStatus("creator", {
  autoRefresh: false, // Отключить автоматическое обновление
  refreshInterval: 60000, // Или увеличить интервал до 60 сек
});

// Обновлять только при фокусе экрана
useEffect(() => {
  const unsubscribe = navigation.addListener("focus", refetch);
  return unsubscribe;
}, [navigation, refetch]);
```

## Часто задаваемые вопросы

**Q: Как отключить автоматическое обновление?**

```javascript
const { status } = useRoleRequestStatus("creator", {
  autoRefresh: false,
});
```

**Q: Как обновить вручную?**

```javascript
const { refetch } = useRoleRequestStatus("creator");
// Позже:
await refetch();
```

**Q: Как отключить уведомления?**

```javascript
// Просто не добавляйте RoleRequestNotification в App.js
```

**Q: Как изменить интервал обновления?**

```javascript
const { status } = useRoleRequestStatus("creator", {
  refreshInterval: 60000, // 60 сек вместо 30 сек
});
```

## Тестирование

### Unit tests для hook

```javascript
import { renderHook, act } from "@testing-library/react-hooks";
import { useRoleRequestStatus } from "../hooks/useRoleRequestStatus";

test("должен загружать статус", async () => {
  const { result, waitForNextUpdate } = renderHook(() =>
    useRoleRequestStatus("creator"),
  );

  expect(result.current.isLoading).toBe(true);
  await waitForNextUpdate();
  expect(result.current.status).toBeDefined();
});
```

### Integration tests

```javascript
test("должен обновлять статус каждые 30 сек", async () => {
  jest.useFakeTimers();
  const { refetch } = renderHook(() => useRoleRequestStatus("creator"));

  jest.advanceTimersByTime(30000);
  // Проверяем что refetch был вызван

  jest.useRealTimers();
});
```
