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
import {
  getPendingEvents,
  reviewEvent,
  requestEventRevision,
} from "../services/api";
import EventCommentChat from "../components/EventCommentChat";

const PendingEventsScreen = () => {
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [modalVisible, setModalVisible] = useState(false);
  const [rejectComment, setRejectComment] = useState("");
  const [selectedEventId, setSelectedEventId] = useState(null);
  const [chatModalVisible, setChatModalVisible] = useState(false);
  const [revisionModalVisible, setRevisionModalVisible] = useState(false);
  const [revisionComment, setRevisionComment] = useState("");

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

  const openRevisionModal = (id) => {
    setSelectedEventId(id);
    setRevisionComment("");
    setRevisionModalVisible(true);
  };

  const handleRequestRevision = async () => {
    if (!selectedEventId) return;
    try {
      await requestEventRevision(selectedEventId, revisionComment);
      Alert.alert("Success", "Revision request sent to creator");
      setRevisionModalVisible(false);
      fetchEvents();
    } catch (error) {
      Alert.alert("Error", `Failed to request revision: ${error.message}`);
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
          title="Chat"
          color="#2196F3"
          onPress={() => {
            setSelectedEventId(item.id);
            setChatModalVisible(true);
          }}
        />
        <View style={{ width: 10 }} />
        <Button
          title="Request Revision"
          color="#FF9800"
          onPress={() => openRevisionModal(item.id)}
        />
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

      {/* Reject Modal */}
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

      {/* Request Revision Modal */}
      <Modal
        animationType="slide"
        transparent={true}
        visible={revisionModalVisible}
        onRequestClose={() => setRevisionModalVisible(false)}
      >
        <View style={styles.centeredView}>
          <View style={styles.modalView}>
            <Text style={styles.modalText}>Request Event Revision</Text>
            <TextInput
              style={styles.input}
              onChangeText={setRevisionComment}
              value={revisionComment}
              placeholder="Describe what needs to be fixed..."
              multiline
            />
            <View style={styles.modalActions}>
              <Button
                title="Cancel"
                onPress={() => setRevisionModalVisible(false)}
              />
              <View style={{ width: 10 }} />
              <Button
                title="Send Request"
                color="#FF9800"
                onPress={handleRequestRevision}
              />
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
    flexWrap: "wrap",
    gap: 5,
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
  chatModalContainer: {
    flex: 1,
  },
});

export default PendingEventsScreen;
