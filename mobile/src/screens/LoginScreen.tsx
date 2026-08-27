/**
 * 登录页。
 *
 * - 三输入框: 服务器地址 / 邮箱 / 密码;
 * - 登录成功自动进入主界面(由 AuthContext 切换导航);
 * - 账户开启两步验证时切换为 TOTP 验证码输入视图;
 * - 错误统一用 Alert 中文提示; 启动时从保存的会话预填服务器地址与邮箱。
 */
import React, { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { normalizeServerUrl } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { loadSession } from '../storage/settings';
import { colors, spacing } from '../theme';

export default function LoginScreen() {
  const { login, loginTotp, expiredNotice } = useAuth();
  // 输入框状态
  const [serverUrl, setServerUrl] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  // TOTP 两步验证状态
  const [totpMode, setTotpMode] = useState(false);
  const [totpToken, setTotpToken] = useState('');
  const [code, setCode] = useState('');
  const [totpLoading, setTotpLoading] = useState(false);

  // 启动时从保存的会话预填服务器地址与邮箱(不预填密码)
  useEffect(() => {
    loadSession()
      .then(saved => {
        if (saved) {
          setServerUrl(saved.serverUrl);
          setEmail(saved.email);
        }
      })
      .catch(() => {
        // 读取失败忽略
      });
  }, []);

  // 登录过期提示(启动时 401 自动登出后展示)
  useEffect(() => {
    if (expiredNotice) {
      Alert.alert('提示', expiredNotice);
    }
  }, [expiredNotice]);

  /** 第一步登录。 */
  const onLoginPress = async () => {
    if (!serverUrl.trim()) {
      Alert.alert('提示', '请输入服务器地址');
      return;
    }
    if (!email.trim()) {
      Alert.alert('提示', '请输入邮箱');
      return;
    }
    if (!password) {
      Alert.alert('提示', '请输入密码');
      return;
    }
    setLoading(true);
    try {
      const data = await login(normalizeServerUrl(serverUrl), email.trim(), password);
      // 需两步验证: 切换到验证码输入视图
      if ('need_totp' in data && data.need_totp === true) {
        setTotpToken(data.totp_token);
        setTotpMode(true);
      }
      // 登录成功由 AuthContext 切换导航, 无需手动跳转
    } catch (err) {
      Alert.alert('登录失败', err instanceof Error ? err.message : '未知错误');
    } finally {
      setLoading(false);
    }
  };

  /** 第二步 TOTP 验证码提交。 */
  const onTotpSubmit = async () => {
    if (!code.trim()) {
      Alert.alert('提示', '请输入验证码');
      return;
    }
    setTotpLoading(true);
    try {
      await loginTotp(normalizeServerUrl(serverUrl), totpToken, code.trim());
      // 成功后由 AuthContext 切换导航
    } catch (err) {
      Alert.alert('验证失败', err instanceof Error ? err.message : '未知错误');
    } finally {
      setTotpLoading(false);
    }
  };

  /** 返回邮箱密码登录。 */
  const onBackToLogin = () => {
    setTotpMode(false);
    setTotpToken('');
    setCode('');
  };

  return (
    <SafeAreaView style={styles.safeArea}>
      <KeyboardAvoidingView
        style={styles.flex}
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
      >
        <ScrollView
          contentContainerStyle={styles.container}
          keyboardShouldPersistTaps="handled"
        >
          <Text style={styles.title}>AXmiPusher</Text>
          <Text style={styles.subtitle}>消息推送平台</Text>

          {!totpMode ? (
            <>
              <Text style={styles.label}>服务器地址</Text>
              <TextInput
                style={styles.input}
                value={serverUrl}
                onChangeText={setServerUrl}
                placeholder="例如 192.168.1.5:8080"
                placeholderTextColor={colors.muted}
                autoCapitalize="none"
                autoCorrect={false}
                keyboardType="url"
              />
              <Text style={styles.label}>邮箱</Text>
              <TextInput
                style={styles.input}
                value={email}
                onChangeText={setEmail}
                placeholder="请输入邮箱"
                placeholderTextColor={colors.muted}
                autoCapitalize="none"
                autoCorrect={false}
                keyboardType="email-address"
              />
              <Text style={styles.label}>密码</Text>
              <TextInput
                style={styles.input}
                value={password}
                onChangeText={setPassword}
                placeholder="请输入密码"
                placeholderTextColor={colors.muted}
                secureTextEntry
              />
              <Pressable
                style={[styles.button, loading && styles.buttonDisabled]}
                onPress={onLoginPress}
                disabled={loading}
              >
                {loading ? (
                  <ActivityIndicator color="#ffffff" />
                ) : (
                  <Text style={styles.buttonText}>登录</Text>
                )}
              </Pressable>
            </>
          ) : (
            <>
              <View style={styles.totpNotice}>
                <Text style={styles.totpNoticeText}>已开启两步验证</Text>
                <Text style={styles.totpNoticeDesc}>
                  请输入手机验证器生成的 6 位动态验证码
                </Text>
              </View>
              <Text style={styles.label}>验证码</Text>
              <TextInput
                style={styles.input}
                value={code}
                onChangeText={setCode}
                placeholder="6 位验证码"
                placeholderTextColor={colors.muted}
                keyboardType="number-pad"
                maxLength={6}
              />
              <Pressable
                style={[styles.button, totpLoading && styles.buttonDisabled]}
                onPress={onTotpSubmit}
                disabled={totpLoading}
              >
                {totpLoading ? (
                  <ActivityIndicator color="#ffffff" />
                ) : (
                  <Text style={styles.buttonText}>完成登录</Text>
                )}
              </Pressable>
              <Pressable style={styles.backLink} onPress={onBackToLogin}>
                <Text style={styles.backLinkText}>返回邮箱密码登录</Text>
              </Pressable>
            </>
          )}
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  flex: {
    flex: 1,
  },
  container: {
    flexGrow: 1,
    justifyContent: 'center',
    padding: spacing.xl,
  },
  title: {
    fontSize: 32,
    fontWeight: 'bold',
    color: colors.primary,
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 14,
    color: colors.muted,
    textAlign: 'center',
    marginTop: spacing.xs,
    marginBottom: spacing.xl,
  },
  label: {
    fontSize: 14,
    color: colors.text,
    marginTop: spacing.md,
    marginBottom: spacing.xs,
  },
  input: {
    backgroundColor: colors.card,
    borderColor: colors.border,
    borderWidth: 1,
    borderRadius: 8,
    paddingHorizontal: spacing.md,
    paddingVertical: 10,
    fontSize: 15,
    color: colors.text,
  },
  button: {
    marginTop: spacing.xl,
    backgroundColor: colors.primary,
    borderRadius: 8,
    paddingVertical: 12,
    alignItems: 'center',
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  buttonText: {
    color: '#ffffff',
    fontSize: 16,
    fontWeight: '600',
  },
  totpNotice: {
    backgroundColor: '#eff6ff',
    borderColor: colors.primary,
    borderWidth: 1,
    borderRadius: 8,
    padding: spacing.md,
    marginBottom: spacing.sm,
  },
  totpNoticeText: {
    color: colors.primary,
    fontSize: 15,
    fontWeight: '600',
  },
  totpNoticeDesc: {
    color: colors.muted,
    fontSize: 13,
    marginTop: spacing.xs,
  },
  backLink: {
    marginTop: spacing.lg,
    alignItems: 'center',
  },
  backLinkText: {
    color: colors.primary,
    fontSize: 14,
  },
});
