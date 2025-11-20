import React, { useEffect, useState } from "react";
import {
  View,
  Text,
  FlatList,
  Button,
  StyleSheet,
  Alert,
  Modal,
  TouchableOpacity,
} from "react-native";
import { getUsers, updateUserRole } from "../services/api";

const ROLES = ["user", "creator", "support", "admin"];

const UsersScreen = () => {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [selectedUser, setSelectedUser] = useState(null);
  const [modalVisible, setModalVisible] = useState(false);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const data = await getUsers();
      setUsers(data || []);
    } catch (error) {
      Alert.alert("Error", "Failed to fetch users");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, []);

  const handleRoleUpdate = async (role) => {
    if (!selectedUser) return;
    try {
      await updateUserRole(selectedUser.id, role);
      Alert.alert("Success", "User role updated");
      setModalVisible(false);
      fetchUsers();
    } catch (error) {
      Alert.alert("Error", "Failed to update role");
    }
  };

  const openRoleModal = (user) => {
    setSelectedUser(user);
    setModalVisible(true);
  };

  const renderItem = ({ item }) => (
    <View style={styles.card}>
      <Text style={styles.email}>{item.email}</Text>
      <Text>ID: {item.id}</Text>
      <Text>Role: {item.role}</Text>
      <Button title="Change Role" onPress={() => openRoleModal(item)} />
    </View>
  );

  return (
    <View style={styles.container}>
      {loading ? (
        <Text>Loading...</Text>
      ) : (
        <FlatList
          data={users}
          keyExtractor={(item) => item.id}
          renderItem={renderItem}
        />
      )}

      <Modal
        animationType="slide"
        transparent={true}
        visible={modalVisible}
        onRequestClose={() => setModalVisible(false)}
      >
        <View style={styles.modalView}>
          <Text style={styles.modalTitle}>
            Select Role for {selectedUser?.email}
          </Text>
          {ROLES.map((role) => (
            <TouchableOpacity
              key={role}
              style={styles.roleButton}
              onPress={() => handleRoleUpdate(role)}
            >
              <Text style={styles.roleText}>{role}</Text>
            </TouchableOpacity>
          ))}
          <Button
            title="Cancel"
            color="red"
            onPress={() => setModalVisible(false)}
          />
        </View>
      </Modal>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 10,
  },
  card: {
    backgroundColor: "#fff",
    padding: 15,
    marginBottom: 10,
    borderRadius: 5,
    elevation: 2,
    flexDirection: "column",
    gap: 5,
  },
  email: {
    fontSize: 16,
    fontWeight: "bold",
  },
  modalView: {
    margin: 20,
    marginTop: 100,
    backgroundColor: "white",
    borderRadius: 20,
    padding: 35,
    alignItems: "center",
    shadowColor: "#000",
    shadowOffset: {
      width: 0,
      height: 2,
    },
    shadowOpacity: 0.25,
    shadowRadius: 4,
    elevation: 5,
  },
  modalTitle: {
    marginBottom: 15,
    textAlign: "center",
    fontSize: 18,
    fontWeight: "bold",
  },
  roleButton: {
    backgroundColor: "#2196F3",
    borderRadius: 10,
    padding: 10,
    elevation: 2,
    marginBottom: 10,
    width: 200,
    alignItems: "center",
  },
  roleText: {
    color: "white",
    fontWeight: "bold",
    textAlign: "center",
  },
});

export default UsersScreen;
