import React, { useState } from "react";
import {
  View,
  Text,
  TextInput,
  Button,
  StyleSheet,
  Alert,
  ActivityIndicator,
  ScrollView,
  SafeAreaView,
} from "react-native";
import { Picker } from "@react-native-picker/picker";
import { requestRole } from "../services/api";

const RoleRequestForm = ({ onClose, onSuccess }) => {
  const [role, setRole] = useState("creator");
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    if (!reason.trim()) {
      Alert.alert("Error", "Please enter a reason");
      return;
    }

    if (reason.trim().length < 10) {
      Alert.alert("Error", "Reason must be at least 10 characters long");
      return;
    }

    if (reason.trim().length > 500) {
      Alert.alert("Error", "Reason must be no more than 500 characters");
      return;
    }

    try {
      setLoading(true);
      const response = await requestRole(role, reason.trim());
      Alert.alert("Success", response.message || "Role request submitted");
      setReason("");
      setRole("creator");
      if (onSuccess) {
        onSuccess();
      }
      if (onClose) {
        onClose();
      }
    } catch (error) {
      const errorMsg =
        error.response?.data?.message ||
        error.message ||
        "Failed to submit role request";
      Alert.alert("Error", errorMsg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView style={styles.container}>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.title}>Request Role Upgrade</Text>

        <View style={styles.formGroup}>
          <Text style={styles.label}>Select Role</Text>
          <Picker
            selectedValue={role}
            onValueChange={(itemValue) => setRole(itemValue)}
            style={styles.picker}
          >
            <Picker.Item label="Creator" value="creator" />
            <Picker.Item label="Support" value="support" />
          </Picker>
        </View>

        <View style={styles.formGroup}>
          <Text style={styles.label}>Reason for Request</Text>
          <TextInput
            style={styles.textarea}
            placeholder="Explain why you want this role (10-500 characters)"
            multiline
            numberOfLines={6}
            value={reason}
            onChangeText={setReason}
            editable={!loading}
            maxLength={500}
          />
          <Text style={styles.charCount}>{reason.length} / 500 characters</Text>
        </View>

        <View style={styles.buttonGroup}>
          <Button
            title={loading ? "Submitting..." : "Submit Request"}
            onPress={handleSubmit}
            disabled={loading}
            color="#007AFF"
          />
          {!loading && <Button title="Cancel" onPress={onClose} color="#666" />}
        </View>

        {loading && <ActivityIndicator size="large" color="#007AFF" />}
      </ScrollView>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#fff",
  },
  content: {
    padding: 20,
  },
  title: {
    fontSize: 24,
    fontWeight: "bold",
    marginBottom: 20,
    color: "#333",
  },
  formGroup: {
    marginBottom: 20,
  },
  label: {
    fontSize: 16,
    fontWeight: "600",
    marginBottom: 8,
    color: "#333",
  },
  picker: {
    borderWidth: 1,
    borderColor: "#ddd",
    borderRadius: 8,
    backgroundColor: "#f5f5f5",
  },
  textarea: {
    borderWidth: 1,
    borderColor: "#ddd",
    borderRadius: 8,
    padding: 12,
    fontSize: 14,
    backgroundColor: "#f5f5f5",
    minHeight: 120,
    textAlignVertical: "top",
  },
  charCount: {
    fontSize: 12,
    color: "#999",
    marginTop: 4,
    textAlign: "right",
  },
  buttonGroup: {
    gap: 10,
    marginTop: 20,
  },
});

export default RoleRequestForm;
