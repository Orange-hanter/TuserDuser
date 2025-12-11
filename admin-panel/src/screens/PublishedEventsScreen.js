import React, { useEffect, useState, useCallback } from "react";
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
import { useTheme } from "../context/ThemeContext";
import { getPublishedEvents, unpublishEvent } from "../services/api";

const PublishedEventsScreen = () => {
  const { theme } = useTheme();
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [unpublishReason, setUnpublishReason] = useState("");
  const [selectedEventId, setSelectedEventId] = useState(null);
  const [searchQuery, setSearchQuery] = useState("");

  const fetchEvents = useCallback(async () => {
    try {
      const data = await getPublishedEvents();
      setEvents(data || []);
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось загрузить опубликованные события");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchEvents();
  }, [fetchEvents]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchEvents();
    setRefreshing(false);
  };

  const openUnpublishModal = (id) => {
    setSelectedEventId(id);
    setUnpublishReason("");
    setModalVisible(true);
  };

  const handleUnpublish = async () => {
    if (!selectedEventId) return;

    if (!unpublishReason.trim()) {
      Alert.alert("Ошибка", "Укажите причину снятия с публикации");
      return;
    }

    try {
      await unpublishEvent(selectedEventId, unpublishReason);
      Alert.alert("Успешно", "Событие снято с публикации");
      setModalVisible(false);
      fetchEvents();
    } catch (error) {
      Alert.alert("Ошибка", `Не удалось снять событие: ${error.message}`);
    }
  };

  const getFilteredEvents = () => {
    if (!searchQuery.trim()) return events;
    const query = searchQuery.toLowerCase();
    return events.filter(
      (event) =>
        event.title?.toLowerCase().includes(query) ||
        event.type?.toLowerCase().includes(query) ||
        event.id?.toLowerCase().includes(query),
    );
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
        <Text style={[styles.title, { color: theme.colors.text }]}>
          {item.title || "Без названия"}
        </Text>
        <View style={styles.statusBadge}>
          <Text style={styles.statusText}>📢 Опубликовано</Text>
        </View>
      </View>

      <View style={styles.infoRow}>
        <Text style={[styles.infoLabel, { color: theme.colors.textSecondary }]}>
          ID:
        </Text>
        <Text style={[styles.infoValue, { color: theme.colors.text }]}>
          {item.id?.substring(0, 8)}...
        </Text>
      </View>

      <View style={styles.infoRow}>
        <Text style={[styles.infoLabel, { color: theme.colors.textSecondary }]}>
          Тип:
        </Text>
        <Text style={[styles.infoValue, { color: theme.colors.text }]}>
          {item.type || "Не указан"}
        </Text>
      </View>

      <View style={styles.infoRow}>
        <Text style={[styles.infoLabel, { color: theme.colors.textSecondary }]}>
          Начало:
        </Text>
        <Text style={[styles.infoValue, { color: theme.colors.text }]}>
          {formatDate(item.start)}
        </Text>
      </View>

      <View style={styles.infoRow}>
        <Text style={[styles.infoLabel, { color: theme.colors.textSecondary }]}>
          Регистраций:
        </Text>
        <Text style={[styles.infoValue, { color: theme.colors.text }]}>
          {item.registrations_count || 0}
        </Text>
      </View>

      <TouchableOpacity
        style={styles.unpublishButton}
        onPress={() => openUnpublishModal(item.id)}
        activeOpacity={0.7}
      >
        <Text style={styles.unpublishButtonText}>🚫 Снять с публикации</Text>
      </TouchableOpacity>
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
      {/* Search Bar */}
      <View style={styles.searchContainer}>
        <TextInput
          style={[
            styles.searchInput,
            { backgroundColor: theme.colors.card, color: theme.colors.text },
          ]}
          placeholder="🔍 Поиск по названию, типу или ID..."
          placeholderTextColor={theme.colors.textSecondary}
          value={searchQuery}
          onChangeText={setSearchQuery}
        />
      </View>

      {/* Stats */}
      <View style={styles.statsContainer}>
        <Text style={[styles.statsText, { color: theme.colors.textSecondary }]}>
          Всего опубликовано: {events.length} | Показано:{" "}
          {getFilteredEvents().length}
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
              {searchQuery ? "Ничего не найдено" : "Нет опубликованных событий"}
            </Text>
          </View>
        }
        contentContainerStyle={events.length === 0 && styles.emptyListContainer}
      />

      {/* Unpublish Modal */}
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
              Снять с публикации
            </Text>
            <Text
              style={[
                styles.modalSubtitle,
                { color: theme.colors.textSecondary },
              ]}
            >
              Укажите причину снятия события с публичного доступа
            </Text>

            <TextInput
              style={[
                styles.input,
                {
                  backgroundColor: theme.colors.background,
                  color: theme.colors.text,
                },
              ]}
              onChangeText={setUnpublishReason}
              value={unpublishReason}
              placeholder="Причина снятия с публикации..."
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
                  styles.confirmButton,
                  !unpublishReason.trim() && styles.disabledButton,
                ]}
                onPress={handleUnpublish}
                disabled={!unpublishReason.trim()}
              >
                <Text style={styles.confirmButtonText}>Снять</Text>
              </TouchableOpacity>
            </View>
          </View>
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
  searchContainer: {
    padding: 12,
  },
  searchInput: {
    borderRadius: 12,
    padding: 12,
    fontSize: 16,
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
    fontSize: 18,
    fontWeight: "bold",
    flex: 1,
    marginRight: 8,
  },
  statusBadge: {
    backgroundColor: "#E8F5E9",
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 8,
  },
  statusText: {
    fontSize: 12,
    color: "#4CAF50",
    fontWeight: "600",
  },
  infoRow: {
    flexDirection: "row",
    marginBottom: 6,
  },
  infoLabel: {
    fontSize: 14,
    width: 100,
  },
  infoValue: {
    fontSize: 14,
    flex: 1,
  },
  unpublishButton: {
    backgroundColor: "#ffebee",
    borderRadius: 8,
    padding: 12,
    marginTop: 12,
    alignItems: "center",
  },
  unpublishButtonText: {
    color: "#d32f2f",
    fontWeight: "600",
    fontSize: 15,
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
  confirmButton: {
    flex: 1,
    backgroundColor: "#d32f2f",
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
});

export default PublishedEventsScreen;
