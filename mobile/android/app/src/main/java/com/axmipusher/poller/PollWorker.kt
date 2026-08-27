package com.axmipusher.poller

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.work.Worker
import androidx.work.WorkerParameters
import com.axmipusher.MainActivity
import java.io.BufferedReader
import java.io.InputStream
import java.io.InputStreamReader
import java.net.HttpURLConnection
import java.net.URL
import java.nio.charset.StandardCharsets
import org.json.JSONArray
import org.json.JSONObject

/**
 * 后台轮询 Worker。由 WorkManager 周期调度(PeriodicWorkRequest), 即使应用被杀/在后台
 * 也能继续工作(WorkManager 由系统进程调度, 重启手机后自动恢复周期任务)。
 *
 * 轮询流程:
 *   a) 读 SharedPreferences, 无 server_url 或 token → 静默成功返回;
 *   b) GET {serverUrl}/api/v1/inbox/unread-count(Bearer token, 15s 超时);
 *   c) HTTP 401 → token 过期, 清除 prefs 里的 token(保留 server_url), 成功返回;
 *   d) 解析 {"code":0,"data":{"unread":N}}, unread<=0 → 成功返回;
 *   e) unread>0 → GET {serverUrl}/api/v1/inbox?read=false&current=1&pageSize=50, 解析 data.data[];
 *   f) 只处理 id > last_max_id 的新消息, 每条发系统通知(通知 id=消息 id 避免重复);
 *   g) 最新消息 id 写回 last_max_id;
 *   h) 网络/解析异常一律 catch 后返回 retry/success, 绝不 crash。
 *
 * Worker 本身运行在 WorkManager 的后台线程, 网络请求直接在这里做, 无需协程。
 */
class PollWorker(appContext: Context, workerParams: WorkerParameters) :
    Worker(appContext, workerParams) {

  companion object {
    private const val TAG = "PollWorker"

    /** 通知渠道 id/名称(站内信) */
    private const val CHANNEL_ID = "inbox"
    private const val CHANNEL_NAME = "站内信"

    /** HTTP 超时(毫秒) */
    private const val TIMEOUT_MS = 15_000

    /** 未读消息列表分页参数 */
    private const val PAGE_SIZE = 50
  }

  override fun doWork(): Result {
    val prefs = applicationContext.getSharedPreferences(
        BackgroundPollerModule.PREFS_NAME, Context.MODE_PRIVATE)

    // a) 未配置(无 server_url 或 token) → 静默成功, 不做任何事
    val serverUrl = prefs.getString(BackgroundPollerModule.KEY_SERVER_URL, null)
    val token = prefs.getString(BackgroundPollerModule.KEY_TOKEN, null)
    if (serverUrl.isNullOrBlank() || token.isNullOrBlank()) {
      Log.d(TAG, "未配置 server_url 或 token, 跳过本次轮询")
      return Result.success()
    }

    try {
      // b) 拉取未读数
      val (unreadCode, unreadBody) = httpGet("$serverUrl/api/v1/inbox/unread-count", token)
      if (unreadCode == 401) {
        // c) token 过期: 清除 token(保留 server_url), 下次打开应用 JS 检测到未配置会引导重新登录
        prefs.edit().remove(BackgroundPollerModule.KEY_TOKEN).apply()
        Log.w(TAG, "未读数接口返回 401, 已清除 token(保留 server_url), 等待重新登录")
        return Result.success()
      }
      if (unreadCode !in 200..299) {
        Log.w(TAG, "未读数接口返回 HTTP $unreadCode, 稍后重试")
        return Result.retry()
      }
      // d) 解析未读数, unread<=0 → 无新消息, 成功返回
      val unreadJson = JSONObject(unreadBody)
      val unread = unreadJson.optJSONObject("data")?.optInt("unread", 0) ?: 0
      if (unread <= 0) {
        return Result.success()
      }

      // e) 拉取未读消息列表
      val listUrl = "$serverUrl/api/v1/inbox?read=false&current=1&pageSize=$PAGE_SIZE"
      val (listCode, listBody) = httpGet(listUrl, token)
      if (listCode == 401) {
        prefs.edit().remove(BackgroundPollerModule.KEY_TOKEN).apply()
        Log.w(TAG, "列表接口返回 401, 已清除 token(保留 server_url)")
        return Result.success()
      }
      if (listCode !in 200..299) {
        Log.w(TAG, "列表接口返回 HTTP $listCode, 稍后重试")
        return Result.retry()
      }
      val listJson = JSONObject(listBody)
      val items = listJson.optJSONObject("data")?.optJSONArray("data") ?: JSONArray()
      if (items.length() == 0) {
        return Result.success()
      }

      // f) 只处理 id > last_max_id 的新消息, 逐条发系统通知
      ensureNotificationChannel()
      var lastMaxId = prefs.getLong(BackgroundPollerModule.KEY_LAST_MAX_ID, 0L)
      var newMaxId = lastMaxId
      for (i in 0 until items.length()) {
        val item = items.optJSONObject(i) ?: continue
        val id = item.optLong("id", 0L)
        if (id <= lastMaxId) {
          continue // 非新消息, 跳过
        }
        val title = item.optString("title", "站内信")
        val content = item.optString("content", "")
        postNotification(id, title, content)
        if (id > newMaxId) {
          newMaxId = id
        }
      }

      // g) 把最新消息 id 写回 last_max_id(轮询游标)
      if (newMaxId > lastMaxId) {
        prefs.edit().putLong(BackgroundPollerModule.KEY_LAST_MAX_ID, newMaxId).apply()
        Log.d(TAG, "轮询完成, 已处理到消息 id=$newMaxId")
      }
      return Result.success()
    } catch (e: Exception) {
      // h) 网络/解析异常一律 catch, 绝不 crash; 返回 retry 交给 WorkManager 带退避重试
      Log.w(TAG, "轮询异常, 稍后重试", e)
      return Result.retry()
    }
  }

  /** 发送 GET 请求, 返回 (HTTP 状态码, 响应体)。15 秒连接/读取超时。 */
  private fun httpGet(urlString: String, token: String): Pair<Int, String> {
    val conn = URL(urlString).openConnection() as HttpURLConnection
    try {
      conn.requestMethod = "GET"
      conn.connectTimeout = TIMEOUT_MS
      conn.readTimeout = TIMEOUT_MS
      conn.setRequestProperty("Accept", "application/json")
      conn.setRequestProperty("Authorization", "Bearer $token")
      val code = conn.responseCode
      // 2xx 读 inputStream, 否则读 errorStream(能拿到错误详情但不影响判断)
      val stream: InputStream? = if (code in 200..299) conn.inputStream else conn.errorStream
      val body = stream?.let { readAll(it) } ?: ""
      return code to body
    } finally {
      conn.disconnect()
    }
  }

  /** 把 InputStream 按 UTF-8 完整读出 */
  private fun readAll(stream: InputStream): String {
    val reader = BufferedReader(InputStreamReader(stream, StandardCharsets.UTF_8))
    return reader.use { it.readText() }
  }

  /** 确保"站内信"通知渠道存在(仅 API 26+ 需要, 幂等) */
  private fun ensureNotificationChannel() {
    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
      val manager =
          applicationContext.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
      val channel =
          NotificationChannel(CHANNEL_ID, CHANNEL_NAME, NotificationManager.IMPORTANCE_HIGH)
      channel.description = "新站内信到达时推送通知"
      manager.createNotificationChannel(channel)
    }
  }

  /** 为新消息发系统通知栏通知; 通知 id 用消息 id(同一消息不会重复通知) */
  private fun postNotification(id: Long, title: String, content: String) {
    val manager = NotificationManagerCompat.from(applicationContext)
    // API 33+ 未授予 POST_NOTIFICATIONS 权限或用户全局关闭通知时静默跳过(不能在此申请权限)
    if (!manager.areNotificationsEnabled()) {
      Log.d(TAG, "通知未授权或被禁用, 跳过消息 id=$id")
      return
    }
    val intent = Intent(applicationContext, MainActivity::class.java)
    val pendingIntent = PendingIntent.getActivity(
        applicationContext,
        (id % Int.MAX_VALUE).toInt(), // 通知 id 用消息 id(取模防溢出, 同一消息稳定)
        intent,
        PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)

    val notification =
        NotificationCompat.Builder(applicationContext, CHANNEL_ID)
            .setSmallIcon(android.R.drawable.stat_notify_chat)
            .setContentTitle(title)
            .setContentText(content)
            .setStyle(NotificationCompat.BigTextStyle().bigText(content))
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()
    manager.notify((id % Int.MAX_VALUE).toInt(), notification)
  }
}
