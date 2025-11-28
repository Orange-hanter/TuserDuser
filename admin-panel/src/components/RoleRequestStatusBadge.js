import React, { useEffect, useState } from "react";
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Animated,
} from "react-native";

/**
 * Компактный индикатор статуса запроса на повышение роли
 * @param {object} props
 * @param {'pending' | 'approved' | 'rejected'} props.status - Статус
 * @param {'small' | 'medium' | 'large'} props.size - Размер (default: medium)
 * @param {boolean} props.showLabel - Показывать текст (default: true)
 * @param {function} props.onPress - Callback при нажатии
 */
const RoleRequestStatusBadge = ({
  status = "pending",
  size = "medium",
  showLabel = true,
  onPress,
}) => {
  const [pulseValue] = useState(new Animated.Value(1));

  // Пульсирующая анимация для pending статуса
  useEffect(() => {
    if (status !== "pending") return;

    const pulse = Animated.loop(
      Animated.sequence([
        Animated.timing(pulseValue, {
          toValue: 1.1,
          duration: 1000,
          useNativeDriver: true,
        }),
        Animated.timing(pulseValue, {
          toValue: 1,
          duration: 1000,
          useNativeDriver: true,
        }),
      ]),
    );

    pulse.start();

    return () => pulse.stop();
  }, [status, pulseValue]);

  const getStatusConfig = () => {
    switch (status) {
      case "approved":
        return {
          backgroundColor: "#4CAF50",
          icon: "✅",
          label: "Одобрено",
          textColor: "#fff",
        };
      case "rejected":
        return {
          backgroundColor: "#F44336",
          icon: "❌",
          label: "Отклонено",
          textColor: "#fff",
        };
      case "pending":
      default:
        return {
          backgroundColor: "#FFC107",
          icon: "⏳",
          label: "Ожидание",
          textColor: "#333",
        };
    }
  };

  const getSizeConfig = () => {
    switch (size) {
      case "small":
        return {
          paddingHorizontal: 8,
          paddingVertical: 4,
          fontSize: 11,
          iconSize: 12,
        };
      case "large":
        return {
          paddingHorizontal: 16,
          paddingVertical: 10,
          fontSize: 15,
          iconSize: 18,
        };
      case "medium":
      default:
        return {
          paddingHorizontal: 12,
          paddingVertical: 7,
          fontSize: 13,
          iconSize: 14,
        };
    }
  };

  const statusConfig = getStatusConfig();
  const sizeConfig = getSizeConfig();

  const animatedStyle =
    status === "pending" ? { transform: [{ scale: pulseValue }] } : {};

  return (
    <TouchableOpacity
      onPress={onPress}
      activeOpacity={onPress ? 0.7 : 1}
      disabled={!onPress}
    >
      <Animated.View
        style={[
          styles.badge,
          {
            backgroundColor: statusConfig.backgroundColor,
            paddingHorizontal: sizeConfig.paddingHorizontal,
            paddingVertical: sizeConfig.paddingVertical,
          },
          animatedStyle,
        ]}
      >
        <View style={styles.badgeContent}>
          <Text style={[styles.icon, { fontSize: sizeConfig.iconSize }]}>
            {statusConfig.icon}
          </Text>
          {showLabel && (
            <Text
              style={[
                styles.label,
                {
                  fontSize: sizeConfig.fontSize,
                  color: statusConfig.textColor,
                  marginLeft: 6,
                },
              ]}
            >
              {statusConfig.label}
            </Text>
          )}
        </View>
      </Animated.View>
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  badge: {
    borderRadius: 20,
    alignSelf: "flex-start",
  },
  badgeContent: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
  },
  icon: {
    fontWeight: "bold",
  },
  label: {
    fontWeight: "600",
  },
});

export default RoleRequestStatusBadge;
