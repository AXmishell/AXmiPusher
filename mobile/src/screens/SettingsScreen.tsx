/**
 * 设置页。
 *
 * - 显示已登录邮箱与服务器地址(只读);
 * - 轮询间隔选择: 15 / 30 / 60 分钟(与原生 Worker 一致, 默认 15), 保存后重新 configure 原生;
 * - 说明前台刷新间隔(固定 30 秒);
 * - "退出登录"按钮: 清 keychain 会话 + 停止原生轮询。
 */
import React, { useEffect, useState } from 'react';
import {
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import {
  DEFAULT_POLL_INTERVAL_MINUTES,
  useAuth,
} from '../context/AuthContext';
import { colors, spacing } from '../theme';

/** 原生后台轮询可选间隔(分钟)。 */
const INTERVAL_OPTIONS = [15, 30, 60] as const;

/** 前台刷新间隔说明(JS 侧固定 30 秒轮询)。 */
const FOREGROUND_INTERVAL_TEXT = '30 秒';

export default function SettingsScreen() {
  const { session, logout, updateSession } = useAuth();
  const [selectedInterval, setSelectedInterval] = useState<number>(
    session?.pollIntervalMinutes ?? DEFAULT_POLL_INTERVAL_MINUTES,
  );
  const [saving, setSaving] = useState(false);

  // 会话就绪后同步当前间隔
  useEffect(() => {
    setSelectedInterval(
      session?.pollIntervalMinutes ?? DEFAULT_POLL_INTERVAL_MINUTES,
    );
  }, [session?.pollIntervalMinutes]);

  /** 保存轮询间隔: 持久化 + 重新配置原生 Worker。 */
  const onSaveInterval = async () => {
    setSaving(true);
    try {
      await updateSession({ pollIntervalMinutes: selectedInterval });
      Alert.alert('已保存', `后台轮询间隔已更新为 ${selectedInterval} 分钟`);
    } catch (err) {
      Alert.alert('保存失败', err instanceof Error ? err.message : '未知错误');
    } finally {
      setSaving(false);
    }
  };

  /** 退出登录(需二次确认)。 */
  const onLogoutPress = () => {
    Alert.alert('退出登录', '确定要退出当前账号吗?', [
      { text: '取消', style: 'cancel' },
      {
        text: '退出',
        style: 'destructive',
        onPress: () => {
          logout().catch(() => {
            // 退出失败也回到登录页(会话已由上下文清空)
          });
        },
      },
    ]);
  };

  return (
    <SafeAreaView style={styles.safeArea} edges={['bottom']}>
      <ScrollView contentContainerStyle={styles.container}>
        {/* 账号信息(只读) */}
        <Text style={styles.sectionTitle}>账号信息</Text>
        <View style={styles.card}>
          <View style={styles.row}>
            <Text style={styles.rowLabel}>登录邮箱</Text>
            <Text style={styles.rowValue}>{session?.email ?? '-'}</Text>
          </View>
          <View style={styles.divider} />
          <View style={styles.row}>
            <Text style={styles.rowLabel}>服务器地址</Text>
            <Text style={styles.rowValue}>{session?.serverUrl ?? '-'}</Text>
          </View>
        </View>

        {/* 轮询间隔 */}
        <Text style={styles.sectionTitle}>后台轮询间隔</Text>
        <View style={styles.card}>
          <Text style={styles.cardHint}>
            应用进入后台后, 原生 Worker 按此间隔检查新站内信并发送系统通知。
          </Text>
          <View style={styles.intervalRow}>
            {INTERVAL_OPTIONS.map(minutes => {
              const selected = selectedInterval === minutes;
              return (
                <Pressable
                  key={minutes}
                  style={[styles.intervalOption, selected && styles.intervalSelected]}
                  onPress={() => setSelectedInterval(minutes)}
                >
                  <Text
                    style={[
                      styles.intervalText,
                      selected && styles.intervalTextSelected,
                    ]}
                  >
                    {minutes} 分钟
                  </Text>
                </Pressable>
              );
            })}
          </View>
          <Pressable
            style={[
              styles.saveButton,
              saving && styles.saveButtonDisabled,
            ]}
            onPress={() => void onSaveInterval()}
            disabled={saving}
          >
            <Text style={styles.saveButtonText}>
              {saving ? '保存中...' : '保存间隔设置'}
            </Text>
          </Pressable>
        </View>

        {/* 前台刷新说明 */}
        <Text style={styles.sectionTitle}>前台刷新</Text>
        <View style={styles.card}>
          <Text style={styles.cardHint}>
            应用在前台时每 {FOREGROUND_INTERVAL_TEXT} 自动刷新一次未读数与站内信。
          </Text>
        </View>

        {/* 退出登录 */}
        <Pressable
          style={styles.logoutButton}
          onPress={onLogoutPress}
        >
          <Text style={styles.logoutText}>退出登录</Text>
        </Pressable>
        <Text style={styles.versionText}>AXmiPusher 移动客户端</Text>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  container: {
    padding: spacing.lg,
  },
  sectionTitle: {
    fontSize: 14,
    color: colors.muted,
    marginTop: spacing.lg,
    marginBottom: spacing.sm,
  },
  card: {
    backgroundColor: colors.card,
    borderRadius: 8,
    borderWidth: StyleSheet.hairlineWidth,
    borderColor: colors.border,
    padding: spacing.lg,
  },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  rowLabel: {
    fontSize: 14,
    color: colors.text,
  },
  rowValue: {
    fontSize: 14,
    color: colors.muted,
    maxWidth: '60%',
    textAlign: 'right',
  },
  divider: {
    height: StyleSheet.hairlineWidth,
    backgroundColor: colors.border,
    marginVertical: spacing.md,
  },
  cardHint: {
    fontSize: 13,
    color: colors.muted,
    lineHeight: 20,
  },
  intervalRow: {
    flexDirection: 'row',
    marginTop: spacing.md,
    marginBottom: spacing.lg,
  },
  intervalOption: {
    flex: 1,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    paddingVertical: 10,
    alignItems: 'center',
    marginHorizontal: 4,
    backgroundColor: colors.card,
  },
  intervalSelected: {
    borderColor: colors.primary,
    backgroundColor: '#eff6ff',
  },
  intervalText: {
    fontSize: 14,
    color: colors.text,
  },
  intervalTextSelected: {
    color: colors.primary,
    fontWeight: '600',
  },
  saveButton: {
    backgroundColor: colors.primary,
    borderRadius: 8,
    paddingVertical: 11,
    alignItems: 'center',
  },
  saveButtonDisabled: {
    opacity: 0.6,
  },
  saveButtonText: {
    color: '#ffffff',
    fontSize: 15,
    fontWeight: '600',
  },
  logoutButton: {
    marginTop: spacing.xl,
    backgroundColor: colors.card,
    borderColor: colors.danger,
    borderWidth: 1,
    borderRadius: 8,
    paddingVertical: 12,
    alignItems: 'center',
  },
  logoutText: {
    color: colors.danger,
    fontSize: 15,
    fontWeight: '600',
  },
  versionText: {
    marginTop: spacing.lg,
    textAlign: 'center',
    fontSize: 12,
    color: colors.muted,
  },
});
