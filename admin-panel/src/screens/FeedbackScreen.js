import React, { useEffect, useState, useCallback } from "react";
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  Alert,
  Modal,
  ScrollView,
  RefreshControl,
  ActivityIndicator,
} from "react-native";
import { useTheme } from "../context/ThemeContext";
import {
  getFeedbackList,
  markFeedbackRead,
  deleteFeedback,
  getUnreadFeedbackCount,
} from "../services/api";

const CATEGORY_LABELS = {
  bug: "🐛 Баг",
  feature: "✨ Функция",
  inconvenience: "😤 Неудобство",
  other: "📝 Другое",
};

const CATEGORY_COLORS = {
  bug: "#DC3545",
  feature: "#28A745",
  inconvenience: "#FFC107",
  other: "#6C757D",
};

const FeedbackScreen = () => {
  const { theme } = useTheme();
  const [feedbacks, setFeedbacks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [selectedFeedback, setSelectedFeedback] = useState(null);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const pageSize = 20;

  const fetchFeedbacks = useCallback(
    async (pageNum = 1, showUnreadOnly = unreadOnly) => {
      try {
        const data = await getFeedbackList(pageNum, pageSize, showUnreadOnly);
        if (pageNum === 1) {
          setFeedbacks(data.feedbacks || []);
        } else {
          setFeedbacks((prev) => [...prev, ...(data.feedbacks || [])]);
        }
        setTotal(data.total);
        setPage(pageNum);
      } catch (error) {
        Alert.alert("Ошибка", "Не удалось загрузить список фидбеков");
      }
    },
    [unreadOnly],
  );

  const fetchUnreadCount = useCallback(async () => {
    try {
      const data = await getUnreadFeedbackCount();
      setUnreadCount(data.unreadCount);
    } catch (error) {
      console.error("Failed to fetch unread count", error);
    }
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    await Promise.all([fetchFeedbacks(1), fetchUnreadCount()]);
    setLoading(false);
  }, [fetchFeedbacks, fetchUnreadCount]);

  useEffect(() => {
    loadData();
  }, []);

  const handleRefresh = async () => {
    setRefreshing(true);
    await Promise.all([fetchFeedbacks(1, unreadOnly), fetchUnreadCount()]);
    setRefreshing(false);
  };

  const handleLoadMore = () => {
    if (feedbacks.length < total) {
      fetchFeedbacks(page + 1, unreadOnly);
    }
  };

  const toggleUnreadFilter = async () => {
    const newUnreadOnly = !unreadOnly;
    setUnreadOnly(newUnreadOnly);
    setLoading(true);
    await fetchFeedbacks(1, newUnreadOnly);
    setLoading(false);
  };

  const handleMarkRead = async (id, isRead) => {
    try {
      await markFeedbackRead(id, isRead);
      setFeedbacks((prev) =>
        prev.map((fb) => (fb.id === id ? { ...fb, isRead } : fb)),
      );
      fetchUnreadCount();
      if (selectedFeedback?.id === id) {
        setSelectedFeedback((prev) => ({ ...prev, isRead }));
      }
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось обновить статус");
    }
  };

  const handleDelete = async (id) => {
    Alert.alert(
      "Подтверждение",
      "Вы уверены, что хотите удалить этот фидбек?",
      [
        { text: "Отмена", style: "cancel" },
        {
          text: "Удалить",
          style: "destructive",
          onPress: async () => {
            try {
              await deleteFeedback(id);
              setFeedbacks((prev) => prev.filter((fb) => fb.id !== id));
              setTotal((prev) => prev - 1);
              fetchUnreadCount();
              setDetailModalVisible(false);
              Alert.alert("Успешно", "Фидбек удалён");
            } catch (error) {
              Alert.alert("Ошибка", "Не удалось удалить фидбек");
            }
          },
        },
      ],
    );
  };

  const openDetailModal = async (feedback) => {
    setSelectedFeedback(feedback);
    setDetailModalVisible(true);
    // Mark as read when opened
    if (!feedback.isRead) {
      await handleMarkRead(feedback.id, true);
    }
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    return date.toLocaleString("ru-RU", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const renderFeedbackItem = ({ item }) => (
    <TouchableOpacity
      style={[
        styles.feedbackItem,
        { backgroundColor: theme.colors.card },
        !item.isRead && styles.unreadItem,
      ]}
      onPress={() => openDetailModal(item)}
    >
      <View style={styles.feedbackHeader}>
        <View style={styles.categoryBadge}>
          <View
            style={[
              styles.categoryDot,
              {
                backgroundColor:
                  CATEGORY_COLORS[item.category] || CATEGORY_COLORS.other,
              },
            ]}
          />
          <Text style={[styles.categoryText, { color: theme.colors.text }]}>
            {CATEGORY_LABELS[item.category] || item.category}
          </Text>
        </View>
        {!item.isRead && (
          <View style={styles.unreadBadge}>
            <Text style={styles.unreadBadgeText}>NEW</Text>
          </View>
        )}
      </View>

      <Text
        style={[styles.feedbackMessage, { color: theme.colors.text }]}
        numberOfLines={2}
      >
        {item.message}
      </Text>

      <View style={styles.feedbackFooter}>
        <Text
          style={[styles.feedbackDate, { color: theme.colors.secondaryText }]}
        >
          {formatDate(item.createdAt)}
        </Text>
        <Text
          style={[styles.feedbackEmail, { color: theme.colors.secondaryText }]}
        >
          {item.userInfo?.email || "Аноним"}
        </Text>
      </View>
    </TouchableOpacity>
  );

  const renderDetailModal = () => {
    if (!selectedFeedback) return null;

    return (
      <Modal
        visible={detailModalVisible}
        animationType="slide"
        transparent={true}
        onRequestClose={() => setDetailModalVisible(false)}
      >
        <View style={styles.modalOverlay}>
          <View
            style={[
              styles.modalContent,
              { backgroundColor: theme.colors.card },
            ]}
          >
            <ScrollView showsVerticalScrollIndicator={false}>
              <View style={styles.modalHeader}>
                <Text style={[styles.modalTitle, { color: theme.colors.text }]}>
                  {CATEGORY_LABELS[selectedFeedback.category] ||
                    selectedFeedback.category}
                </Text>
                <TouchableOpacity onPress={() => setDetailModalVisible(false)}>
                  <Text style={styles.closeButton}>✕</Text>
                </TouchableOpacity>
              </View>

              <View style={styles.detailSection}>
                <Text
                  style={[
                    styles.detailLabel,
                    { color: theme.colors.secondaryText },
                  ]}
                >
                  Сообщение
                </Text>
                <Text
                  style={[styles.detailValue, { color: theme.colors.text }]}
                >
                  {selectedFeedback.message}
                </Text>
              </View>

              <View style={styles.detailSection}>
                <Text
                  style={[
                    styles.detailLabel,
                    { color: theme.colors.secondaryText },
                  ]}
                >
                  Дата
                </Text>
                <Text
                  style={[styles.detailValue, { color: theme.colors.text }]}
                >
                  {formatDate(selectedFeedback.createdAt)}
                </Text>
              </View>

              <View style={styles.detailSection}>
                <Text
                  style={[
                    styles.detailLabel,
                    { color: theme.colors.secondaryText },
                  ]}
                >
                  Пользователь
                </Text>
                <Text
                  style={[styles.detailValue, { color: theme.colors.text }]}
                >
                  Email: {selectedFeedback.userInfo?.email || "Не указан"}
                </Text>
                <Text
                  style={[styles.detailValue, { color: theme.colors.text }]}
                >
                  Имя: {selectedFeedback.userInfo?.firstName || "—"}{" "}
                  {selectedFeedback.userInfo?.lastName || ""}
                </Text>
              </View>

              <View style={styles.detailSection}>
                <Text
                  style={[
                    styles.detailLabel,
                    { color: theme.colors.secondaryText },
                  ]}
                >
                  Окружение
                </Text>
                <Text
                  style={[styles.detailValue, { color: theme.colors.text }]}
                >
                  ОС: {selectedFeedback.environment?.os || "—"}
                </Text>
                <Text
                  style={[styles.detailValue, { color: theme.colors.text }]}
                >
                  Экран: {selectedFeedback.environment?.screenSize || "—"}
                </Text>
                <Text
                  style={[styles.detailValue, { color: theme.colors.text }]}
                >
                  PWA: {selectedFeedback.environment?.pwa ? "Да" : "Нет"}
                </Text>
                <Text
                  style={[styles.detailValue, { color: theme.colors.text }]}
                  numberOfLines={2}
                >
                  URL: {selectedFeedback.environment?.url || "—"}
                </Text>
              </View>

              <View style={styles.modalActions}>
                <TouchableOpacity
                  style={[styles.actionButton, styles.markButton]}
                  onPress={() =>
                    handleMarkRead(
                      selectedFeedback.id,
                      !selectedFeedback.isRead,
                    )
                  }
                >
                  <Text style={styles.actionButtonText}>
                    {selectedFeedback.isRead
                      ? "📭 Непрочитанное"
                      : "📬 Прочитанное"}
                  </Text>
                </TouchableOpacity>

                <TouchableOpacity
                  style={[styles.actionButton, styles.deleteButton]}
                  onPress={() => handleDelete(selectedFeedback.id)}
                >
                  <Text style={styles.actionButtonText}>🗑 Удалить</Text>
                </TouchableOpacity>
              </View>
            </ScrollView>
          </View>
        </View>
      </Modal>
    );
  };

  if (loading && feedbacks.length === 0) {
    return (
      <View
        style={[styles.centered, { backgroundColor: theme.colors.background }]}
      >
        <ActivityIndicator size="large" color={theme.colors.primary} />
        <Text style={[styles.loadingText, { color: theme.colors.text }]}>
          Загрузка...
        </Text>
      </View>
    );
  }

  return (
    <View
      style={[styles.container, { backgroundColor: theme.colors.background }]}
    >
      <View style={styles.header}>
        <Text style={[styles.headerTitle, { color: theme.colors.text }]}>
          Обратная связь
        </Text>
        {unreadCount > 0 && (
          <View style={styles.unreadCountBadge}>
            <Text style={styles.unreadCountText}>{unreadCount}</Text>
          </View>
        )}
      </View>

      <View style={styles.filterRow}>
        <TouchableOpacity
          style={[styles.filterButton, unreadOnly && styles.filterButtonActive]}
          onPress={toggleUnreadFilter}
        >
          <Text
            style={[
              styles.filterButtonText,
              unreadOnly && styles.filterButtonTextActive,
            ]}
          >
            {unreadOnly ? "📬 Только новые" : "📋 Все"}
          </Text>
        </TouchableOpacity>
        <Text style={[styles.totalText, { color: theme.colors.secondaryText }]}>
          Всего: {total}
        </Text>
      </View>

      <FlatList
        data={feedbacks}
        keyExtractor={(item) => item.id}
        renderItem={renderFeedbackItem}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} />
        }
        onEndReached={handleLoadMore}
        onEndReachedThreshold={0.5}
        contentContainerStyle={styles.listContent}
        ListEmptyComponent={
          <View style={styles.emptyContainer}>
            <Text
              style={[styles.emptyText, { color: theme.colors.secondaryText }]}
            >
              {unreadOnly ? "Нет непрочитанных фидбеков" : "Нет фидбеков"}
            </Text>
          </View>
        }
        ListFooterComponent={
          feedbacks.length < total ? (
            <ActivityIndicator
              style={styles.loadMore}
              color={theme.colors.primary}
            />
          ) : null
        }
      />

      {renderDetailModal()}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  centered: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
  },
  loadingText: {
    marginTop: 10,
    fontSize: 16,
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    padding: 16,
    paddingBottom: 8,
  },
  headerTitle: {
    fontSize: 24,
    fontWeight: "bold",
  },
  unreadCountBadge: {
    backgroundColor: "#DC3545",
    borderRadius: 12,
    paddingHorizontal: 8,
    paddingVertical: 2,
    marginLeft: 10,
  },
  unreadCountText: {
    color: "#FFF",
    fontSize: 12,
    fontWeight: "bold",
  },
  filterRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 16,
    paddingBottom: 8,
  },
  filterButton: {
    backgroundColor: "#E9ECEF",
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 20,
  },
  filterButtonActive: {
    backgroundColor: "#007AFF",
  },
  filterButtonText: {
    color: "#495057",
    fontWeight: "600",
  },
  filterButtonTextActive: {
    color: "#FFF",
  },
  totalText: {
    fontSize: 14,
  },
  listContent: {
    padding: 16,
    paddingTop: 8,
  },
  feedbackItem: {
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 3,
  },
  unreadItem: {
    borderLeftWidth: 4,
    borderLeftColor: "#007AFF",
  },
  feedbackHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: 8,
  },
  categoryBadge: {
    flexDirection: "row",
    alignItems: "center",
  },
  categoryDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
    marginRight: 8,
  },
  categoryText: {
    fontSize: 14,
    fontWeight: "600",
  },
  unreadBadge: {
    backgroundColor: "#007AFF",
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 10,
  },
  unreadBadgeText: {
    color: "#FFF",
    fontSize: 10,
    fontWeight: "bold",
  },
  feedbackMessage: {
    fontSize: 15,
    lineHeight: 22,
    marginBottom: 8,
  },
  feedbackFooter: {
    flexDirection: "row",
    justifyContent: "space-between",
  },
  feedbackDate: {
    fontSize: 12,
  },
  feedbackEmail: {
    fontSize: 12,
  },
  emptyContainer: {
    alignItems: "center",
    padding: 40,
  },
  emptyText: {
    fontSize: 16,
  },
  loadMore: {
    padding: 20,
  },
  modalOverlay: {
    flex: 1,
    backgroundColor: "rgba(0, 0, 0, 0.5)",
    justifyContent: "flex-end",
  },
  modalContent: {
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    padding: 20,
    maxHeight: "90%",
  },
  modalHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    marginBottom: 20,
  },
  modalTitle: {
    fontSize: 20,
    fontWeight: "bold",
  },
  closeButton: {
    fontSize: 24,
    color: "#999",
  },
  detailSection: {
    marginBottom: 16,
  },
  detailLabel: {
    fontSize: 12,
    marginBottom: 4,
    textTransform: "uppercase",
  },
  detailValue: {
    fontSize: 15,
    lineHeight: 22,
  },
  modalActions: {
    flexDirection: "row",
    justifyContent: "space-between",
    marginTop: 20,
    gap: 12,
  },
  actionButton: {
    flex: 1,
    paddingVertical: 12,
    borderRadius: 8,
    alignItems: "center",
  },
  markButton: {
    backgroundColor: "#17A2B8",
  },
  deleteButton: {
    backgroundColor: "#DC3545",
  },
  actionButtonText: {
    color: "#FFF",
    fontWeight: "600",
    fontSize: 14,
  },
});

export default FeedbackScreen;
