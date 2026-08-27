package com.axmipusher.poller

import com.facebook.react.ReactPackage
import com.facebook.react.bridge.NativeModule
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.uimanager.ViewManager

/**
 * 原生模块包: 向 React Native 注册 BackgroundPollerModule。
 * legacy bridge 包(ReactPackage 实现), RN 0.87 新架构经 interop 层自动兼容, 无需 TurboModule 代码。
 */
class BackgroundPollerPackage : ReactPackage {

  override fun createNativeModules(reactContext: ReactApplicationContext): List<NativeModule> =
      listOf(BackgroundPollerModule(reactContext))

  override fun createViewManagers(reactContext: ReactApplicationContext): List<ViewManager<*, *>> =
      emptyList()
}
