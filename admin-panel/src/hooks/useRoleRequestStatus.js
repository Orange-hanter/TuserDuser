import { useEffect, useState, useCallback, useRef } from "react";
import { getRoleRequestStatus } from "../services/api";

/**
 * Hook для получения и отслеживания статуса запроса на повышение роли
 * @param {string} role - Роль, для которой нужно получить статус (например, 'creator')
 * @param {object} options - Опции
 * @param {number} options.refreshInterval - Интервал обновления в мс (default: 30000)
 * @param {boolean} options.autoRefresh - Автоматическое обновление (default: true)
 * @returns {object} - { status, isLoading, error, refetch, lastUpdated }
 */
export const useRoleRequestStatus = (role, options = {}) => {
  const { refreshInterval = 30000, autoRefresh = true } = options;

  const [status, setStatus] = useState(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(null);
  const isMountedRef = useRef(true);
  const intervalRef = useRef(null);

  const fetchStatus = useCallback(async () => {
    if (!role) {
      setError("Role is required");
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      const data = await getRoleRequestStatus(role);

      if (isMountedRef.current) {
        setStatus(data);
        setLastUpdated(new Date());
      }
    } catch (err) {
      if (isMountedRef.current) {
        setError(err.message || "Failed to fetch role request status");
        console.error("Error fetching role request status:", err);
      }
    } finally {
      if (isMountedRef.current) {
        setIsLoading(false);
      }
    }
  }, [role]);

  // Initial fetch
  useEffect(() => {
    isMountedRef.current = true;
    fetchStatus();

    return () => {
      isMountedRef.current = false;
    };
  }, [fetchStatus]);

  // Auto-refresh
  useEffect(() => {
    if (!autoRefresh || !role) return;

    intervalRef.current = setInterval(() => {
      if (isMountedRef.current) {
        fetchStatus();
      }
    }, refreshInterval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [role, autoRefresh, refreshInterval, fetchStatus]);

  const refetch = useCallback(() => {
    fetchStatus();
  }, [fetchStatus]);

  return {
    status,
    isLoading,
    error,
    refetch,
    lastUpdated,
  };
};

export default useRoleRequestStatus;
