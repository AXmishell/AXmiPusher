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
- Python 3.9+(可选, 仅 APK 签名工具 `scripts/android_signer.py` 需要, 纯标准库)

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

### APK 签名工具(推荐, 一键完成全部签名流程)

`scripts/android_signer.py`(纯 Python 标准库, 零第三方依赖, 跨平台 Windows/macOS/Linux, 中文交互)封装了 keystore 生成 → 配置 → 构建签名 → 验证 → 导出 GitHub Secret → 备份的完整流程。自动探测 keytool / apksigner / gradlew / JDK(优先 Java 17, 满足 RN 构建工具链要求)。

**子命令总览**(均支持 `--help`):

| 子命令 | 功能 |
|---|---|
| `generate` | 生成新 keystore(交互式填写证书 DN, 仅 CN 必填)+ 写 `keystore.properties` + 导出 base64 + 打印 GitHub Secret 配置表 |
| `sign` | 构建并签名 release APK(gradlew assembleRelease, 自动注入 JDK 17) |
| `verify` | 验证 APK 签名(apksigner), 与本地 keystore 指纹对比, 输出"正式签名 ✓"或警告 |
| `base64` | 导出 keystore 单行 base64(供 Secret `ANDROID_KEYSTORE_BASE64`) |
| `secrets` | 打印 GitHub Secrets 配置指引 |
| `backup` | 一键备份 keystore + base64 + README 到备份目录 |

**典型流程**:

```bash
cd mobile

# 1. 首次生成正式签名(交互式; 密码 <6 位会被拒绝; 已存在 keystore 时询问确认并自动备份旧钥)
python scripts/android_signer.py generate
#    交互示例:
#      CN(组织/名称, 必填): 你的组织名        ← 必填
#      组织单位 OU(回车跳过):                 ← 可选, 隐私起见脚本不内置默认值
#      ...(其余字段同理, 回车跳过)
#    生成后自动输出 GitHub Secret 配置表(密码仅显示这一次)

# 2. 构建并签名 APK(自动探测 JDK 17; --version 可指定版本号)
python scripts/android_signer.py sign --version 1.2.3

# 3. 验证签名
python scripts/android_signer.py verify     # 输出「正式签名 ✓」即成功

# 4. 一键备份签名材料(务必妥善离线保管!)
python scripts/android_signer.py backup

# 5. 导出 base64 / 查看 Secret 指引(配置 CI 正式签名时用)
python scripts/android_signer.py base64
python scripts/android_signer.py secrets
```

**安全与隐私说明**:
- 证书 DN **不固化在脚本中**, 每次 `generate` 交互式输入(仅 CN 必填), 避免组织/地址等隐私信息出现在代码里
- keystore 密码自动生成时仅打印一次; 所有密码/密钥文件不写日志、不入库(`.gitignore` 已忽略 `keystore.properties` / `*.keystore` / `*.jks`)
- 已存在 keystore 时 `generate` 会先备份为 `release.keystore.old-<时间戳>` 再重建, 不会静默覆盖
- 工具探测不到 keytool/apksigner/JDK 时给出明确中文指引(设置 `JAVA_HOME` / `ANDROID_HOME`)

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
