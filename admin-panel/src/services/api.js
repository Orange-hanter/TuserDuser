import axios from "axios";
import { Platform } from "react-native";

// Use localhost for iOS/Web and 10.0.2.2 for Android Emulator
const BASE_URL =
  Platform.OS === "android" ? "http://10.0.2.2:8080" : "http://localhost:8080";

const api = axios.create({
  baseURL: BASE_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

let authToken = null;

export const setAuthToken = (token) => {
  authToken = token;
  if (token) {
    api.defaults.headers.common["Authorization"] = `Bearer ${token}`;
  } else {
    delete api.defaults.headers.common["Authorization"];
  }
};

// Add interceptor to ensure header is set
api.interceptors.request.use(
  (config) => {
    if (authToken) {
      config.headers.Authorization = `Bearer ${authToken}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  },
);

export const login = async (email, password) => {
  const response = await api.post("v1/api/auth/login", { email, password });
  return response.data;
};

export const getPendingEvents = async () => {
  const response = await api.get("/v1/api/events/pending");
  return response.data;
};

export const reviewEvent = async (id, action, comment = "") => {
  const response = await api.post(`/v1/api/events/${id}/review`, {
    action,
    comment,
  });
  return response.data;
};

export const getUsers = async () => {
  const response = await api.get("v1/api/admin/users");
  return response.data;
};

export const updateUserRole = async (userId, role) => {
  const response = await api.put("v1/api/admin/users/role", {
    user_id: userId,
    role,
  });
  return response.data;
};

// Creator Event Comments API
export const getEventComments = async (eventId) => {
  const response = await api.get(`/v1/api/admin/events/${eventId}/comments`);
  return response.data;
};

export const addEventComment = async (eventId, comment) => {
  const response = await api.post(
    `/v1/api/creator/events/${eventId}/comments`,
    {
      comment,
    },
  );
  return response.data;
};

export const requestEventRevision = async (eventId, comment) => {
  const response = await api.post(
    `/v1/api/admin/events/${eventId}/request-revision`,
    {
      comment,
    },
  );
  return response.data;
};

export const requestRole = async (role, reason) => {
  const response = await api.post("/v1/api/users/request-role", {
    role,
    reason,
  });
  return response.data;
};

export const getRoleRequestStatus = async (role) => {
  const response = await api.get("/v1/api/users/request-role/status", {
    params: { role },
  });
  return response.data;
};

export const getAllRoleRequests = async () => {
  const response = await api.get("/v1/api/users/request-role/all");
  return response.data;
};

export const getPendingRoleRequests = async (limit = 20, offset = 0) => {
  const response = await api.get("/v1/api/admin/role-requests/pending", {
    params: { limit, offset },
  });
  return response.data;
};

export const approveRoleRequest = async (requestId, notes = "") => {
  const response = await api.post(
    `/v1/api/admin/role-requests/${requestId}/approve`,
    {
      request_id: requestId,
      notes,
    },
  );
  return response.data;
};

export const rejectRoleRequest = async (requestId, reason) => {
  const response = await api.post(
    `/v1/api/admin/role-requests/${requestId}/reject`,
    {
      request_id: requestId,
      reason,
    },
  );
  return response.data;
};

export default api;
