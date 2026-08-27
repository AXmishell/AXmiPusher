/**
 * 根导航。
 *
 * 按认证状态切换:
 * - bootstrapping → Loading(启动加载);
 * - signedOut → Login(登录页);
 * - signedIn → Main(底部 Tabs: 站内信 + 设置)。
 *
 * 使用 React Navigation 7 组件式 API:
 * NavigationContainer + createNativeStackNavigator + createBottomTabNavigator。
 */
import React from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { NavigationContainer } from '@react-navigation/native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { useAuth } from '../context/AuthContext';
import InboxScreen from '../screens/InboxScreen';
import LoginScreen from '../screens/LoginScreen';
import SettingsScreen from '../screens/SettingsScreen';
import { colors } from '../theme';
import type { MainTabParamList, RootStackParamList } from './types';

const RootStack = createNativeStackNavigator<RootStackParamList>();
const Tab = createBottomTabNavigator<MainTabParamList>();

/** 启动加载页。 */
function LoadingScreen() {
  return (
    <View style={styles.loading}>
      <ActivityIndicator size="large" color={colors.primary} />
      <Text style={styles.loadingText}>正在加载...</Text>
    </View>
  );
}

/** 已登录主界面: 底部两个页签。 */
function MainTabs() {
  return (
    <Tab.Navigator
      screenOptions={{
        headerTitleAlign: 'center',
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.muted,
      }}
    >
      <Tab.Screen
        name="Inbox"
        component={InboxScreen}
        options={{ title: '站内信' }}
      />
      <Tab.Screen
        name="Settings"
        component={SettingsScreen}
        options={{ title: '设置' }}
      />
    </Tab.Navigator>
  );
}

/** 根导航: 按认证状态渲染对应页面。 */
export default function RootNavigator() {
  const { status } = useAuth();
  return (
    <NavigationContainer>
      <RootStack.Navigator screenOptions={{ headerShown: false }}>
        {status === 'bootstrapping' ? (
          <RootStack.Screen name="Loading" component={LoadingScreen} />
        ) : status === 'signedIn' ? (
          <RootStack.Screen name="Main" component={MainTabs} />
        ) : (
          <RootStack.Screen name="Login" component={LoginScreen} />
        )}
      </RootStack.Navigator>
    </NavigationContainer>
  );
}

const styles = StyleSheet.create({
  loading: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: colors.background,
  },
  loadingText: {
    marginTop: 12,
    color: colors.muted,
    fontSize: 14,
  },
});
