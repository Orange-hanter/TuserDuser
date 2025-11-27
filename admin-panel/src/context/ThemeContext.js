import React, { createContext, useState, useContext } from "react";

// Theme definitions
export const themes = {
  light: {
    colors: {
      background: "#ffffff",
      surface: "#f5f5f5",
      text: "#000000",
      textSecondary: "#666666",
      border: "#e0e0e0",
      primary: "#2196F3",
      success: "#4CAF50",
      warning: "#FF9800",
      danger: "#f44336",
      cardBackground: "#ffffff",
      cardBorder: "#ddd",
      headerBackground: "#ffffff",
      inputBackground: "#fafafa",
      inputBorder: "#ddd",
    },
  },
  dark: {
    colors: {
      background: "#121212",
      surface: "#1e1e1e",
      text: "#ffffff",
      textSecondary: "#b0b0b0",
      border: "#333333",
      primary: "#64B5F6",
      success: "#81C784",
      warning: "#FFB74D",
      danger: "#ef5350",
      cardBackground: "#1f1f1f",
      cardBorder: "#333333",
      headerBackground: "#1e1e1e",
      inputBackground: "#2a2a2a",
      inputBorder: "#444444",
    },
  },
};

// Create context
const ThemeContext = createContext();

// Provider component
export const ThemeProvider = ({ children }) => {
  const [isDarkMode, setIsDarkMode] = useState(false);

  const currentTheme = isDarkMode ? themes.dark : themes.light;

  const toggleTheme = () => {
    setIsDarkMode((prev) => !prev);
  };

  return (
    <ThemeContext.Provider
      value={{ isDarkMode, toggleTheme, theme: currentTheme }}
    >
      {children}
    </ThemeContext.Provider>
  );
};

// Hook to use theme
export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider");
  }
  return context;
};
