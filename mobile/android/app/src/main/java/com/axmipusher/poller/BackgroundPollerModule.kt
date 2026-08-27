package com.axmipusher.poller

import android.content.Context
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import com.facebook.react.bridge.Promise
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.bridge.ReactContextBaseJavaModule
import com.facebook.react.bridge.ReactMethod
import com.facebook.react.bridge.ReadableMap
import java.util.concurrent.TimeUnit

/**
 * 后台轮询模块(legacy bridge 模块, RN 0.87 新架构经 interop 层兼容)。
 *
 * 职责: 将用户填写的服务器地址 / JWT / 轮询间隔存入 SharedPreferences,
 *       并按间隔调度 WorkManager 周期任务(PollWorker)。真正的网络请求只在
 *       PollWorker.doWork() 的后台线程里进行, 本模块不发起任何网络请求。
 *
 * JS 侧调用方式(必须一致):
 *   const BP = NativeModules.BackgroundPoller;
 *   await BP.configure({serverUrl, token, pollIntervalMinutes});
 *   await BP.start();  await BP.stop();
 *   await BP.setLastMaxId(id);  await BP.getLastMaxId();  await BP.isConfigured();
 */
class BackgroundPollerModule(reactContext: ReactApplicationContext) :
    ReactContextBaseJavaModule(reactContext) {

  companion object {
    /** 模块名, JS 侧 NativeModules.BackgroundPoller 必须与此一致 */
    const val NAME = "BackgroundPoller"

    /** SharedPreferences 文件名, Module 与 Worker 共用同一份 */
    const val PREFS_NAME = "axmipusher_poller"

    /** WorkManager 周期任务唯一名, 用 enqueueUniquePeriodicWork 避免重复 */
    const val UNIQUE_WORK_NAME = "axmipusher_poll"

    /** WorkManager 周期任务最小间隔(分钟), 低于该值按此值调度 */
    const val MIN_INTERVAL_MINUTES = 15L

    // SharedPreferences 键(与 PollWorker 共用)
    const val KEY_SERVER_URL = "server_url"
    const val KEY_TOKEN = "token"
    const val KEY_LAST_MAX_ID = "last_max_id"
    const val KEY_POLL_INTERVAL_MINUTES = "poll_interval_minutes"
  }

  private val prefs
    get() = reactApplicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

  override fun getName(): String = NAME

  /** 保存配置并按新间隔调度周期任务 */
  @ReactMethod
  fun configure(config: ReadableMap, promise: Promise) {
    try {
      val serverUrl = config.getString("serverUrl") ?: ""
      val token = config.getString("token") ?: ""
      val pollIntervalMinutes = config.getDouble("pollIntervalMinutes").toLong()
      if (serverUrl.isBlank() || token.isBlank()) {
        promise.reject("E_INVALID_CONFIG", "serverUrl 与 token 不能为空")
        return
      }
      prefs.edit()
          .putString(KEY_SERVER_URL, serverUrl)
          .putString(KEY_TOKEN, token)
          .putLong(KEY_POLL_INTERVAL_MINUTES, pollIntervalMinutes)
          .apply()
      schedule()
      promise.resolve(null)
    } catch (e: Exception) {
      promise.reject("E_CONFIGURE", e)
    }
  }

  /** 用当前配置调度周期任务(未配置则 no-op) */
  @ReactMethod
  fun start(promise: Promise) {
    try {
      if (!isConfigured()) {
        promise.resolve(null) // 未配置, no-op
        return
      }
      schedule()
      promise.resolve(null)
    } catch (e: Exception) {
      promise.reject("E_START", e)
    }
  }

  /** 取消 WorkManager 周期任务并清空全部配置(含 server_url / token / 游标) */
  @ReactMethod
  fun stop(promise: Promise) {
    try {
      WorkManager.getInstance(reactApplicationContext).cancelUniqueWork(UNIQUE_WORK_NAME)
      prefs.edit().clear().apply()
      promise.resolve(null)
    } catch (e: Exception) {
      promise.reject("E_STOP", e)
    }
  }

  /** 写入已处理的最新消息 id(轮询游标) */
  @ReactMethod
  fun setLastMaxId(id: Double, promise: Promise) {
    try {
      prefs.edit().putLong(KEY_LAST_MAX_ID, id.toLong()).apply()
      promise.resolve(null)
    } catch (e: Exception) {
      promise.reject("E_SET_LAST_MAX_ID", e)
    }
  }

  /** 读取当前轮询游标(最新已处理消息 id) */
  @ReactMethod
  fun getLastMaxId(promise: Promise) {
    try {
      promise.resolve(prefs.getLong(KEY_LAST_MAX_ID, 0L).toDouble())
    } catch (e: Exception) {
      promise.reject("E_GET_LAST_MAX_ID", e)
    }
  }

  /** 是否已配置(server_url 与 token 均非空) */
  @ReactMethod
  fun isConfigured(promise: Promise) {
    try {
      promise.resolve(isConfigured())
    } catch (e: Exception) {
      promise.reject("E_IS_CONFIGURED", e)
    }
  }

  private fun isConfigured(): Boolean {
    val serverUrl = prefs.getString(KEY_SERVER_URL, null)
    val token = prefs.getString(KEY_TOKEN, null)
    return !serverUrl.isNullOrBlank() && !token.isNullOrBlank()
  }

  /** 按 poll_interval_minutes 调度周期任务; 低于最小间隔时按最小间隔调度 */
  private fun schedule() {
    var intervalMinutes = prefs.getLong(KEY_POLL_INTERVAL_MINUTES, MIN_INTERVAL_MINUTES)
    if (intervalMinutes < MIN_INTERVAL_MINUTES) {
      intervalMinutes = MIN_INTERVAL_MINUTES
    }
    val request =
        PeriodicWorkRequestBuilder<PollWorker>(intervalMinutes, TimeUnit.MINUTES).build()
    WorkManager.getInstance(reactApplicationContext)
        .enqueueUniquePeriodicWork(
            UNIQUE_WORK_NAME, ExistingPeriodicWorkPolicy.UPDATE, request)
  }
}
