import React, { createContext, useState, useContext } from "react";
import { setAuthToken } from "../services/api";

const AuthContext = createContext();

export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(null);

  const signIn = (userData, accessToken) => {
    setUser(userData);
    setToken(accessToken);
    setAuthToken(accessToken);
  };

  const signOut = () => {
    setUser(null);
    setToken(null);
    setAuthToken(null);
  };

  return (
    <AuthContext.Provider value={{ user, token, signIn, signOut }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => useContext(AuthContext);
