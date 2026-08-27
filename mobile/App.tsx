/**
 * AXmiPusher 消息推送平台 Android 客户端 — 应用根组件。
 *
 * 结构: SafeAreaProvider(刘海屏适配) > AuthProvider(会话/认证) > RootNavigator(根导航)。
 * 替换默认模板 App, 完整应用层位于 src/ 目录。
 */
import React from 'react';
import { StatusBar } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { AuthProvider } from './src/context/AuthContext';
import RootNavigator from './src/navigation';

/**
 * 应用根组件。
 */
function App() {
  return (
    <SafeAreaProvider>
      <StatusBar barStyle="dark-content" />
      <AuthProvider>
        <RootNavigator />
      </AuthProvider>
    </SafeAreaProvider>
  );
}

export default App;
