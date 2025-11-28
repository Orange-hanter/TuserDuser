import React, { useEffect, useState, useCallback, useRef } from "react";
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
  TouchableOpacity,
  Share,
  Clipboard,
} from "react-native";
import { getAllRoleRequests, getRoleRequestStatus } from "../services/api";

const RoleRequestStatus = ({ role, navigation }) => {
  const [request, setRequest] = useState(null);
  const [allRequests, setAllRequests] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [expandedId, setExpandedId] = useState(null);
  const [autoRefreshing, setAutoRefreshing] = useState(true);
  const intervalRef = useRef(null);
  const hasPendingRef = useRef(false);

  const fetchStatus = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getAllRoleRequests();
      setAllRequests(data || []);

      // If specific role provided, also fetch that status
      if (role) {
        const statusData = await getRoleRequestStatus(role);
        setRequest(statusData);
      }
    } catch (error) {
      Alert.alert(
        "Error",
        "Failed to fetch role request status: " + error.message,
      );
    } finally {
      setLoading(false);
    }
  }, [role]);

  useEffect(() => {
    fetchStatus();
  }, [role, fetchStatus]);

  // Автоматическое обновление каждые 30 сек для pending запросов
  useEffect(() => {
    if (!autoRefreshing) return;

    // Проверяем есть ли pending запросы
    const hasPending = allRequests.some((req) => req.status === "pending");
    hasPendingRef.current = hasPending;

    if (!hasPending) return;

    intervalRef.current = setInterval(() => {
      if (hasPendingRef.current) {
        onRefresh();
      }
    }, 30000);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [autoRefreshing, allRequests]);

  const onRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      const data = await getAllRoleRequests();
      const sortedData = (data || []).sort((a, b) => {
        // Сортировка: Сначала pending, потом approved, потом rejected
        const statusOrder = { pending: 0, approved: 1, rejected: 2 };
        return (statusOrder[a.status] || 3) - (statusOrder[b.status] || 3);
      });
      setAllRequests(sortedData);
      if (role) {
        const statusData = await getRoleRequestStatus(role);
        setRequest(statusData);
      }
    } catch (error) {
      Alert.alert("Error", "Failed to refresh: " + error.message);
    } finally {
      setRefreshing(false);
    }
  }, [role]);

  const getStatusColor = (status) => {
    switch (status) {
      case "approved":
        return "#4CAF50";
      case "rejected":
        return "#F44336";
      case "pending":
        return "#FFC107";
      default:
        return "#999";
    }
  };

  const getStatusLabel = (status) => {
    switch (status) {
      case "approved":
        return "✅ Approved";
      case "rejected":
        return "❌ Rejected";
      case "pending":
        return "⏳ Pending";
      default:
        return status;
    }
  };

  const handleCopyId = (id) => {
    Clipboard.setString(id);
    Alert.alert("Скопировано", "ID запроса скопирован в буфер обмена");
  };

  const handleRetry = () => {
    if (navigation) {
      navigation.navigate("RoleRequest");
    }
  };

  const isExpanded = (id) => expandedId === id;
  const toggleExpanded = (id) => {
    setExpandedId(isExpanded(id) ? null : id);
  };

  const renderRequestItem = ({ item }) => (
    <TouchableOpacity
      style={styles.requestCard}
      onPress={() => toggleExpanded(item.id)}
      activeOpacity={0.7}
    >
      <View style={styles.requestHeader}>
        <View style={styles.headerLeft}>
          <Text style={styles.roleLabel}>
            {item.requested_role.toUpperCase()}
          </Text>
          <TouchableOpacity
            onLongPress={() => handleCopyId(item.id)}
            delayLongPress={500}
          >
            <Text style={styles.requestId}>
              ID: {item.id.substring(0, 12)}...
            </Text>
          </TouchableOpacity>
        </View>
        <Text
          style={[
            styles.statusBadge,
            { backgroundColor: getStatusColor(item.status) },
          ]}
        >
          {getStatusLabel(item.status)}
        </Text>
      </View>

      {isExpanded(item.id) && (
        <View style={styles.expandedContent}>
          <Text style={styles.reasonTitle}>Причина:</Text>
          <Text style={styles.reason}>{item.reason}</Text>

          <View style={styles.timestamps}>
            <Text style={styles.timestamp}>
              📅 Отправлено: {new Date(item.created_at).toLocaleDateString()}
            </Text>
            {item.reviewed_at && (
              <Text style={styles.timestamp}>
                ✏️ Рассмотрено:{" "}
                {new Date(item.reviewed_at).toLocaleDateString()}
              </Text>
            )}
          </View>

          {item.review_notes && (
            <View style={styles.reviewNotes}>
              <Text style={styles.reviewNotesTitle}>Комментарий админа:</Text>
              <Text style={styles.reviewNotesText}>{item.review_notes}</Text>
            </View>
          )}

          {item.status === "approved" && (
            <View style={styles.successMessage}>
              <Text style={styles.successText}>
                🎉 Поздравляем! Ваш запрос одобрен!
              </Text>
            </View>
          )}

          {item.status === "rejected" && (
            <TouchableOpacity style={styles.retryButton} onPress={handleRetry}>
              <Text style={styles.retryButtonText}>Попробовать снова →</Text>
            </TouchableOpacity>
          )}

          <TouchableOpacity
            style={styles.copyButton}
            onPress={() => handleCopyId(item.id)}
          >
            <Text style={styles.copyButtonText}>Скопировать ID</Text>
          </TouchableOpacity>
        </View>
      )}
    </TouchableOpacity>
  );

  if (loading && !refreshing) {
    return (
      <SafeAreaView style={styles.container}>
        <View style={styles.loadingContainer}>
          <ActivityIndicator size="large" color="#007AFF" />
          <Text style={styles.loadingText}>Проверяется статус...</Text>
        </View>
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
        <Text style={styles.title}>Role Request Status</Text>

        {allRequests.length === 0 ? (
          <View style={styles.emptyState}>
            <Text style={styles.emptyStateText}>No role requests yet</Text>
          </View>
        ) : (
          <FlatList
            data={allRequests}
            renderItem={renderRequestItem}
            keyExtractor={(item) => item.id}
            scrollEnabled={false}
          />
        )}
      </ScrollView>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#f5f5f5",
  },
  loadingContainer: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
  },
  loadingText: {
    marginTop: 12,
    fontSize: 14,
    color: "#666",
  },
  scrollContent: {
    padding: 16,
  },
  title: {
    fontSize: 24,
    fontWeight: "bold",
    marginBottom: 20,
    color: "#333",
  },
  emptyState: {
    alignItems: "center",
    justifyContent: "center",
    paddingVertical: 40,
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
    marginBottom: 8,
  },
  headerLeft: {
    flex: 1,
  },
  roleLabel: {
    fontSize: 18,
    fontWeight: "bold",
    color: "#333",
    marginBottom: 4,
  },
  requestId: {
    fontSize: 12,
    color: "#999",
  },
  statusBadge: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 20,
    color: "#fff",
    fontWeight: "600",
    fontSize: 12,
    overflow: "hidden",
  },
  expandedContent: {
    marginTop: 12,
    paddingTop: 12,
    borderTopWidth: 1,
    borderTopColor: "#eee",
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
  timestamps: {
    backgroundColor: "#f9f9f9",
    borderRadius: 6,
    padding: 10,
    marginBottom: 12,
  },
  timestamp: {
    fontSize: 12,
    color: "#666",
    lineHeight: 18,
  },
  reviewNotes: {
    backgroundColor: "#e3f2fd",
    borderRadius: 6,
    padding: 10,
    marginBottom: 12,
    borderLeftWidth: 4,
    borderLeftColor: "#2196F3",
  },
  reviewNotesTitle: {
    fontSize: 12,
    fontWeight: "600",
    color: "#1976D2",
    marginBottom: 4,
  },
  reviewNotesText: {
    fontSize: 13,
    color: "#333",
    lineHeight: 18,
  },
  successMessage: {
    backgroundColor: "#c8e6c9",
    borderRadius: 6,
    padding: 12,
    marginBottom: 12,
    borderLeftWidth: 4,
    borderLeftColor: "#4CAF50",
  },
  successText: {
    fontSize: 14,
    color: "#2e7d32",
    fontWeight: "600",
  },
  retryButton: {
    backgroundColor: "#007AFF",
    borderRadius: 6,
    paddingVertical: 10,
    paddingHorizontal: 12,
    marginBottom: 10,
    alignItems: "center",
  },
  retryButtonText: {
    color: "#fff",
    fontSize: 14,
    fontWeight: "600",
  },
  copyButton: {
    backgroundColor: "#f0f0f0",
    borderRadius: 6,
    paddingVertical: 10,
    paddingHorizontal: 12,
    alignItems: "center",
  },
  copyButtonText: {
    color: "#333",
    fontSize: 13,
    fontWeight: "500",
  },
});

export default RoleRequestStatus;
