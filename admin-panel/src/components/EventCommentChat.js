import React, { useEffect, useState, useRef, useCallback } from "react";
import {
  View,
  Text,
  FlatList,
  TextInput,
  Button,
  StyleSheet,
  Alert,
  ActivityIndicator,
  ScrollView,
} from "react-native";
import {
  SafeAreaView,
  useSafeAreaInsets,
} from "react-native-safe-area-context";
import { getEventComments, addEventComment } from "../services/api";

const EventCommentChat = ({ eventId, onClose }) => {
  const insets = useSafeAreaInsets();
  const [comments, setComments] = useState([]);
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [newComment, setNewComment] = useState("");

  // Use ref to track previous comments to avoid unnecessary re-renders
  const prevCommentsRef = useRef(null);
  const isMountedRef = useRef(true);

  // Memoized fetch function that only updates state if data changed
  const fetchComments = useCallback(async () => {
    try {
      const data = await getEventComments(eventId);

      // Only set loading true on first load
      if (prevCommentsRef.current === null && isMountedRef.current) {
        setLoading(true);
      }

      const newComments = data || [];

      // Compare with previous data - only update if changed
      const commentsJson = JSON.stringify(newComments);
      const prevJson = JSON.stringify(prevCommentsRef.current);

      if (commentsJson !== prevJson && isMountedRef.current) {
        setComments(newComments);
        prevCommentsRef.current = newComments;
      }

      if (isMountedRef.current) {
        setLoading(false);
      }
    } catch (error) {
      if (isMountedRef.current) {
        Alert.alert("Error", "Failed to load comments: " + error.message);
        setLoading(false);
      }
    }
  }, [eventId]);

  useEffect(() => {
    isMountedRef.current = true;
    fetchComments();

    // Refresh comments every 5 seconds, but only if data changed
    const interval = setInterval(() => {
      if (isMountedRef.current) {
        fetchComments();
      }
    }, 5000);

    return () => {
      isMountedRef.current = false;
      clearInterval(interval);
    };
  }, [eventId, fetchComments]);

  const handleSendComment = useCallback(async () => {
    if (!newComment.trim()) {
      Alert.alert("Warning", "Please enter a comment");
      return;
    }

    try {
      setSending(true);
      await addEventComment(eventId, newComment);
      setNewComment("");
      Alert.alert("Success", "Comment added");
      // Fetch updated comments immediately after sending
      await fetchComments();
    } catch (error) {
      Alert.alert("Error", "Failed to send comment: " + error.message);
    } finally {
      setSending(false);
    }
  }, [eventId, newComment, fetchComments]);

  const renderComment = useCallback(
    ({ item }) => (
      <View
        style={[
          styles.commentBubble,
          item.authorRole === "admin"
            ? styles.adminComment
            : styles.creatorComment,
        ]}
      >
        <Text style={styles.commentAuthor}>
          {item.authorRole.toUpperCase()} •{" "}
          {new Date(item.createdAt).toLocaleString()}
        </Text>
        <Text style={styles.commentText}>{item.comment}</Text>
      </View>
    ),
    [],
  );

  return (
    <SafeAreaView
      style={[
        styles.safeArea,
        { paddingTop: insets.top, paddingBottom: insets.bottom },
      ]}
    >
      <View style={styles.container}>
        <View style={styles.header}>
          <Text style={styles.title}>Event Comments Discussion</Text>
          <Button title="Close" onPress={onClose} color="#666" />
        </View>

        {loading ? (
          <View style={styles.centerContent}>
            <ActivityIndicator size="large" color="#007AFF" />
          </View>
        ) : (
          <>
            {comments.length === 0 ? (
              <View style={styles.centerContent}>
                <Text style={styles.emptyText}>
                  No comments yet. Start the conversation!
                </Text>
              </View>
            ) : (
              <FlatList
                data={comments}
                keyExtractor={(item) => item.id.toString()}
                renderItem={renderComment}
                style={styles.commentsList}
                scrollEnabled={true}
                contentContainerStyle={styles.listContent}
              />
            )}

            <View style={styles.inputContainer}>
              <TextInput
                style={styles.input}
                placeholder="Type your comment..."
                value={newComment}
                onChangeText={setNewComment}
                multiline
                editable={!sending}
                placeholderTextColor="#999"
              />
              <Button
                title={sending ? "Sending..." : "Send"}
                onPress={handleSendComment}
                disabled={sending}
              />
            </View>
          </>
        )}
      </View>
    </SafeAreaView>
  );
};

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: "#f5f5f5",
  },
  container: {
    flex: 1,
    backgroundColor: "#f5f5f5",
    display: "flex",
    flexDirection: "column",
  },
  header: {
    backgroundColor: "#fff",
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: "#ddd",
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    minHeight: 50,
  },
  title: {
    fontSize: 17,
    fontWeight: "600",
    color: "#333",
    flex: 1,
  },
  centerContent: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    paddingHorizontal: 16,
  },
  emptyText: {
    fontSize: 14,
    color: "#999",
    textAlign: "center",
  },
  listContent: {
    paddingHorizontal: 12,
    paddingVertical: 12,
    flexGrow: 1,
  },
  commentsList: {
    flex: 1,
    backgroundColor: "#f5f5f5",
  },
  commentBubble: {
    marginBottom: 12,
    marginHorizontal: 4,
    paddingHorizontal: 12,
    paddingVertical: 10,
    borderRadius: 12,
    maxWidth: "88%",
  },
  adminComment: {
    backgroundColor: "#E3F2FD",
    alignSelf: "flex-start",
    borderLeftWidth: 3,
    borderLeftColor: "#FF6B6B",
  },
  creatorComment: {
    backgroundColor: "#F0F0F0",
    alignSelf: "flex-end",
    borderLeftWidth: 3,
    borderLeftColor: "#4CAF50",
  },
  commentAuthor: {
    fontSize: 11,
    fontWeight: "600",
    color: "#555",
    marginBottom: 6,
  },
  commentText: {
    fontSize: 13,
    color: "#333",
    lineHeight: 18,
  },
  inputContainer: {
    backgroundColor: "#fff",
    paddingHorizontal: 12,
    paddingVertical: 10,
    paddingBottom: 12,
    borderTopWidth: 1,
    borderTopColor: "#e0e0e0",
    flexDirection: "row",
    alignItems: "flex-end",
    gap: 8,
    minHeight: 60,
  },
  input: {
    flex: 1,
    borderWidth: 1,
    borderColor: "#ddd",
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 8,
    maxHeight: 80,
    minHeight: 40,
    textAlignVertical: "top",
    fontSize: 13,
    backgroundColor: "#fafafa",
  },
});

export default React.memo(EventCommentChat);
