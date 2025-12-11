import React, { useEffect, useState, useCallback, useRef } from "react";
import {
  View,
  Text,
  FlatList,
  StyleSheet,
  Alert,
  TouchableOpacity,
  RefreshControl,
  ActivityIndicator,
  TextInput,
  Modal,
  KeyboardAvoidingView,
  Platform,
} from "react-native";
import { useTheme } from "../context/ThemeContext";
import {
  getAdminChats,
  getChatMessages,
  sendChatMessage,
} from "../services/api";

const CHAT_TOPICS = {
  publication: { label: "📝 Вопросы публикации", color: "#2196F3" },
  access: { label: "🔐 Повышение доступа", color: "#FF9800" },
  support: { label: "🛠 Техподдержка", color: "#4CAF50" },
  other: { label: "💬 Прочее", color: "#9C27B0" },
};

const AdminChatsScreen = () => {
  const { theme } = useTheme();
  const [chats, setChats] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [selectedTopic, setSelectedTopic] = useState(null);
  const [selectedChat, setSelectedChat] = useState(null);
  const [chatModalVisible, setChatModalVisible] = useState(false);
  const [messages, setMessages] = useState([]);
  const [messagesLoading, setMessagesLoading] = useState(false);
  const [newMessage, setNewMessage] = useState("");
  const [sending, setSending] = useState(false);
  const flatListRef = useRef(null);
  const messagesIntervalRef = useRef(null);

  const fetchChats = useCallback(async () => {
    try {
      const data = await getAdminChats(selectedTopic);
      setChats(data || []);
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось загрузить чаты");
    } finally {
      setLoading(false);
    }
  }, [selectedTopic]);

  useEffect(() => {
    fetchChats();
  }, [fetchChats]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await fetchChats();
    setRefreshing(false);
  };

  const openChat = async (chat) => {
    setSelectedChat(chat);
    setChatModalVisible(true);
    setMessagesLoading(true);

    try {
      const data = await getChatMessages(chat.id);
      setMessages(data || []);
      // Scroll to bottom
      setTimeout(() => {
        flatListRef.current?.scrollToEnd({ animated: false });
      }, 100);
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось загрузить сообщения");
    } finally {
      setMessagesLoading(false);
    }

    // Auto-refresh messages
    messagesIntervalRef.current = setInterval(async () => {
      try {
        const data = await getChatMessages(chat.id);
        setMessages(data || []);
      } catch (error) {
        console.error("Failed to refresh messages", error);
      }
    }, 5000);
  };

  const closeChat = () => {
    setChatModalVisible(false);
    setSelectedChat(null);
    setMessages([]);
    setNewMessage("");
    if (messagesIntervalRef.current) {
      clearInterval(messagesIntervalRef.current);
    }
    // Refresh chats to update unread counts
    fetchChats();
  };

  const handleSendMessage = async () => {
    if (!newMessage.trim() || !selectedChat) return;

    try {
      setSending(true);
      await sendChatMessage(selectedChat.id, newMessage.trim());
      setNewMessage("");
      // Refresh messages
      const data = await getChatMessages(selectedChat.id);
      setMessages(data || []);
      // Scroll to bottom
      setTimeout(() => {
        flatListRef.current?.scrollToEnd({ animated: true });
      }, 100);
    } catch (error) {
      Alert.alert("Ошибка", "Не удалось отправить сообщение");
    } finally {
      setSending(false);
    }
  };

  const getTopicInfo = (topic) => {
    return CHAT_TOPICS[topic] || CHAT_TOPICS.other;
  };

  const formatDate = (dateString) => {
    const date = new Date(dateString);
    const now = new Date();
    const isToday = date.toDateString() === now.toDateString();

    if (isToday) {
      return date.toLocaleTimeString("ru-RU", {
        hour: "2-digit",
        minute: "2-digit",
      });
    }
    return date.toLocaleDateString("ru-RU", {
      day: "2-digit",
      month: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const getChatsByTopic = () => {
    if (!selectedTopic) return chats;
    return chats.filter((chat) => chat.topic === selectedTopic);
  };

  const renderTopicFilter = () => (
    <View style={styles.topicFilterContainer}>
      <TouchableOpacity
        style={[
          styles.topicFilterButton,
          !selectedTopic && styles.topicFilterButtonActive,
        ]}
        onPress={() => setSelectedTopic(null)}
      >
        <Text
          style={[
            styles.topicFilterText,
            !selectedTopic && styles.topicFilterTextActive,
          ]}
        >
          Все
        </Text>
      </TouchableOpacity>
      {Object.entries(CHAT_TOPICS).map(([key, value]) => (
        <TouchableOpacity
          key={key}
          style={[
            styles.topicFilterButton,
            selectedTopic === key && { backgroundColor: value.color },
          ]}
          onPress={() => setSelectedTopic(selectedTopic === key ? null : key)}
        >
          <Text
            style={[
              styles.topicFilterText,
              selectedTopic === key && styles.topicFilterTextActive,
            ]}
          >
            {value.label}
          </Text>
        </TouchableOpacity>
      ))}
    </View>
  );

  const renderChatItem = ({ item }) => {
    const topicInfo = getTopicInfo(item.topic);

    return (
      <TouchableOpacity
        style={[styles.chatCard, { backgroundColor: theme.colors.card }]}
        onPress={() => openChat(item)}
        activeOpacity={0.7}
      >
        <View style={styles.chatHeader}>
          <View style={styles.chatUserInfo}>
            <View style={[styles.avatar, { backgroundColor: topicInfo.color }]}>
              <Text style={styles.avatarText}>
                {item.user_email?.charAt(0).toUpperCase() || "?"}
              </Text>
            </View>
            <View style={styles.chatTitleContainer}>
              <Text
                style={[styles.chatUserEmail, { color: theme.colors.text }]}
              >
                {item.user_email || "Неизвестный пользователь"}
              </Text>
              <View
                style={[
                  styles.topicBadge,
                  { backgroundColor: topicInfo.color + "20" },
                ]}
              >
                <Text
                  style={[styles.topicBadgeText, { color: topicInfo.color }]}
                >
                  {topicInfo.label}
                </Text>
              </View>
            </View>
          </View>
          {item.unread_count > 0 && (
            <View style={styles.unreadBadge}>
              <Text style={styles.unreadBadgeText}>{item.unread_count}</Text>
            </View>
          )}
        </View>

        <Text
          style={[styles.lastMessage, { color: theme.colors.textSecondary }]}
          numberOfLines={2}
        >
          {item.last_message || "Нет сообщений"}
        </Text>

        <View style={styles.chatFooter}>
          <Text
            style={[styles.chatTime, { color: theme.colors.textSecondary }]}
          >
            {item.last_message_at ? formatDate(item.last_message_at) : ""}
          </Text>
        </View>
      </TouchableOpacity>
    );
  };

  const renderMessage = ({ item }) => {
    const isAdmin = item.sender_role === "admin";

    return (
      <View
        style={[
          styles.messageBubble,
          isAdmin ? styles.adminMessage : styles.userMessage,
        ]}
      >
        <Text style={styles.messageAuthor}>
          {isAdmin ? "АДМИН" : item.sender_email || "Пользователь"} •{" "}
          {formatDate(item.created_at)}
        </Text>
        <Text style={styles.messageText}>{item.message}</Text>
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
      {/* Topic Filter */}
      {renderTopicFilter()}

      {/* Stats */}
      <View style={styles.statsContainer}>
        <Text style={[styles.statsText, { color: theme.colors.textSecondary }]}>
          Всего чатов: {chats.length} | Показано: {getChatsByTopic().length}
        </Text>
      </View>

      <FlatList
        data={getChatsByTopic()}
        keyExtractor={(item) => item.id}
        renderItem={renderChatItem}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} />
        }
        ListEmptyComponent={
          <View style={styles.emptyState}>
            <Text style={styles.emptyStateIcon}>💬</Text>
            <Text
              style={[
                styles.emptyStateText,
                { color: theme.colors.textSecondary },
              ]}
            >
              {selectedTopic ? "Нет чатов по этой теме" : "Нет активных чатов"}
            </Text>
          </View>
        }
        contentContainerStyle={
          getChatsByTopic().length === 0 && styles.emptyListContainer
        }
      />

      {/* Chat Modal */}
      <Modal
        animationType="slide"
        transparent={false}
        visible={chatModalVisible}
        onRequestClose={closeChat}
      >
        <KeyboardAvoidingView
          style={[
            styles.chatModalContainer,
            { backgroundColor: theme.colors.background },
          ]}
          behavior={Platform.OS === "ios" ? "padding" : undefined}
          keyboardVerticalOffset={Platform.OS === "ios" ? 0 : 0}
        >
          {/* Chat Header */}
          <View
            style={[
              styles.chatModalHeader,
              { backgroundColor: theme.colors.card },
            ]}
          >
            <TouchableOpacity onPress={closeChat} style={styles.closeButton}>
              <Text style={styles.closeButtonText}>← Назад</Text>
            </TouchableOpacity>
            <View style={styles.chatModalTitleContainer}>
              <Text
                style={[styles.chatModalTitle, { color: theme.colors.text }]}
              >
                {selectedChat?.user_email || "Чат"}
              </Text>
              {selectedChat && (
                <Text
                  style={[
                    styles.chatModalSubtitle,
                    { color: getTopicInfo(selectedChat.topic).color },
                  ]}
                >
                  {getTopicInfo(selectedChat.topic).label}
                </Text>
              )}
            </View>
          </View>

          {/* Messages */}
          {messagesLoading ? (
            <View style={styles.centerContent}>
              <ActivityIndicator size="large" color="#007AFF" />
            </View>
          ) : (
            <FlatList
              ref={flatListRef}
              data={messages}
              keyExtractor={(item) => item.id}
              renderItem={renderMessage}
              style={styles.messagesList}
              contentContainerStyle={styles.messagesListContent}
              ListEmptyComponent={
                <View style={styles.emptyMessagesState}>
                  <Text
                    style={[
                      styles.emptyStateText,
                      { color: theme.colors.textSecondary },
                    ]}
                  >
                    Начните диалог с пользователем
                  </Text>
                </View>
              }
            />
          )}

          {/* Message Input */}
          <View
            style={[
              styles.inputContainer,
              { backgroundColor: theme.colors.card },
            ]}
          >
            <TextInput
              style={[
                styles.messageInput,
                {
                  backgroundColor: theme.colors.background,
                  color: theme.colors.text,
                },
              ]}
              placeholder="Введите сообщение..."
              placeholderTextColor={theme.colors.textSecondary}
              value={newMessage}
              onChangeText={setNewMessage}
              multiline
              editable={!sending}
            />
            <TouchableOpacity
              style={[
                styles.sendButton,
                (!newMessage.trim() || sending) && styles.sendButtonDisabled,
              ]}
              onPress={handleSendMessage}
              disabled={!newMessage.trim() || sending}
            >
              <Text style={styles.sendButtonText}>{sending ? "..." : "➤"}</Text>
            </TouchableOpacity>
          </View>
        </KeyboardAvoidingView>
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
  topicFilterContainer: {
    flexDirection: "row",
    flexWrap: "wrap",
    padding: 12,
    gap: 8,
  },
  topicFilterButton: {
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 20,
    backgroundColor: "#f0f0f0",
  },
  topicFilterButtonActive: {
    backgroundColor: "#007AFF",
  },
  topicFilterText: {
    fontSize: 13,
    color: "#666",
  },
  topicFilterTextActive: {
    color: "#fff",
  },
  statsContainer: {
    paddingHorizontal: 16,
    paddingBottom: 8,
  },
  statsText: {
    fontSize: 13,
  },
  chatCard: {
    marginHorizontal: 12,
    marginBottom: 10,
    borderRadius: 12,
    padding: 14,
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.1,
    shadowRadius: 2,
    elevation: 2,
  },
  chatHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "flex-start",
    marginBottom: 10,
  },
  chatUserInfo: {
    flexDirection: "row",
    alignItems: "center",
    flex: 1,
  },
  avatar: {
    width: 40,
    height: 40,
    borderRadius: 20,
    justifyContent: "center",
    alignItems: "center",
    marginRight: 12,
  },
  avatarText: {
    color: "#fff",
    fontSize: 18,
    fontWeight: "bold",
  },
  chatTitleContainer: {
    flex: 1,
  },
  chatUserEmail: {
    fontSize: 15,
    fontWeight: "600",
    marginBottom: 4,
  },
  topicBadge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 8,
    alignSelf: "flex-start",
  },
  topicBadgeText: {
    fontSize: 11,
    fontWeight: "500",
  },
  unreadBadge: {
    backgroundColor: "#FF3B30",
    minWidth: 22,
    height: 22,
    borderRadius: 11,
    justifyContent: "center",
    alignItems: "center",
    paddingHorizontal: 6,
  },
  unreadBadgeText: {
    color: "#fff",
    fontSize: 12,
    fontWeight: "bold",
  },
  lastMessage: {
    fontSize: 14,
    lineHeight: 20,
    marginBottom: 8,
  },
  chatFooter: {
    flexDirection: "row",
    justifyContent: "flex-end",
  },
  chatTime: {
    fontSize: 12,
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
  // Chat Modal Styles
  chatModalContainer: {
    flex: 1,
  },
  chatModalHeader: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: 16,
    paddingVertical: 12,
    paddingTop: 50,
    borderBottomWidth: 1,
    borderBottomColor: "#e0e0e0",
  },
  closeButton: {
    marginRight: 12,
  },
  closeButtonText: {
    fontSize: 16,
    color: "#007AFF",
  },
  chatModalTitleContainer: {
    flex: 1,
  },
  chatModalTitle: {
    fontSize: 17,
    fontWeight: "600",
  },
  chatModalSubtitle: {
    fontSize: 13,
    marginTop: 2,
  },
  messagesList: {
    flex: 1,
  },
  messagesListContent: {
    padding: 12,
  },
  messageBubble: {
    marginBottom: 12,
    padding: 12,
    borderRadius: 12,
    maxWidth: "85%",
  },
  adminMessage: {
    backgroundColor: "#E3F2FD",
    alignSelf: "flex-start",
    borderLeftWidth: 3,
    borderLeftColor: "#2196F3",
  },
  userMessage: {
    backgroundColor: "#F5F5F5",
    alignSelf: "flex-end",
    borderRightWidth: 3,
    borderRightColor: "#4CAF50",
  },
  messageAuthor: {
    fontSize: 11,
    fontWeight: "600",
    color: "#666",
    marginBottom: 4,
  },
  messageText: {
    fontSize: 14,
    color: "#333",
    lineHeight: 20,
  },
  emptyMessagesState: {
    flex: 1,
    justifyContent: "center",
    alignItems: "center",
    paddingTop: 100,
  },
  inputContainer: {
    flexDirection: "row",
    alignItems: "flex-end",
    padding: 12,
    paddingBottom: 30,
    borderTopWidth: 1,
    borderTopColor: "#e0e0e0",
    gap: 10,
  },
  messageInput: {
    flex: 1,
    borderWidth: 1,
    borderColor: "#e0e0e0",
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 10,
    maxHeight: 100,
    fontSize: 15,
  },
  sendButton: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: "#007AFF",
    justifyContent: "center",
    alignItems: "center",
  },
  sendButtonDisabled: {
    backgroundColor: "#ccc",
  },
  sendButtonText: {
    color: "#fff",
    fontSize: 18,
  },
});

export default AdminChatsScreen;
