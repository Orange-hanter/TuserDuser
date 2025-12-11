import React, { useEffect, useState, useCallback } from "react";
import {
  View,
  Text,
  FlatList,
  StyleSheet,
  Alert,
  ActivityIndicator,
  ScrollView,
  SafeAreaView,
  RefreshControl,
  Button,
  Modal,
  TextInput,
  TouchableOpacity,
} from "react-native";
import {
  getPendingRoleRequests,
  approveRoleRequest,
  rejectRoleRequest,
} from "../services/api";

const RoleRequestsManagement = () => {
  const [requests, setRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [total, setTotal] = useState(0);
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [actionType, setActionType] = useState(null); // "approve" or "reject"
  const [notes, setNotes] = useState("");
  const [processing, setProcessing] = useState(false);

  const fetchRequests = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getPendingRoleRequests(20, 0);
      setRequests(data.requests || []);
      setTotal(data.total || 0);
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось загрузить заявки: " + error.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRequests();
  }, [fetchRequests]);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      const data = await getPendingRoleRequests(20, 0);
      setRequests(data.requests || []);
      setTotal(data.total || 0);
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось обновить: " + error.message);
    } finally {
      setRefreshing(false);
    }
  }, []);

  const handleApprove = async () => {
    if (!selectedRequest) return;

    try {
      setProcessing(true);
      await approveRoleRequest(selectedRequest.id, notes);
      Alert.alert("Успешно", "Заявка одобрена");
      setModalVisible(false);
      setNotes("");
      setSelectedRequest(null);
      await fetchRequests();
    } catch (error) {
      Alert.alert("Ошибка", error.message || "Не удалось одобрить заявку");
    } finally {
      setProcessing(false);
    }
  };

  const handleReject = async () => {
    if (!selectedRequest) return;

    if (!notes.trim()) {
      Alert.alert("Ошибка", "Укажите причину отказа");
      return;
    }

    try {
      setProcessing(true);
      await rejectRoleRequest(selectedRequest.id, notes);
      Alert.alert("Успешно", "Заявка отклонена");
      setModalVisible(false);
      setNotes("");
      setSelectedRequest(null);
      await fetchRequests();
    } catch (error) {
      Alert.alert("Ошибка", error.message || "Не удалось отклонить заявку");
    } finally {
      setProcessing(false);
    }
  };

  const openActionModal = (request, type) => {
    setSelectedRequest(request);
    setActionType(type);
    setNotes("");
    setModalVisible(true);
  };

  const getRoleLabel = (role) => {
    const labels = {
      creator: "✍️ Создатель",
      support: "🛠 Поддержка",
      admin: "👑 Администратор",
    };
    return labels[role] || role;
  };

  const renderRequestItem = ({ item }) => (
    <View style={styles.requestCard}>
      <View style={styles.requestHeader}>
        <View>
          <Text style={styles.roleLabel}>
            {getRoleLabel(item.requested_role)}
          </Text>
          <Text style={styles.requestId}>ID: {item.id.substring(0, 8)}...</Text>
        </View>
        <Text style={styles.statusBadge}>⏳ Ожидает</Text>
      </View>

      <Text style={styles.reasonTitle}>Причина:</Text>
      <Text style={styles.reason}>{item.reason}</Text>

      <Text style={styles.timestamp}>
        📅 Создано: {new Date(item.created_at).toLocaleDateString("ru-RU")}
      </Text>

      <View style={styles.actionButtons}>
        <TouchableOpacity
          style={[styles.button, styles.approveButton]}
          onPress={() => openActionModal(item, "approve")}
          disabled={processing}
        >
          <Text style={styles.buttonText}>✅ Одобрить</Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.button, styles.rejectButton]}
          onPress={() => openActionModal(item, "reject")}
          disabled={processing}
        >
          <Text style={styles.buttonText}>❌ Отклонить</Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  if (loading && !refreshing) {
    return (
      <SafeAreaView style={styles.container}>
        <ActivityIndicator size="large" color="#007AFF" />
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.container}>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
      >
        <View style={styles.header}>
          <Text style={styles.title}>Заявки на доступ</Text>
          <Text style={styles.subtitle}>Ожидает рассмотрения: {total}</Text>
        </View>

        {requests.length === 0 ? (
          <View style={styles.emptyState}>
            <Text style={styles.emptyStateIcon}>📭</Text>
            <Text style={styles.emptyStateText}>
              Нет заявок на рассмотрение
            </Text>
          </View>
        ) : (
          <FlatList
            data={requests}
            renderItem={renderRequestItem}
            keyExtractor={(item) => item.id}
            scrollEnabled={false}
          />
        )}
      </ScrollView>

      <Modal visible={modalVisible} animationType="slide" transparent={true}>
        <SafeAreaView style={styles.modalContainer}>
          <View style={styles.modalContent}>
            <View style={styles.modalHeader}>
              <Text style={styles.modalTitle}>
                {actionType === "approve"
                  ? "Одобрить заявку"
                  : "Отклонить заявку"}
              </Text>
              <TouchableOpacity
                onPress={() => {
                  setModalVisible(false);
                  setSelectedRequest(null);
                }}
              >
                <Text style={styles.closeButton}>✕</Text>
              </TouchableOpacity>
            </View>

            {selectedRequest && (
              <>
                <View style={styles.modalBody}>
                  <Text style={styles.modalLabel}>Запрашиваемая роль:</Text>
                  <Text style={styles.modalValue}>
                    {selectedRequest.requested_role === "creator"
                      ? "✍️ Создатель"
                      : selectedRequest.requested_role === "support"
                        ? "🛠 Поддержка"
                        : selectedRequest.requested_role}
                  </Text>{" "}
                  <Text style={styles.modalLabel}>Причина пользователя:</Text>
                  <Text style={styles.modalValue}>
                    {selectedRequest.reason}
                  </Text>
                  <Text style={styles.modalLabel}>
                    {actionType === "approve"
                      ? "Комментарий (необязательно)"
                      : "Причина отказа"}
                  </Text>
                  <TextInput
                    style={styles.notesInput}
                    placeholder={
                      actionType === "approve"
                        ? "Добавьте комментарий..."
                        : "Объясните причину отказа..."
                    }
                    value={notes}
                    onChangeText={setNotes}
                    multiline
                    numberOfLines={4}
                    editable={!processing}
                    maxLength={500}
                  />
                  <Text style={styles.charCount}>{notes.length} / 500</Text>
                </View>

                <View style={styles.modalFooter}>
                  {actionType === "approve" ? (
                    <TouchableOpacity
                      style={[styles.modalButton, styles.approveButtonModal]}
                      onPress={handleApprove}
                      disabled={processing}
                    >
                      <Text style={styles.modalButtonText}>
                        {processing ? "Одобряем..." : "✅ Одобрить"}
                      </Text>
                    </TouchableOpacity>
                  ) : (
                    <TouchableOpacity
                      style={[styles.modalButton, styles.rejectButtonModal]}
                      onPress={handleReject}
                      disabled={processing || !notes.trim()}
                    >
                      <Text style={styles.modalButtonText}>
                        {processing ? "Отклоняем..." : "❌ Отклонить"}
                      </Text>
                    </TouchableOpacity>
                  )}

                  <TouchableOpacity
                    style={[styles.modalButton, styles.cancelButton]}
                    onPress={() => {
                      setModalVisible(false);
                      setSelectedRequest(null);
                      setNotes("");
                    }}
                    disabled={processing}
                  >
                    <Text style={styles.modalButtonText}>Отмена</Text>
                  </TouchableOpacity>
                </View>
              </>
            )}
          </View>
        </SafeAreaView>
      </Modal>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#f5f5f5",
  },
  scrollContent: {
    padding: 16,
  },
  header: {
    marginBottom: 20,
  },
  title: {
    fontSize: 24,
    fontWeight: "bold",
    color: "#333",
  },
  subtitle: {
    fontSize: 14,
    color: "#999",
    marginTop: 4,
  },
  emptyState: {
    alignItems: "center",
    justifyContent: "center",
    paddingVertical: 40,
  },
  emptyStateIcon: {
    fontSize: 48,
    marginBottom: 12,
  },
  emptyStateText: {
    fontSize: 16,
    color: "#999",
  },
  requestCard: {
    backgroundColor: "#fff",
    borderRadius: 8,
    padding: 16,
    marginBottom: 16,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.2,
    shadowRadius: 2,
    elevation: 3,
  },
  requestHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-start",
    marginBottom: 12,
  },
  roleLabel: {
    fontSize: 18,
    fontWeight: "bold",
    color: "#333",
  },
  requestId: {
    fontSize: 12,
    color: "#999",
    marginTop: 4,
  },
  statusBadge: {
    backgroundColor: "#FFC107",
    color: "#333",
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 20,
    fontWeight: "600",
    fontSize: 12,
  },
  reasonTitle: {
    fontSize: 14,
    fontWeight: "600",
    color: "#666",
    marginBottom: 4,
  },
  reason: {
    fontSize: 14,
    color: "#333",
    lineHeight: 20,
    marginBottom: 12,
  },
  timestamp: {
    fontSize: 12,
    color: "#999",
    marginBottom: 16,
  },
  actionButtons: {
    flexDirection: "row",
    gap: 10,
  },
  button: {
    flex: 1,
    paddingVertical: 10,
    paddingHorizontal: 12,
    borderRadius: 6,
    alignItems: "center",
  },
  approveButton: {
    backgroundColor: "#4CAF50",
  },
  rejectButton: {
    backgroundColor: "#F44336",
  },
  buttonText: {
    color: "#fff",
    fontWeight: "600",
    fontSize: 13,
  },
  modalContainer: {
    flex: 1,
    backgroundColor: "rgba(0, 0, 0, 0.5)",
    justifyContent: "flex-end",
  },
  modalContent: {
    backgroundColor: "#fff",
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    maxHeight: "80%",
  },
  modalHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingHorizontal: 20,
    paddingVertical: 16,
    borderBottomWidth: 1,
    borderBottomColor: "#eee",
  },
  modalTitle: {
    fontSize: 18,
    fontWeight: "bold",
    color: "#333",
  },
  closeButton: {
    fontSize: 24,
    color: "#999",
  },
  modalBody: {
    paddingHorizontal: 20,
    paddingVertical: 16,
  },
  modalLabel: {
    fontSize: 13,
    fontWeight: "600",
    color: "#666",
    marginTop: 12,
    marginBottom: 4,
  },
  modalValue: {
    fontSize: 14,
    color: "#333",
    paddingVertical: 8,
    paddingHorizontal: 12,
    backgroundColor: "#f5f5f5",
    borderRadius: 6,
  },
  notesInput: {
    borderWidth: 1,
    borderColor: "#ddd",
    borderRadius: 6,
    padding: 12,
    marginTop: 8,
    backgroundColor: "#f9f9f9",
    fontSize: 14,
    textAlignVertical: "top",
  },
  charCount: {
    fontSize: 12,
    color: "#999",
    marginTop: 4,
    textAlign: "right",
  },
  modalFooter: {
    paddingHorizontal: 20,
    paddingVertical: 16,
    borderTopWidth: 1,
    borderTopColor: "#eee",
    gap: 10,
  },
  modalButton: {
    paddingVertical: 12,
    paddingHorizontal: 20,
    borderRadius: 6,
    alignItems: "center",
  },
  approveButtonModal: {
    backgroundColor: "#4CAF50",
  },
  rejectButtonModal: {
    backgroundColor: "#F44336",
  },
  cancelButton: {
    backgroundColor: "#999",
  },
  modalButtonText: {
    color: "#fff",
    fontWeight: "600",
    fontSize: 14,
  },
});

export default RoleRequestsManagement;
