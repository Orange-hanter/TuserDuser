import React from "react";
import { NavigationContainer } from "@react-navigation/native";
import { createStackNavigator } from "@react-navigation/stack";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider, useAuth } from "./src/context/AuthContext";
import { ThemeProvider, useTheme } from "./src/context/ThemeContext";
import LoginScreen from "./src/screens/LoginScreen";
import PendingEventsScreen from "./src/screens/PendingEventsScreen";
import UsersScreen from "./src/screens/UsersScreen";
import FeedbackScreen from "./src/screens/FeedbackScreen";
import RoleRequestForm from "./src/components/RoleRequestForm";
import RoleRequestStatus from "./src/components/RoleRequestStatus";
import RoleRequestsManagement from "./src/components/RoleRequestsManagement";
import RoleRequestNotification from "./src/components/RoleRequestNotification";
import { Button, View, Text } from "react-native";

const Stack = createStackNavigator();

const HomeScreen = ({ navigation }) => {
  const { signOut, user } = useAuth();
  const { theme, isDarkMode, toggleTheme } = useTheme();

  return (
    <View
      style={[
        { flex: 1, alignItems: "center", justifyContent: "center" },
        { backgroundColor: theme.colors.background },
      ]}
    >
      <Text
        style={[
          { fontSize: 20, marginBottom: 20 },
          { color: theme.colors.text },
        ]}
      >
        Welcome, {user?.email}
      </Text>
      <View style={{ width: "80%", gap: 10 }}>
        <Button
          title="Manage Pending Events"
          onPress={() => navigation.navigate("PendingEvents")}
        />
        <View style={{ height: 10 }} />
        <Button
          title="Manage Users"
          onPress={() => navigation.navigate("Users")}
        />
        <View style={{ height: 10 }} />
        <Button
          title="📬 Feedback"
          onPress={() => navigation.navigate("Feedback")}
          color="#6C63FF"
        />
        <View style={{ height: 10 }} />
        <Button
          title="Request Role Upgrade"
          onPress={() => navigation.navigate("RoleRequest")}
          color="#17A2B8"
        />
        <View style={{ height: 10 }} />
        <Button
          title="Check Role Request Status"
          onPress={() => navigation.navigate("RoleRequestStatus")}
          color="#6C63FF"
        />
        <View style={{ height: 10 }} />
        <Button
          title="Manage Role Requests"
          onPress={() => navigation.navigate("RoleRequestsManagement")}
          color="#FF6B6B"
        />
        <View style={{ height: 20 }} />
        <Button
          title={isDarkMode ? "☀️ Light Mode" : "🌙 Dark Mode"}
          onPress={toggleTheme}
          color={theme.colors.warning}
        />
        <View style={{ height: 10 }} />
        <Button title="Logout" color="red" onPress={signOut} />
      </View>
    </View>
  );
};

const AppNavigator = () => {
  const { token } = useAuth();

  return (
    <NavigationContainer>
      <Stack.Navigator>
        {token ? (
          <>
            <Stack.Screen name="Home" component={HomeScreen} />
            <Stack.Screen
              name="PendingEvents"
              component={PendingEventsScreen}
              options={{ title: "Pending Events" }}
            />
            <Stack.Screen
              name="Users"
              component={UsersScreen}
              options={{ title: "User Management" }}
            />
            <Stack.Screen
              name="Feedback"
              component={FeedbackScreen}
              options={{ title: "Feedback" }}
            />
            <Stack.Screen
              name="RoleRequest"
              component={RoleRequestForm}
              options={{ title: "Request Role Upgrade" }}
            />
            <Stack.Screen
              name="RoleRequestStatus"
              component={RoleRequestStatus}
              options={{ title: "Role Request Status" }}
            />
            <Stack.Screen
              name="RoleRequestsManagement"
              component={RoleRequestsManagement}
              options={{ title: "Manage Role Requests" }}
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
          <RoleRequestNotification />
        </ThemeProvider>
      </AuthProvider>
    </SafeAreaProvider>
  );
}
