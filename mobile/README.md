# AXmiPusher Android 客户端

基于 **React Native 0.87** 的 AXmiPusher 消息推送平台 Android 客户端。核心功能:登录(服务器地址 + 邮箱 + 密码,支持 TOTP 两步验证)、接收**站内信**并推送到手机通知栏、站内信列表与已读管理。

## 功能

- **登录**:配置服务器地址(如 `192.168.1.5:8080`)、邮箱、密码;账户开启两步验证时自动进入 TOTP 验证码流程;JWT 存于系统安全存储(keychain/EncryptedSharedPreferences)
- **站内信列表**:未读加粗 + 蓝点标识、下拉刷新、上拉分页加载、点击展开内容并自动标记已读、全部已读、底部页签未读角标
- **通知推送(核心)**:
  - **前台**:每 30 秒轮询一次未读数与站内信(应用打开时)
  - **后台/被杀**:原生 WorkManager Worker 按设置间隔(15/30/60 分钟,Android 平台最小 15 分钟)轮询并发送系统通知;点通知打开应用进入站内信列表;手机重启后自动恢复
  - **去重**:JS 与原生 Worker 通过 SharedPreferences 共享 `last_max_id` 游标,同一消息不会重复通知
- **设置**:查看账号/服务器信息、调整后台轮询间隔、退出登录

## 技术架构

| 层 | 技术 |
|---|---|
| 框架 | React Native 0.87.1(新架构)+ TypeScript |
| 导航 | React Navigation 7(native-stack + bottom-tabs) |
| 安全存储 | react-native-keychain(JWT 会话) |
| 后台轮询 | AndroidX WorkManager 周期任务(原生 Kotlin 模块 `BackgroundPoller`,legacy bridge) |
| 通知 | Android 系统通知(NotificationChannel "站内信",原生 Worker 直接发送) |

```
mobile/
├── android/                       # 原生工程
│   └── app/src/main/java/com/axmipusher/poller/
│       ├── BackgroundPollerModule.kt   # JS 桥(configure/start/stop/setLastMaxId/...)
│       ├── PollWorker.kt               # WorkManager 周期轮询 + 系统通知
│       └── BackgroundPollerPackage.kt  # ReactPackage 注册
├── src/
│   ├── api/                       # types / client(fetch 封装) / auth / inbox
│   ├── context/AuthContext.tsx    # 会话恢复、登录(含 TOTP)、401 自动登出、原生轮询编排
│   ├── services/poller.ts         # 前台 30s 轮询 + 去重 + lastMaxId 同步
│   ├── navigation/                # 根导航(登录 / 主 Tabs)
│   ├── screens/                   # LoginScreen / InboxScreen / SettingsScreen
│   ├── storage/settings.ts        # keychain 会话持久化
│   └── native/BackgroundPoller.ts # 原生模块类型声明 + 空实现兜底
└── App.tsx                        # 根组件
```

## 环境要求

- Node.js ≥ 22
- JDK 17 或 21(推荐使用 Android Studio 自带的 JBR)
- Android SDK(compileSdk 37 / targetSdk 36 / minSdk 24)
- Android Studio(建议,自带 SDK 与模拟器)

## 构建与运行

```bash
cd mobile
npm install          # registry 已配置 npmmirror(mobile/.npmrc)
npm start            # 启动 Metro

# 另开终端: 连接设备/模拟器后
npm run android      # debug 构建并安装

# 打 release APK(需先配置签名, 见下方"正式签名"; 未配置时自动回退 debug 签名)
cd android && ./gradlew assembleRelease   # 输出 android/app/build/outputs/apk/release/

# 指定版本号(默认 1.0; CI 推 tag 时自动注入, 如 APP_VERSION=1.2.3 → versionCode 10203)
# Windows PowerShell: $env:APP_VERSION="1.2.3"; 再执行 gradlew assembleRelease
```

> 本机开发注意:release 变体已通过 `manifestPlaceholders` + `finalizeDsl` 回调 + `src/release/AndroidManifest.xml`(tools:replace)三重保障允许 **http 明文**(连接内网 `http://192.168.x.x:8080` 自建服务器),使用 https 域名同样正常。

### 正式签名(可选, 推荐)

release 默认回退使用模板 debug keystore(可安装测试)。要正式签名, 在 `android/app/` 下放置:

```
keystore.properties(不入库, 已被 .gitignore 忽略)
  storeFile=release.keystore
  storePassword=***
  keyAlias=***
  keyPassword=***
```

生成 keystore 示例(`keytool` 随 JDK 提供):

```bash
keytool -genkeypair -v -keystore release.keystore -alias axmipusher \
  -keyalg RSA -keysize 2048 -validity 36500
```

**CI 发布签名**:推 `v*` tag 时 `.github/workflows/release.yml` 自动构建并发布 APK 到 GitHub Releases。如需正式签名, 在仓库 Settings → Secrets and variables 配置 4 个 Secret:

| Secret | 说明 |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | `base64 release.keystore` 的输出 |
| `ANDROID_KEYSTORE_PASSWORD` | keystore 密码 |
| `ANDROID_KEY_ALIAS` | 别名(如上例 `axmipusher`) |
| `ANDROID_KEY_PASSWORD` | 别名密码 |

未配置 Secret 时 CI 出 debug 签名包(仍可安装)。**注意**:正式签名 keystore 务必妥善保管并长期保留, 丢失后无法再对老用户推送升级。

## 使用说明

1. 打开应用,填写:**服务器地址**(AXmiPusher 主程序地址,如 `http://192.168.1.5:8080`,不填协议自动补 `http://`)、**邮箱**、**密码**
2. 若账号开启了两步验证,按提示输入 6 位 TOTP 验证码
3. 登录后自动进入站内信列表;后台轮询 Worker 同步启动
4. Android 13+ 首次会请求**通知权限**,请允许(否则后台收不到通知)
5. 设置页可调整后台轮询间隔(15/30/60 分钟)

## 与后端 API 的对应关系

| 应用行为 | 后端接口(均需 `Authorization: Bearer <JWT>`) |
|---|---|
| 登录 | `POST /api/v1/auth/login`(开 TOTP 时返回 `need_totp`+`totp_token`) |
| TOTP 二阶段 | `POST /api/v1/auth/login/totp` |
| 会话校验 | `GET /api/v1/auth/me` |
| 站内信列表 | `GET /api/v1/inbox?current&pageSize&read` |
| 未读数 | `GET /api/v1/inbox/unread-count` |
| 标记已读 | `PUT /api/v1/inbox/:id/read` |
| 全部已读 | `PUT /api/v1/inbox/read-all` |

> 站内信接口仅接受**用户 JWT**(`kind=user`),服务端 API Key 不可用;JWT 有效期 24 小时,过期后应用自动登出并提示重新登录。

## 已知限制

- 后台推送依赖轮询,受 Android 平台限制后台周期任务**最小间隔 15 分钟**(WorkManager);前台即时刷新(30 秒)不受影响。若需毫秒级实时推送,需服务端支持设备注册 + FCM(本项目后端当前无此能力)
- 应用被系统强行杀死后,WorkManager 任务仍由系统调度;个别厂商(小米/华为等)需在系统设置中允许应用自启动/后台运行
