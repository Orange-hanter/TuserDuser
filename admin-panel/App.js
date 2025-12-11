import React from "react";
import { NavigationContainer } from "@react-navigation/native";
import { createStackNavigator } from "@react-navigation/stack";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider, useAuth } from "./src/context/AuthContext";
import { ThemeProvider, useTheme } from "./src/context/ThemeContext";
import LoginScreen from "./src/screens/LoginScreen";
import PendingEventsScreen from "./src/screens/PendingEventsScreen";
import PublishedEventsScreen from "./src/screens/PublishedEventsScreen";
import UsersScreen from "./src/screens/UsersScreen";
import FeedbackScreen from "./src/screens/FeedbackScreen";
import RoleRequestsManagement from "./src/components/RoleRequestsManagement";
import AdminChatsScreen from "./src/screens/AdminChatsScreen";
import {
  Button,
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  ScrollView,
} from "react-native";

const Stack = createStackNavigator();

const MenuButton = ({ title, icon, onPress, color = "#007AFF", badge }) => (
  <TouchableOpacity
    style={[styles.menuButton, { borderLeftColor: color }]}
    onPress={onPress}
    activeOpacity={0.7}
  >
    <View style={styles.menuButtonContent}>
      <Text style={styles.menuIcon}>{icon}</Text>
      <Text style={styles.menuTitle}>{title}</Text>
    </View>
    {badge > 0 && (
      <View style={[styles.badge, { backgroundColor: color }]}>
        <Text style={styles.badgeText}>{badge}</Text>
      </View>
    )}
    <Text style={styles.menuArrow}>›</Text>
  </TouchableOpacity>
);

const HomeScreen = ({ navigation }) => {
  const { signOut, user } = useAuth();
  const { theme, isDarkMode, toggleTheme } = useTheme();

  return (
    <ScrollView
      style={[styles.container, { backgroundColor: theme.colors.background }]}
      contentContainerStyle={styles.scrollContent}
    >
      <View style={styles.header}>
        <Text style={[styles.welcomeText, { color: theme.colors.text }]}>
          Админ-панель
        </Text>
        <Text style={[styles.userEmail, { color: theme.colors.textSecondary }]}>
          {user?.email}
        </Text>
      </View>

      <View style={styles.section}>
        <Text style={[styles.sectionTitle, { color: theme.colors.text }]}>
          👥 Пользователи
        </Text>
        <MenuButton
          title="Список пользователей"
          icon="👤"
          onPress={() => navigation.navigate("Users")}
          color="#4CAF50"
        />
        <MenuButton
          title="Заявки на уровень доступа"
          icon="🔐"
          onPress={() => navigation.navigate("RoleRequestsManagement")}
          color="#FF9800"
        />
      </View>

      <View style={styles.section}>
        <Text style={[styles.sectionTitle, { color: theme.colors.text }]}>
          📅 События
        </Text>
        <MenuButton
          title="Новые события (модерация)"
          icon="📝"
          onPress={() => navigation.navigate("PendingEvents")}
          color="#2196F3"
        />
        <MenuButton
          title="Опубликованные события"
          icon="📢"
          onPress={() => navigation.navigate("PublishedEvents")}
          color="#9C27B0"
        />
      </View>

      <View style={styles.section}>
        <Text style={[styles.sectionTitle, { color: theme.colors.text }]}>
          💬 Коммуникации
        </Text>
        <MenuButton
          title="Фидбеки"
          icon="📬"
          onPress={() => navigation.navigate("Feedback")}
          color="#6C63FF"
        />
        <MenuButton
          title="Чаты с пользователями"
          icon="💭"
          onPress={() => navigation.navigate("AdminChats")}
          color="#00BCD4"
        />
      </View>

      <View style={styles.footer}>
        <TouchableOpacity
          style={styles.themeButton}
          onPress={toggleTheme}
          activeOpacity={0.7}
        >
          <Text style={styles.themeButtonText}>
            {isDarkMode ? "☀️ Светлая тема" : "🌙 Тёмная тема"}
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={styles.logoutButton}
          onPress={signOut}
          activeOpacity={0.7}
        >
          <Text style={styles.logoutButtonText}>🚪 Выйти</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  scrollContent: {
    padding: 16,
  },
  header: {
    marginBottom: 24,
    paddingBottom: 16,
    borderBottomWidth: 1,
    borderBottomColor: "#e0e0e0",
  },
  welcomeText: {
    fontSize: 28,
    fontWeight: "bold",
    marginBottom: 4,
  },
  userEmail: {
    fontSize: 14,
    opacity: 0.7,
  },
  section: {
    marginBottom: 20,
  },
  sectionTitle: {
    fontSize: 16,
    fontWeight: "600",
    marginBottom: 12,
    marginLeft: 4,
  },
  menuButton: {
    backgroundColor: "#fff",
    borderRadius: 12,
    padding: 16,
    marginBottom: 8,
    flexDirection: "row",
    alignItems: "center",
    shadowColor: "#000",
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.1,
    shadowRadius: 2,
    elevation: 2,
    borderLeftWidth: 4,
  },
  menuButtonContent: {
    flex: 1,
    flexDirection: "row",
    alignItems: "center",
  },
  menuIcon: {
    fontSize: 20,
    marginRight: 12,
  },
  menuTitle: {
    fontSize: 16,
    fontWeight: "500",
    color: "#333",
  },
  menuArrow: {
    fontSize: 24,
    color: "#ccc",
    fontWeight: "300",
  },
  badge: {
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 10,
    marginRight: 8,
  },
  badgeText: {
    color: "#fff",
    fontSize: 12,
    fontWeight: "bold",
  },
  footer: {
    marginTop: 20,
    paddingTop: 20,
    borderTopWidth: 1,
    borderTopColor: "#e0e0e0",
    gap: 10,
  },
  themeButton: {
    backgroundColor: "#f5f5f5",
    borderRadius: 12,
    padding: 14,
    alignItems: "center",
  },
  themeButtonText: {
    fontSize: 16,
    color: "#666",
  },
  logoutButton: {
    backgroundColor: "#ffebee",
    borderRadius: 12,
    padding: 14,
    alignItems: "center",
  },
  logoutButtonText: {
    fontSize: 16,
    color: "#d32f2f",
    fontWeight: "500",
  },
});

const AppNavigator = () => {
  const { token } = useAuth();

  return (
    <NavigationContainer>
      <Stack.Navigator>
        {token ? (
          <>
            <Stack.Screen
              name="Home"
              component={HomeScreen}
              options={{ title: "Админ-панель", headerShown: false }}
            />
            <Stack.Screen
              name="PendingEvents"
              component={PendingEventsScreen}
              options={{ title: "Новые события" }}
            />
            <Stack.Screen
              name="PublishedEvents"
              component={PublishedEventsScreen}
              options={{ title: "Опубликованные события" }}
            />
            <Stack.Screen
              name="Users"
              component={UsersScreen}
              options={{ title: "Пользователи" }}
            />
            <Stack.Screen
              name="Feedback"
              component={FeedbackScreen}
              options={{ title: "Фидбеки" }}
            />
            <Stack.Screen
              name="RoleRequestsManagement"
              component={RoleRequestsManagement}
              options={{ title: "Заявки на доступ" }}
            />
            <Stack.Screen
              name="AdminChats"
              component={AdminChatsScreen}
              options={{ title: "Чаты с пользователями" }}
            />
          </>
        ) : (
          <Stack.Screen
            name="Login"
            component={LoginScreen}
            options={{ headerShown: false }}
          />
        )}
      </Stack.Navigator>
    </NavigationContainer>
  );
};

export default function App() {
  return (
    <SafeAreaProvider>
      <AuthProvider>
        <ThemeProvider>
          <AppNavigator />
        </ThemeProvider>
      </AuthProvider>
    </SafeAreaProvider>
  );
}
