import React, { useEffect, useState } from "react";
import {
  View,
  Text,
  FlatList,
  StyleSheet,
  Alert,
  TextInput,
  Modal,
  TouchableOpacity,
  RefreshControl,
  ActivityIndicator,
} from "react-native";
import {
  getPendingEvents,
  reviewEvent,
  requestEventRevision,
} from "../services/api";
import { useTheme } from "../context/ThemeContext";
import EventCommentChat from "../components/EventCommentChat";

const PendingEventsScreen = () => {
  const { theme } = useTheme();
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [rejectComment, setRejectComment] = useState("");
  const [selectedEventId, setSelectedEventId] = useState(null);
  const [chatModalVisible, setChatModalVisible] = useState(false);
  const [revisionModalVisible, setRevisionModalVisible] = useState(false);
  const [revisionComment, setRevisionComment] = useState("");

  // Filter states
  const [selectedStatuses, setSelectedStatuses] = useState([]);
  const [selectedTypes, setSelectedTypes] = useState([]);

  const fetchEvents = async () => {
    try {
      const data = await getPendingEvents();
      setEvents(data || []);
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось загрузить события на модерацию");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEvents();
  }, []);

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchEvents();
    setRefreshing(false);
  };

  const handleApprove = async (id) => {
    try {
      await reviewEvent(id, "approve", "Одобрено администратором");
      Alert.alert("Успешно", "Событие одобрено");
      fetchEvents();
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось одобрить событие");
    }
  };

  const openRejectModal = (id) => {
    setSelectedEventId(id);
    setRejectComment("");
    setModalVisible(true);
  };

  const handleReject = async () => {
    if (!selectedEventId) return;
    if (!rejectComment.trim()) {
      Alert.alert("Ошибка", "Укажите причину отклонения");
      return;
    }
    try {
      await reviewEvent(selectedEventId, "reject", rejectComment);
      Alert.alert("Успешно", "Событие отклонено");
      setModalVisible(false);
      fetchEvents();
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось отклонить событие");
    }
  };

  const openRevisionModal = (id) => {
    setSelectedEventId(id);
    setRevisionComment("");
    setRevisionModalVisible(true);
  };

  const handleRequestRevision = async () => {
    if (!selectedEventId) return;
    if (!revisionComment.trim()) {
      Alert.alert("Ошибка", "Укажите что нужно исправить");
      return;
    }
    try {
      await requestEventRevision(selectedEventId, revisionComment);
      Alert.alert("Успешно", "Запрос на доработку отправлен создателю");
      setRevisionModalVisible(false);
      fetchEvents();
    } catch (error) {
      Alert.alert("Ошибка", `Не удалось отправить запрос: ${error.message}`);
    }
  };

  // Filter helpers
  const toggleStatusFilter = (status) => {
    setSelectedStatuses((prev) =>
      prev.includes(status)
        ? prev.filter((s) => s !== status)
        : [...prev, status],
    );
  };

  const toggleTypeFilter = (type) => {
    setSelectedTypes((prev) =>
      prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type],
    );
  };

  const getFilteredEvents = () => {
    return events.filter((event) => {
      const statusMatch =
        selectedStatuses.length === 0 ||
        selectedStatuses.includes(event.status);
      const typeMatch =
        selectedTypes.length === 0 || selectedTypes.includes(event.type);
      return statusMatch && typeMatch;
    });
  };

  const getUniqueStatuses = () => {
    const statuses = new Set(events.map((e) => e.status || "pending"));
    return Array.from(statuses).sort();
  };

  const getUniqueTypes = () => {
    const types = new Set(events.map((e) => e.type || "Unknown"));
    return Array.from(types).sort();
  };

  const getStatusLabel = (status) => {
    const labels = {
      pending: "⏳ Ожидает",
      revision: "📝 На доработке",
      approved: "✅ Одобрено",
      rejected: "❌ Отклонено",
    };
    return labels[status] || status;
  };

  const formatDate = (dateString) => {
    if (!dateString) return "Не указана";
    const date = new Date(dateString);
    return date.toLocaleDateString("ru-RU", {
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const renderItem = ({ item }) => (
    <View style={[styles.card, { backgroundColor: theme.colors.card }]}>
      <View style={styles.cardHeader}>
        <Text
          style={[styles.title, { color: theme.colors.text }]}
          numberOfLines={2}
        >
          {item.title || "Без названия"}
        </Text>
        <View
          style={[
            styles.statusBadge,
            {
              backgroundColor:
                item.status === "pending" ? "#FFF3E0" : "#E3F2FD",
            },
          ]}
        >
          <Text style={styles.statusText}>{getStatusLabel(item.status)}</Text>
        </View>
      </View>

      <View style={styles.infoContainer}>
        <View style={styles.infoRow}>
          <Text
            style={[styles.infoLabel, { color: theme.colors.textSecondary }]}
          >
            ID:
          </Text>
          <Text style={[styles.infoValue, { color: theme.colors.text }]}>
            {item.id?.substring(0, 8)}...
          </Text>
        </View>
        <View style={styles.infoRow}>
          <Text
            style={[styles.infoLabel, { color: theme.colors.textSecondary }]}
          >
            Тип:
          </Text>
          <Text style={[styles.infoValue, { color: theme.colors.text }]}>
            {item.type || "Не указан"}
          </Text>
        </View>
        <View style={styles.infoRow}>
          <Text
            style={[styles.infoLabel, { color: theme.colors.textSecondary }]}
          >
            Начало:
          </Text>
          <Text style={[styles.infoValue, { color: theme.colors.text }]}>
            {formatDate(item.start)}
          </Text>
        </View>
      </View>

      <View style={styles.actions}>
        <TouchableOpacity
          style={[styles.actionButton, styles.approveButton]}
          onPress={() => handleApprove(item.id)}
        >
          <Text style={styles.actionButtonText}>✅ Одобрить</Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={[styles.actionButton, styles.chatButton]}
          onPress={() => {
            setSelectedEventId(item.id);
            setChatModalVisible(true);
          }}
        >
          <Text style={styles.actionButtonText}>💬 Чат</Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={[styles.actionButton, styles.revisionButton]}
          onPress={() => openRevisionModal(item.id)}
        >
          <Text style={styles.actionButtonText}>📝 Доработка</Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={[styles.actionButton, styles.rejectButton]}
          onPress={() => openRejectModal(item.id)}
        >
          <Text style={styles.actionButtonText}>❌ Отклонить</Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  if (loading) {
    return (
      <View
        style={[
          styles.centerContent,
          { backgroundColor: theme.colors.background },
        ]}
      >
        <ActivityIndicator size="large" color="#007AFF" />
      </View>
    );
  }

  return (
    <View
      style={[styles.container, { backgroundColor: theme.colors.background }]}
    >
      {/* Filter Bar */}
      <View style={[styles.filterBar, { backgroundColor: theme.colors.card }]}>
        <Text style={[styles.filterLabel, { color: theme.colors.text }]}>
          Статус:
        </Text>
        <View style={styles.filterButtonsContainer}>
          {getUniqueStatuses().map((status) => (
            <TouchableOpacity
              key={status}
              style={[
                styles.filterButton,
                selectedStatuses.includes(status) && styles.filterButtonActive,
              ]}
              onPress={() => toggleStatusFilter(status)}
            >
              <Text
                style={[
                  styles.filterButtonText,
                  selectedStatuses.includes(status) &&
                    styles.filterButtonTextActive,
                ]}
              >
                {getStatusLabel(status)}
              </Text>
            </TouchableOpacity>
          ))}
        </View>

        <Text style={[styles.filterLabel, { color: theme.colors.text }]}>
          Тип:
        </Text>
        <View style={styles.filterButtonsContainer}>
          {getUniqueTypes().map((type) => (
            <TouchableOpacity
              key={type}
              style={[
                styles.filterButton,
                selectedTypes.includes(type) && styles.filterButtonActiveGreen,
              ]}
              onPress={() => toggleTypeFilter(type)}
            >
              <Text
                style={[
                  styles.filterButtonText,
                  selectedTypes.includes(type) && styles.filterButtonTextActive,
                ]}
              >
                {type}
              </Text>
            </TouchableOpacity>
          ))}
        </View>

        {(selectedStatuses.length > 0 || selectedTypes.length > 0) && (
          <TouchableOpacity
            style={styles.clearFiltersButton}
            onPress={() => {
              setSelectedStatuses([]);
              setSelectedTypes([]);
            }}
          >
            <Text style={styles.clearFiltersText}>Сбросить фильтры</Text>
          </TouchableOpacity>
        )}
      </View>

      {/* Stats */}
      <View style={styles.statsContainer}>
        <Text style={[styles.statsText, { color: theme.colors.textSecondary }]}>
          Всего: {events.length} | Показано: {getFilteredEvents().length}
        </Text>
      </View>

      <FlatList
        data={getFilteredEvents()}
        keyExtractor={(item) => item.id}
        renderItem={renderItem}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} />
        }
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyStateIcon}>📭</Text>
            <Text
              style={[
                styles.emptyStateText,
                { color: theme.colors.textSecondary },
              ]}
            >
              Нет событий для модерации
            </Text>
          </View>
        }
        contentContainerStyle={
          getFilteredEvents().length === 0 && styles.emptyListContainer
        }
      />

      {/* Reject Modal */}
      <Modal
        animationType="slide"
        transparent={true}
        visible={modalVisible}
        onRequestClose={() => setModalVisible(false)}
      >
        <View style={styles.modalOverlay}>
          <View
            style={[styles.modalView, { backgroundColor: theme.colors.card }]}
          >
            <Text style={[styles.modalTitle, { color: theme.colors.text }]}>
              Отклонить событие
            </Text>
            <Text
              style={[
                styles.modalSubtitle,
                { color: theme.colors.textSecondary },
              ]}
            >
              Укажите причину отклонения
            </Text>
            <TextInput
              style={[
                styles.input,
                {
                  backgroundColor: theme.colors.background,
                  color: theme.colors.text,
                },
              ]}
              onChangeText={setRejectComment}
              value={rejectComment}
              placeholder="Причина отклонения..."
              placeholderTextColor={theme.colors.textSecondary}
              multiline
              numberOfLines={4}
            />
            <View style={styles.modalActions}>
              <TouchableOpacity
                style={styles.cancelButton}
                onPress={() => setModalVisible(false)}
              >
                <Text style={styles.cancelButtonText}>Отмена</Text>
              </TouchableOpacity>
              <TouchableOpacity
                style={[
                  styles.confirmRejectButton,
                  !rejectComment.trim() && styles.disabledButton,
                ]}
                onPress={handleReject}
                disabled={!rejectComment.trim()}
              >
                <Text style={styles.confirmButtonText}>Отклонить</Text>
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </Modal>

      {/* Request Revision Modal */}
      <Modal
        animationType="slide"
        transparent={true}
        visible={revisionModalVisible}
        onRequestClose={() => setRevisionModalVisible(false)}
      >
        <View style={styles.modalOverlay}>
          <View
            style={[styles.modalView, { backgroundColor: theme.colors.card }]}
          >
            <Text style={[styles.modalTitle, { color: theme.colors.text }]}>
              Запросить доработку
            </Text>
            <Text
              style={[
                styles.modalSubtitle,
                { color: theme.colors.textSecondary },
              ]}
            >
              Опишите что нужно исправить
            </Text>
            <TextInput
              style={[
                styles.input,
                {
                  backgroundColor: theme.colors.background,
                  color: theme.colors.text,
                },
              ]}
              onChangeText={setRevisionComment}
              value={revisionComment}
              placeholder="Что нужно исправить..."
              placeholderTextColor={theme.colors.textSecondary}
              multiline
              numberOfLines={4}
            />
            <View style={styles.modalActions}>
              <TouchableOpacity
                style={styles.cancelButton}
                onPress={() => setRevisionModalVisible(false)}
              >
                <Text style={styles.cancelButtonText}>Отмена</Text>
              </TouchableOpacity>
              <TouchableOpacity
                style={[
                  styles.confirmRevisionButton,
                  !revisionComment.trim() && styles.disabledButton,
                ]}
                onPress={handleRequestRevision}
                disabled={!revisionComment.trim()}
              >
                <Text style={styles.confirmButtonText}>Отправить</Text>
              </TouchableOpacity>
            </View>
          </View>
        </View>
      </Modal>

      {/* Chat Modal */}
      <Modal
        animationType="slide"
        transparent={false}
        visible={chatModalVisible}
        onRequestClose={() => setChatModalVisible(false)}
      >
        <View style={styles.chatModalContainer}>
          <EventCommentChat
            eventId={selectedEventId}
            onClose={() => {
              setChatModalVisible(false);
              fetchEvents();
            }}
          />
        </View>
      </Modal>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  centerContent: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
  },
  filterBar: {
    padding: 12,
    margin: 12,
    borderRadius: 12,
  },
  filterLabel: {
    fontSize: 14,
    fontWeight: "600",
    marginTop: 8,
    marginBottom: 8,
  },
  filterButtonsContainer: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8,
    marginBottom: 8,
  },
  filterButton: {
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 20,
    backgroundColor: "#f0f0f0",
  },
  filterButtonActive: {
    backgroundColor: "#2196F3",
  },
  filterButtonActiveGreen: {
    backgroundColor: "#4CAF50",
  },
  filterButtonText: {
    fontSize: 13,
    color: "#666",
  },
  filterButtonTextActive: {
    color: "#fff",
  },
  clearFiltersButton: {
    marginTop: 8,
    padding: 8,
    alignItems: "center",
  },
  clearFiltersText: {
    color: "#FF9800",
    fontWeight: "600",
  },
  statsContainer: {
    paddingHorizontal: 16,
    paddingBottom: 8,
  },
  statsText: {
    fontSize: 13,
  },
  card: {
    marginHorizontal: 12,
    marginBottom: 12,
    borderRadius: 12,
    padding: 16,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.1,
    shadowRadius: 2,
    elevation: 2,
  },
  cardHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-start",
    marginBottom: 12,
  },
  title: {
    fontSize: 17,
    fontWeight: "bold",
    flex: 1,
    marginRight: 8,
  },
  statusBadge: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 8,
  },
  statusText: {
    fontSize: 11,
    fontWeight: "600",
  },
  infoContainer: {
    marginBottom: 12,
  },
  infoRow: {
    flexDirection: "row",
    marginBottom: 4,
  },
  infoLabel: {
    fontSize: 13,
    width: 70,
  },
  infoValue: {
    fontSize: 13,
    flex: 1,
  },
  actions: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8,
  },
  actionButton: {
    paddingHorizontal: 12,
    paddingVertical: 10,
    borderRadius: 8,
    minWidth: 70,
    alignItems: "center",
  },
  approveButton: {
    backgroundColor: "#E8F5E9",
  },
  chatButton: {
    backgroundColor: "#E3F2FD",
  },
  revisionButton: {
    backgroundColor: "#FFF3E0",
  },
  rejectButton: {
    backgroundColor: "#FFEBEE",
  },
  actionButtonText: {
    fontSize: 13,
    fontWeight: "600",
  },
  emptyState: {
    alignItems: "center",
    paddingTop: 60,
  },
  emptyStateIcon: {
    fontSize: 48,
    marginBottom: 12,
  },
  emptyStateText: {
    fontSize: 16,
    textAlign: "center",
  },
  emptyListContainer: {
    flexGrow: 1,
  },
  modalOverlay: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    backgroundColor: "rgba(0,0,0,0.5)",
  },
  modalView: {
    margin: 20,
    borderRadius: 20,
    padding: 24,
    width: "85%",
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.25,
    shadowRadius: 4,
    elevation: 5,
  },
  modalTitle: {
    fontSize: 20,
    fontWeight: "bold",
    marginBottom: 8,
    textAlign: "center",
  },
  modalSubtitle: {
    fontSize: 14,
    marginBottom: 16,
    textAlign: "center",
  },
  input: {
    borderWidth: 1,
    borderColor: "#e0e0e0",
    borderRadius: 12,
    padding: 12,
    fontSize: 15,
    minHeight: 100,
    textAlignVertical: "top",
    marginBottom: 16,
  },
  modalActions: {
    flexDirection: "row",
    justifyContent: "space-between",
    gap: 12,
  },
  cancelButton: {
    flex: 1,
    backgroundColor: "#f5f5f5",
    borderRadius: 12,
    padding: 14,
    alignItems: "center",
  },
  cancelButtonText: {
    color: "#666",
    fontWeight: "600",
    fontSize: 16,
  },
  confirmRejectButton: {
    flex: 1,
    backgroundColor: "#d32f2f",
    borderRadius: 12,
    padding: 14,
    alignItems: "center",
  },
  confirmRevisionButton: {
    flex: 1,
    backgroundColor: "#FF9800",
    borderRadius: 12,
    padding: 14,
    alignItems: "center",
  },
  confirmButtonText: {
    color: "#fff",
    fontWeight: "600",
    fontSize: 16,
  },
  disabledButton: {
    opacity: 0.5,
  },
  chatModalContainer: {
    flex: 1,
  },
});

export default PendingEventsScreen;
