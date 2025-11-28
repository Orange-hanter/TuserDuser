import React, { useEffect, useRef, useState } from "react";
import { Alert, AsyncStorage } from "react-native";
import { getAllRoleRequests } from "../services/api";

/**
 * Компонент для отслеживания изменений статуса запросов на повышение ролей
 * и отправки уведомлений пользователю
 */
const RoleRequestNotification = () => {
  const [isMonitoring, setIsMonitoring] = useState(false);
  const intervalRef = useRef(null);
  const previousStatesRef = useRef({});

  // Загрузить сохраненные состояния из AsyncStorage
  const loadPreviousStates = async () => {
    try {
      const stored = await AsyncStorage.getItem("@roleRequestsStates");
      if (stored) {
        previousStatesRef.current = JSON.parse(stored);
      }
    } catch (error) {
      console.error("Error loading previous states:", error);
    }
  };

  // Сохранить состояния в AsyncStorage
  const savePreviousStates = async (states) => {
    try {
      await AsyncStorage.setItem("@roleRequestsStates", JSON.stringify(states));
    } catch (error) {
      console.error("Error saving previous states:", error);
    }
  };

  // Показать уведомление
  const showNotification = (title, message, onPress) => {
    Alert.alert(title, message, [
      {
        text: "Закрыть",
        onPress: () => {},
        style: "cancel",
      },
      ...(onPress
        ? [
            {
              text: "Перейти",
              onPress,
              style: "default",
            },
          ]
        : []),
    ]);
  };

  // Проверить изменения статусов
  const checkStatusChanges = async () => {
    try {
      const data = await getAllRoleRequests();
      const currentRequests = data || [];

      for (const request of currentRequests) {
        const requestKey = `${request.id}`;
        const previousState = previousStatesRef.current[requestKey];

        // Если это первый раз видим этот запрос, просто сохраняем
        if (!previousState) {
          previousStatesRef.current[requestKey] = request;
          continue;
        }

        // Если статус изменился
        if (previousState.status !== request.status) {
          let title = "";
          let message = "";

          if (request.status === "approved") {
            title = "✅ Запрос одобрен!";
            message = `Ваш запрос на роль "${request.requested_role}" одобрен администратором!\n\n${
              request.review_notes ? `Комментарий: ${request.review_notes}` : ""
            }`;
          } else if (request.status === "rejected") {
            title = "❌ Запрос отклонен";
            message = `Ваш запрос на роль "${request.requested_role}" отклонен.\n\nПричина: ${request.review_notes || "Не указана"}`;
          }

          if (title) {
            showNotification(title, message);
          }

          // Сохранить обновленное состояние
          previousStatesRef.current[requestKey] = request;
        }
      }

      // Сохранить состояния
      await savePreviousStates(previousStatesRef.current);
    } catch (error) {
      console.error("Error checking status changes:", error);
    }
  };

  // Запустить мониторинг
  const startMonitoring = async () => {
    if (isMonitoring) return;

    await loadPreviousStates();
    setIsMonitoring(true);

    // Первая проверка
    await checkStatusChanges();

    // Проверять каждые 30 секунд
    intervalRef.current = setInterval(async () => {
      await checkStatusChanges();
    }, 30000);
  };

  // Остановить мониторинг
  const stopMonitoring = () => {
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }
    setIsMonitoring(false);
  };

  // При загрузке компонента начать мониторинг
  useEffect(() => {
    startMonitoring();

    return () => {
      stopMonitoring();
    };
  }, []);

  // Компонент не рендерит ничего, только выполняет побочные эффекты
  return null;
};

export default RoleRequestNotification;
