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

  // Filter states
  const [selectedStatuses, setSelectedStatuses] = useState([]);
  const [selectedTypes, setSelectedTypes] = useState([]);

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
      {/* Filter Bar */}
      <View style={styles.filterBar}>
        <Text style={styles.filterLabel}>Status:</Text>
        <View style={styles.filterButtonsContainer}>
          {getUniqueStatuses().map((status) => (
            <Button
              key={status}
              title={status}
              onPress={() => toggleStatusFilter(status)}
              color={selectedStatuses.includes(status) ? "#2196F3" : "#ccc"}
            />
          ))}
        </View>

        <Text style={styles.filterLabel}>Type:</Text>
        <View style={styles.filterButtonsContainer}>
          {getUniqueTypes().map((type) => (
            <Button
              key={type}
              title={type}
              onPress={() => toggleTypeFilter(type)}
              color={selectedTypes.includes(type) ? "#4CAF50" : "#ccc"}
            />
          ))}
        </View>

        {(selectedStatuses.length > 0 || selectedTypes.length > 0) && (
          <Button
            title="Clear Filters"
            onPress={() => {
              setSelectedStatuses([]);
              setSelectedTypes([]);
            }}
            color="#FF9800"
          />
        )}
      </View>

      {loading ? (
        <Text>Loading...</Text>
      ) : (
        <FlatList
          data={getFilteredEvents()}
          keyExtractor={(item) => item.id}
          renderItem={renderItem}
          ListEmptyComponent={<Text>No events match filters</Text>}
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
  filterBar: {
    backgroundColor: "#f9f9f9",
    padding: 12,
    marginBottom: 10,
    borderRadius: 5,
    borderWidth: 1,
    borderColor: "#e0e0e0",
  },
  filterLabel: {
    fontSize: 13,
    fontWeight: "600",
    marginTop: 8,
    marginBottom: 6,
    color: "#333",
  },
  filterButtonsContainer: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 6,
    marginBottom: 8,
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
