import React, { useEffect, useState, useCallback } from "react";
import {
  View,
  Text,
  FlatList,
  StyleSheet,
  Alert,
  Modal,
  TouchableOpacity,
  TextInput,
  RefreshControl,
  ActivityIndicator,
} from "react-native";
import { useTheme } from "../context/ThemeContext";
import { getUsers, updateUserRole } from "../services/api";

const ROLES = [
  { value: "user", label: "👤 Пользователь", color: "#9E9E9E" },
  { value: "creator", label: "✍️ Создатель", color: "#4CAF50" },
  { value: "support", label: "🛠 Поддержка", color: "#2196F3" },
  { value: "admin", label: "👑 Администратор", color: "#FF9800" },
];

const UsersScreen = () => {
  const { theme } = useTheme();
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [selectedUser, setSelectedUser] = useState(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const fetchUsers = useCallback(async () => {
    try {
      const data = await getUsers();
      setUsers(data || []);
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось загрузить список пользователей");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchUsers();
    setRefreshing(false);
  };

  const handleRoleUpdate = async (role) => {
    if (!selectedUser) return;
    try {
      await updateUserRole(selectedUser.id, role);
      Alert.alert("Успешно", "Роль пользователя обновлена");
      setModalVisible(false);
      fetchUsers();
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось обновить роль");
    }
  };

  const openRoleModal = (user) => {
    setSelectedUser(user);
    setModalVisible(true);
  };

  const getRoleInfo = (roleValue) => {
    return ROLES.find((r) => r.value === roleValue) || ROLES[0];
  };

  const getFilteredUsers = () => {
    if (!searchQuery.trim()) return users;
    const query = searchQuery.toLowerCase();
    return users.filter(
      (user) =>
        user.email?.toLowerCase().includes(query) ||
        user.id?.toLowerCase().includes(query) ||
        user.role?.toLowerCase().includes(query),
    );
  };

  const renderItem = ({ item }) => {
    const roleInfo = getRoleInfo(item.role);

    return (
      <View style={[styles.card, { backgroundColor: theme.colors.card }]}>
        <View style={styles.cardHeader}>
          <View style={[styles.avatar, { backgroundColor: roleInfo.color }]}>
            <Text style={styles.avatarText}>
              {item.email?.charAt(0).toUpperCase() || "?"}
            </Text>
          </View>
          <View style={styles.userInfo}>
            <Text style={[styles.email, { color: theme.colors.text }]}>
              {item.email}
            </Text>
            <Text
              style={[styles.userId, { color: theme.colors.textSecondary }]}
            >
              ID: {item.id?.substring(0, 12)}...
            </Text>
          </View>
        </View>

        <View style={styles.roleContainer}>
          <View
            style={[
              styles.roleBadge,
              { backgroundColor: roleInfo.color + "20" },
            ]}
          >
            <Text style={[styles.roleText, { color: roleInfo.color }]}>
              {roleInfo.label}
            </Text>
          </View>
          <TouchableOpacity
            style={styles.changeRoleButton}
            onPress={() => openRoleModal(item)}
          >
            <Text style={styles.changeRoleText}>Изменить роль</Text>
          </TouchableOpacity>
        </View>
      </View>
    );
  };

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
          placeholder="🔍 Поиск по email, ID или роли..."
          placeholderTextColor={theme.colors.textSecondary}
          value={searchQuery}
          onChangeText={setSearchQuery}
        />
      </View>

      {/* Stats */}
      <View style={styles.statsContainer}>
        <Text style={[styles.statsText, { color: theme.colors.textSecondary }]}>
          Всего пользователей: {users.length} | Показано:{" "}
          {getFilteredUsers().length}
        </Text>
      </View>

      <FlatList
        data={getFilteredUsers()}
        keyExtractor={(item) => item.id}
        renderItem={renderItem}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} />
        }
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyStateIcon}>👥</Text>
            <Text
              style={[
                styles.emptyStateText,
                { color: theme.colors.textSecondary },
              ]}
            >
              {searchQuery ? "Ничего не найдено" : "Нет пользователей"}
            </Text>
          </View>
        }
        contentContainerStyle={
          getFilteredUsers().length === 0 && styles.emptyListContainer
        }
      />

      {/* Role Selection Modal */}
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
              Изменить роль
            </Text>
            <Text
              style={[
                styles.modalSubtitle,
                { color: theme.colors.textSecondary },
              ]}
            >
              {selectedUser?.email}
            </Text>

            <View style={styles.rolesContainer}>
              {ROLES.map((role) => (
                <TouchableOpacity
                  key={role.value}
                  style={[
                    styles.roleOption,
                    selectedUser?.role === role.value &&
                      styles.roleOptionCurrent,
                    { borderColor: role.color },
                  ]}
                  onPress={() => handleRoleUpdate(role.value)}
                  disabled={selectedUser?.role === role.value}
                >
                  <Text style={[styles.roleOptionText, { color: role.color }]}>
                    {role.label}
                  </Text>
                  {selectedUser?.role === role.value && (
                    <Text style={styles.currentLabel}>Текущая</Text>
                  )}
                </TouchableOpacity>
              ))}
            </View>

            <TouchableOpacity
              style={styles.cancelButton}
              onPress={() => setModalVisible(false)}
            >
              <Text style={styles.cancelButtonText}>Отмена</Text>
            </TouchableOpacity>
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
    marginBottom: 10,
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
    alignItems: "center",
    marginBottom: 12,
  },
  avatar: {
    width: 48,
    height: 48,
    borderRadius: 24,
    justifyContent: "center",
    alignItems: "center",
    marginRight: 12,
  },
  avatarText: {
    color: "#fff",
    fontSize: 20,
    fontWeight: "bold",
  },
  userInfo: {
    flex: 1,
  },
  email: {
    fontSize: 16,
    fontWeight: "600",
    marginBottom: 4,
  },
  userId: {
    fontSize: 12,
  },
  roleContainer: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
  },
  roleBadge: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 16,
  },
  roleText: {
    fontSize: 13,
    fontWeight: "600",
  },
  changeRoleButton: {
    paddingHorizontal: 12,
    paddingVertical: 8,
    backgroundColor: "#f0f0f0",
    borderRadius: 8,
  },
  changeRoleText: {
    color: "#007AFF",
    fontSize: 14,
    fontWeight: "500",
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
    marginBottom: 4,
    textAlign: "center",
  },
  modalSubtitle: {
    fontSize: 14,
    marginBottom: 20,
    textAlign: "center",
  },
  rolesContainer: {
    gap: 10,
    marginBottom: 20,
  },
  roleOption: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    padding: 14,
    borderRadius: 12,
    borderWidth: 2,
    backgroundColor: "#fff",
  },
  roleOptionCurrent: {
    opacity: 0.6,
  },
  roleOptionText: {
    fontSize: 16,
    fontWeight: "600",
  },
  currentLabel: {
    fontSize: 12,
    color: "#999",
  },
  cancelButton: {
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
});

export default UsersScreen;
