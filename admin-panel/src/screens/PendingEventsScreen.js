import React, { useEffect, useState } from "react";
import {
  View,
  Text,
  FlatList,
  Button,
  StyleSheet,
  Alert,
  TextInput,
  Modal,
} from "react-native";
import { getPendingEvents, reviewEvent } from "../services/api";

const PendingEventsScreen = () => {
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [modalVisible, setModalVisible] = useState(false);
  const [rejectComment, setRejectComment] = useState("");
  const [selectedEventId, setSelectedEventId] = useState(null);

  const fetchEvents = async () => {
    setLoading(true);
    try {
      const data = await getPendingEvents();
      setEvents(data || []);
    } catch (error) {
      Alert.alert("Error", "Failed to fetch pending events");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEvents();
  }, []);

  const handleApprove = async (id) => {
    try {
      await reviewEvent(id, "approve", "Approved by admin");
      Alert.alert("Success", `Event approved`);
      fetchEvents();
    } catch (error) {
      Alert.alert("Error", `Failed to approve event`);
    }
  };

  const openRejectModal = (id) => {
    setSelectedEventId(id);
    setRejectComment("");
    setModalVisible(true);
  };

  const handleReject = async () => {
    if (!selectedEventId) return;
    try {
      await reviewEvent(selectedEventId, "reject", rejectComment);
      Alert.alert("Success", `Event rejected`);
      setModalVisible(false);
      fetchEvents();
    } catch (error) {
      Alert.alert("Error", `Failed to reject event`);
    }
  };

  const renderItem = ({ item }) => (
    <View style={styles.card}>
      <Text style={styles.title}>{item.title || "No Title"}</Text>
      <Text>ID: {item.id}</Text>
      <Text>Type: {item.type}</Text>
      <Text>Start: {item.start}</Text>
      <View style={styles.actions}>
        <Button title="Approve" onPress={() => handleApprove(item.id)} />
        <View style={{ width: 10 }} />
        <Button
          title="Reject"
          color="red"
          onPress={() => openRejectModal(item.id)}
        />
      </View>
    </View>
  );

  return (
    <View style={styles.container}>
      {loading ? (
        <Text>Loading...</Text>
      ) : (
        <FlatList
          data={events}
          keyExtractor={(item) => item.id}
          renderItem={renderItem}
          ListEmptyComponent={<Text>No pending events</Text>}
        />
      )}
      <Button title="Refresh" onPress={fetchEvents} />

      <Modal
        animationType="slide"
        transparent={true}
        visible={modalVisible}
        onRequestClose={() => setModalVisible(false)}
      >
        <View style={styles.centeredView}>
          <View style={styles.modalView}>
            <Text style={styles.modalText}>Reason for rejection:</Text>
            <TextInput
              style={styles.input}
              onChangeText={setRejectComment}
              value={rejectComment}
              placeholder="Enter comment"
              multiline
            />
            <View style={styles.modalActions}>
              <Button title="Cancel" onPress={() => setModalVisible(false)} />
              <View style={{ width: 10 }} />
              <Button
                title="Confirm Reject"
                color="red"
                onPress={handleReject}
              />
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
    padding: 10,
  },
  card: {
    backgroundColor: "#fff",
    padding: 15,
    marginBottom: 10,
    borderRadius: 5,
    elevation: 2,
  },
  title: {
    fontSize: 18,
    fontWeight: "bold",
    marginBottom: 5,
  },
  actions: {
    flexDirection: "row",
    marginTop: 10,
  },
  centeredView: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    marginTop: 22,
    backgroundColor: "rgba(0,0,0,0.5)",
  },
  modalView: {
    margin: 20,
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
    width: "80%",
  },
  modalText: {
    marginBottom: 15,
    textAlign: "center",
    fontWeight: "bold",
    fontSize: 18,
  },
  input: {
    height: 100,
    width: "100%",
    margin: 12,
    borderWidth: 1,
    padding: 10,
    textAlignVertical: "top",
    borderColor: "#ccc",
    borderRadius: 5,
  },
  modalActions: {
    flexDirection: "row",
    marginTop: 15,
  },
});

export default PendingEventsScreen;
