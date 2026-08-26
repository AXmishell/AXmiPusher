# 鐢ㄦ埛涓績 API 鏂囨。

## 骞冲彴绠€浠?
AXmiPusher 鏄竴涓粺涓€鍙楃悊鐨勬秷鎭帹閫佸钩鍙般€傝皟鐢ㄦ柟鍙渶鎶婃秷鎭氦缁欏钩鍙? 骞冲彴璐熻矗鍚庣画涓€鍒? 娑堟伅杩涘叆闃熷垪, 鎸夐€氶亾閰嶇疆鍒嗗彂鍒?Webhook 鍥炶皟銆丒mail(SMTP)銆丄PNs(iOS)銆丗CM(Android)銆佺珯鍐呬俊绛夊涓笭閬? 骞跺洖鍐欏彂閫佺姸鎬佷笌鍥炴墽缁熻銆?
骞冲彴鏋舵瀯涓庢湰鏂囨。鐩稿叧鐨勪簨瀹?

- 鍚庣涓?Go + Gin, 鍗曡繘绋嬫壙杞?HTTP 鎺ュ彛涓庡唴缃暟鎹簱杞娑堣垂鑰呫€?- 娑堟伅鍙戦€佷负"鍙楃悊"璇箟: 鎻愪氦鍚庣珛鍗宠繑鍥炲彈鐞嗙粨鏋? 瀹為檯鍙戦€佺敱鍚庡彴娑堣垂鑰呭紓姝ュ畬鎴? 鍙€氳繃娑堟伅鐘舵€佷笌鍥炶皟鑾风煡缁撴灉銆?- 鐢ㄦ埛涓績(鏈墠绔?瀵瑰簲鐨勫叏閮ㄤ笟鍔?API 鍧囦负鏈枃妗ｈ寖鍥淬€傜鐞嗗悗鍙?`/admin/*`)API 涓嶅湪鏈枃妗ｈ寖鍥? 璇峰弬鑰冮儴缃蹭晶绠＄悊鏂囨。銆?- 鎵€鏈変笟鍔?API 瑕佹眰骞冲彴宸插畬鎴愬畨瑁?鏈畨瑁呮椂杩斿洖 503, 闇€鍏堣闂?`/install` 瀹屾垚瀹夎鍚戝)銆?
### 閫傜敤璇昏€?
鏈枃妗ｉ潰鍚戦渶瑕侀€氳繃 API 闆嗘垚 AXmiPusher 鐨勭涓夋柟寮€鍙戣€? 鍖呮嫭涓ょ被鍦烘櫙:

1. 缃戦〉闆嗘垚: 浣跨敤娉ㄥ唽/鐧诲綍鑾峰彇鐨勭敤鎴?JWT 璋冪敤鍏ㄩ儴涓氬姟 API(鐢ㄦ埛涓績鍚屾鑳藉姏)銆?2. 鏈嶅姟绔泦鎴? 浣跨敤 API Key(`mp_` 鍓嶇紑)浠呰皟鐢ㄦ秷鎭彂閫併€佹秷鎭煡璇㈢瓑鎺堟潈绔偣, 鏃犻渶鎸佹湁鐢ㄦ埛璐﹀彿瀵嗙爜銆?
鏂囨。涓瘡涓鐐圭殑璇锋眰涓庡搷搴斿瓧娈靛潎浠ュ綋鍓嶄唬鐮佷负鍑? 鍙洿鎺ョ収鐫€璋冮€氥€?
---

## 1. 鍩虹淇℃伅

### Base URL

- 鐢熶骇鐜涓荤▼搴忛粯璁ょ鍙?`8080`, 鎵€鏈変笟鍔?API 甯﹁矾寰勫墠缂€ `/api/v1`:

```
http://<host>:8080/api/v1
```

- 鍏紑鍋ュ悍妫€鏌? `GET http://<host>:8080/api/v1/health`
- 骞冲彴鏈畨瑁呮椂, 涓氬姟 API 杩斿洖 503, 娴忚鍣ㄨ闂?`/install` 鍙繘鍏ュ畨瑁呭悜瀵笺€?
### 鍝嶅簲缁熶竴鏍煎紡

鎵€鏈変笟鍔?API(闄ゆ秷鎭彂閫佸彈鐞嗐€丼erver閰卞吋瀹瑰眰銆佹敮浠樺洖璋冨)缁熶竴杩斿洖濡備笅 JSON 缁撴瀯:

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

- `code`: 涓氬姟鐮? `0` 琛ㄧず鎴愬姛, 闈?0 琛ㄧず澶辫触銆?- `message`: 浜虹被鍙鐨勮鏄? 澶辫触鏃舵槸鍏蜂綋閿欒淇℃伅銆?- `data`: 涓氬姟鏁版嵁, 澶辫触鏃堕€氬父涓?`null` 鎴栫己澶便€?
### 涓氬姟鐮佽〃

| 涓氬姟鐮?| HTTP 鐘舵€?| 鍚箟 |
|--------|-----------|------|
| 0 | 200 | 鎴愬姛 |
| 40000 | 400 | 鍙傛暟閿欒(璇锋眰浣撴牎楠屽け璐ャ€佸瓧娈电己澶?鏍煎紡閿欒銆佷笟鍔℃牎楠屼笉閫氳繃) |
| 40100 | 401 | 鏈璇佹垨鍑瘉鏃犳晥(缂哄皯/杩囨湡/閿欒鐨?JWT 鎴?API Key) |
| 40300 | 403 | 鏃犳潈闄愭墽琛屾鎿嶄綔 |
| 40400 | 404 | 璧勬簮涓嶅瓨鍦?|
| 40900 | 409 | 鍐茬獊(閭宸叉敞鍐屻€佹ā鏉?code 閲嶅銆佸箓绛夐噸澶嶇瓑) |
| 42900 | 429 | 闄愭祦(鍙戦€侀鐜囪秴闄? |
| 50000 | 500 | 鏈嶅姟绔敊璇?|
| 50300 | 503 | 骞冲彴灏氭湭瀹夎, 璇峰厛璁块棶 /install 瀹屾垚瀹夎 |

### HTTP 鐘舵€佺害瀹?
- 缁濆ぇ澶氭暟鎴愬姛鍝嶅簲涓?HTTP `200`銆?- 娑堟伅鍙戦€佸彈鐞?`POST /api/v1/messages` 鎴愬姛鏃惰繑鍥?HTTP `202 Accepted`, 琛ㄧず宸插彈鐞嗐€佽繘鍏ュ紓姝ュ彂閫佹祦绋嬨€?- Server閰卞吋瀹瑰眰涓庢敮浠樺洖璋冧笉閬靛惊涓婅堪涓氬姟鐮佹牸寮? 鍗曠嫭璇存槑(瑙佺 14 绔?銆?- 閿欒鏃朵笟鍔＄爜涓?HTTP 鐘舵€佸搴斿叧绯昏涓婅〃銆?
---

## 2. 璁よ瘉鏂瑰紡

骞冲彴鎻愪緵涓ょ鍑瘉, 鍧囬€氳繃璇锋眰澶翠紶閫?

```
Authorization: Bearer <token>
```

### (a) 鐢ㄦ埛 JWT

- 閫氳繃娉ㄥ唽鎴栫櫥褰曡幏鍙? 褰㈠ `eyJhbGciOi...`銆?- 鐢ㄤ簬鐢ㄦ埛涓績鐨勫叏閮ㄤ笟鍔?API(闇€鐧诲綍鐨勭鐐?銆?- JWT 鍐?`kind=user`, 涓庣鐞嗗憳 JWT(`kind=admin`)鍙屽悜闅旂: 鐢ㄦ埛 token 鏃犳硶璁块棶 `/admin/*` 绔偣, 绠＄悊鍛?token 涔熸棤娉曡闂敤鎴蜂腑蹇冪鐐广€?- TOTP 鏈惎鐢ㄦ椂鐧诲綍鐩存帴杩斿洖 `token`; 鍚敤鏃剁櫥褰曚负涓ら樁娈?瑙?3.3 / 3.4)銆?
### (b) 鏈嶅姟绔?API Key

- 鍦ㄧ敤鎴蜂腑蹇冨垱寤? 鏄庢枃浠?`mp_` 鍓嶇紑寮€澶?鏈嶅姟绔瓨鍌?SHA-256 鍝堝笇)銆?- 涓?JWT 浣跨敤鐩稿悓鐨勮姹傚ご鏍煎紡: `Authorization: Bearer <api_key>`銆?- **浠呴檺閮ㄥ垎绔偣**: 娑堟伅妯″潡 `POST /api/v1/messages`銆乣GET /api/v1/messages`銆乣GET /api/v1/messages/:id` 鍚屾椂鎺ュ彈鐢ㄦ埛 JWT 鎴?API Key(瑙佺 4 绔?銆傚叾浣欓渶鐧诲綍鐨勭鐐逛笉鎺ュ彈 API Key銆?- 鍒涘缓鏃舵槑鏂囦粎杩斿洖涓€娆? 璇峰Ε鍠勪繚瀛? 鏈嶅姟绔棤娉曞啀鏌ュ洖鏄庢枃銆?
### 401 璇箟

鏈惡甯﹀嚟璇併€佸嚟璇佹棤鏁堟垨宸茶繃鏈? 鍧囪繑鍥?

```json
{
  "code": 40100,
  "message": "缂哄皯璁よ瘉鍑瘉"  // 鎴?"鍑瘉鏃犳晥鎴栧凡杩囨湡" / "缂哄皯 API Key" / "API Key 鏃犳晥"
}
```

---

## 3. 璁よ瘉妯″潡(/auth)

鍏ㄩ儴璺敱鎸傚湪 `/api/v1/auth` 涓嬨€傛敞鍐?鐧诲綍/鍙戠爜/涓ゆ鐧诲綍鍏紑, 鍏朵綑闇€鐢ㄦ埛 JWT銆?
### 3.1 POST /api/v1/auth/register/send-code

鍙戦€佹敞鍐岄偖绠遍獙璇佺爜銆?
璇锋眰浣?

```json
{
  "email": "user@example.com"
}
```

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒(email 缂哄け鎴栭潪娉? |
| 40900 | 璇ラ偖绠卞凡娉ㄥ唽 |
| 42900 | 鍙戦€佽繃浜庨绻? 璇风◢鍚庡啀璇?60 绉掑喎鍗? |
| 50000 | 绯荤粺閭欢鏈厤缃? 璇疯仈绯荤鐞嗗憳; 鎴栧彂閫佸け璐?|

### 3.2 POST /api/v1/auth/register

娉ㄥ唽骞惰嚜鍔ㄧ櫥褰?娉ㄥ唽鍗崇櫥褰?銆?
璇锋眰浣?

```json
{
  "email": "user@example.com",
  "password": "your-password",
  "confirm_password": "your-password",
  "nickname": "鎴戠殑鐢ㄦ埛鍚?,
  "code": "123456"
}
```

瀛楁璇存槑:

- `email`(蹇呭～): 娉ㄥ唽閭銆?- `password`(蹇呭～): 瀵嗙爜銆?- `confirm_password`(鍙€?: 纭瀵嗙爜, 鎻愪緵鏃堕』涓?`password` 涓€鑷? 鍚﹀垯 400銆?- `nickname`(鍙€?: 鐢ㄦ埛鍚? 缂虹渷浣跨敤 email銆?- `code`(蹇呭～): 閭楠岃瘉鐮?鏉ヨ嚜 3.1 鐨?send-code)銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "token": "eyJhbGciOi...",
    "user": {
      "id": 1,
      "email": "user@example.com",
      "nickname": "鎴戠殑鐢ㄦ埛鍚?,
      "quota": "{}",
      "plan_id": 0,
      "role": "tenant_admin",
      "status": "active",
      "qq": "",
      "last_login_ip": "",
      "totp_enabled": false,
      "last_login_at": null,
      "created_at": "2026-08-27T10:00:00+08:00",
      "updated_at": "2026-08-27T10:00:00+08:00"
    }
  }
}
```

`user` 瀵硅薄涓哄綋鍓嶇敤鎴峰畬鏁翠俊鎭?`password_hash`銆乣totp_secret` 绛夋晱鎰熷瓧娈典笉鍥炰紶)銆?
澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒; confirm_password 涓?password 涓嶄竴鑷?涓ゆ杈撳叆鐨勫瘑鐮佷笉涓€鑷?; 楠岃瘉鐮侀敊璇垨宸茶繃鏈?|
| 40900 | 璇ラ偖绠卞凡娉ㄥ唽; 鍏朵粬鍒涘缓鍐茬獊 |
| 50000 | 绛惧彂浠ょ墝澶辫触 |

### 3.3 POST /api/v1/auth/login

瀵嗙爜鐧诲綍銆傝嫢鐢ㄦ埛宸插惎鐢?TOTP, 杩斿洖涓存椂鍑瘉杩涘叆绗簩姝? 鍚﹀垯鐩存帴杩斿洖 token銆?
璇锋眰浣?

```json
{
  "email": "user@example.com",
  "password": "your-password"
}
```

鎴愬姛鍝嶅簲(HTTP 200), 鏈惎鐢?TOTP:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "token": "eyJhbGciOi...",
    "user": { "...": "鍚?3.2 鐨?user 缁撴瀯" }
  }
}
```

鎴愬姛鍝嶅簲(HTTP 200), 宸插惎鐢?TOTP(涓ら樁娈电涓€姝?:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "need_totp": true,
    "totp_token": "eyJhbGciOi..."
  }
}
```

`totp_token` 涓?5 鍒嗛挓鏈夋晥鏈熺殑涓存椂鍑瘉, 闇€鍦?3.4 涓娇鐢ㄣ€傚惎鐢?TOTP 鏃舵湰姝ラ涓嶄細鏇存柊鏈€杩戠櫥褰曟椂闂? 鐧诲綍鐘舵€佷互绗簩姝ヤ负鍑嗐€?
澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒 |
| 40100 | 閭鎴栧瘑鐮侀敊璇?|
| 50000 | 绛惧彂楠岃瘉鍑瘉澶辫触 |

### 3.4 POST /api/v1/auth/login/totp

TOTP 绗簩姝ョ櫥褰? 鏍￠獙楠岃瘉鐮佸悗绛惧彂姝ｅ紡 token銆?
璇锋眰浣?

```json
{
  "totp_token": "eyJhbGciOi...",
  "code": "123456"
}
```

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "token": "eyJhbGciOi...",
    "user": { "...": "鍚?3.2 鐨?user 缁撴瀯" }
  }
}
```

澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒 |
| 40100 | 楠岃瘉鍑瘉鏃犳晥鎴栧凡杩囨湡; 楠岃瘉鐮侀敊璇?閿欒淇℃伅鐢辨湇鍔＄杩斿洖) |
| 50000 | 绛惧彂浠ょ墝澶辫触 |

### 3.5 GET /api/v1/auth/me

鑾峰彇褰撳墠鐧诲綍鐢ㄦ埛淇℃伅銆傞渶鐢ㄦ埛 JWT銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user": { "...": "鍚?3.2 鐨?user 缁撴瀯" },
    "is_admin": false
  }
}
```

- `is_admin`: 鏄惁涓哄钩鍙拌秴绠¤鑹?鍘嗗彶瀛楁, 鏅€氱敤鎴锋亽涓?false)銆?
澶辫触璇存槑: 鏈櫥褰曟垨鍑瘉鏃犳晥杩斿洖 40100銆?
### 3.6 PUT /api/v1/auth/profile

鏇存柊鐢ㄦ埛鍚?QQ銆傞渶鐢ㄦ埛 JWT銆?
璇锋眰浣?

```json
{
  "nickname": "鏂扮敤鎴峰悕",
  "qq": "123456789"
}
```

瀛楁璇存槑:

- `nickname`(鍙€?: 鐢ㄦ埛鍚? 鏈€澶?64 瀛椼€?- `qq`(鍙€?: QQ 鍙风爜, 鏈€澶?32 瀛? 鍙负绌恒€?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user": { "...": "鏇存柊鍚庣殑 user 缁撴瀯" }
  }
}
```

澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒; 鐢ㄦ埛鍚嶈繃闀?鏈€澶?64 瀛?; QQ 鍙疯繃闀?|
| 40100 | 鏈櫥褰?|
| 50000 | 鏇存柊澶辫触 |

### 3.7 PUT /api/v1/auth/email

淇敼鐧诲綍閭(鍞竴鎬ф牎楠?銆傞渶鐢ㄦ埛 JWT銆?
璇锋眰浣?

```json
{
  "email": "new@example.com"
}
```

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user": { "...": "鏇存柊鍚庣殑 user 缁撴瀯" }
  }
}
```

澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒(email 缂哄け鎴栭潪娉? |
| 40100 | 鏈櫥褰?|
| 40900 | 閭宸茶娉ㄥ唽 |
| 50000 | 淇敼澶辫触 |

### 3.8 POST /api/v1/auth/change-password

淇敼褰撳墠鐢ㄦ埛瀵嗙爜銆傞渶鐢ㄦ埛 JWT銆?
璇锋眰浣?

```json
{
  "old_password": "old-password",
  "new_password": "new-password"
}
```

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒; 鏃у瘑鐮佷笉姝ｇ‘; 鍏朵粬淇敼澶辫触 |
| 40100 | 鏈櫥褰?|

### 3.9 POST /api/v1/auth/totp/setup

鐢熸垚 TOTP 瀵嗛挜涓庝簩缁寸爜(浠呮湭鍚敤 TOTP 鏃跺彲鐢?銆傞渶鐢ㄦ埛 JWT銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "secret": "JBSWY3DPEHPK3PXP",
    "otpauth_url": "otpauth://totp/AXmiPusher:user@example.com?secret=...",
    "qr_data_url": "data:image/png;base64,iVBORw0KGgo..."
  }
}
```

- `secret`: Base32 瀵嗛挜, 渚涚敤鎴锋墜鍔ㄥ綍鍏ラ獙璇佸櫒銆?- `otpauth_url`: 鏍囧噯 otpauth 閾炬帴銆?- `qr_data_url`: 浜岀淮鐮?PNG 鐨?data URL, 鍓嶇 `<img>` 鐩存帴灞曠ず, 鏃犻渶浜岀淮鐮佸簱銆?
澶辫触璇存槑: 鏈櫥褰?40100; 宸插惎鐢?TOTP 鎴栧弬鏁伴棶棰樿繑鍥?40000銆?
### 3.10 POST /api/v1/auth/totp/confirm

鐢ㄩ獙璇佺爜纭鍚敤涓ゆ楠岃瘉銆傞渶鐢ㄦ埛 JWT銆?
璇锋眰浣?

```json
{
  "code": "123456"
}
```

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "totp_enabled": true }
}
```

澶辫触璇存槑: 鏈櫥褰?40100; 楠岃瘉鐮侀敊璇垨鍏跺畠澶辫触杩斿洖 40000銆?
### 3.11 POST /api/v1/auth/totp/disable

鐢ㄥ綋鍓嶉獙璇佺爜鍏抽棴涓ゆ楠岃瘉銆傞渶鐢ㄦ埛 JWT銆?
璇锋眰浣?

```json
{
  "code": "123456"
}
```

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "totp_enabled": false }
}
```

澶辫触璇存槑: 鏈櫥褰?40100; 楠岃瘉鐮侀敊璇垨鍏跺畠澶辫触杩斿洖 40000銆?
---

## 4. 娑堟伅妯″潡(/messages)

鍏ㄩ儴璺敱鎸傚湪 `/api/v1/messages` 涓? 閴存潈鏂瑰紡涓?**RequireAuthOrAPIKey**: 鐢ㄦ埛 JWT 鎴栨湇鍔＄ API Key 鍧囧彲璁块棶銆?
### 4.1 POST /api/v1/messages

鍙戦€佹秷鎭彈鐞?鍗曞彂涓庢壒閲忓叡鐢?銆傝繑鍥?HTTP 202 琛ㄧず宸插彈鐞? 瀹為檯鍙戦€佸紓姝ヨ繘琛屻€?
璇锋眰浣?

```json
{
  "request_id": "optional-id-001",
  "title": "璁㈠崟閫氱煡",
  "content": "鎮ㄧ殑璁㈠崟宸插彂璐?,
  "channel": "auto",
  "template_code": "order_notify",
  "priority": "normal",
  "recipients": [
    { "target": "user@example.com", "params": { "name": "寮犱笁" } }
  ]
}
```

瀛楁璇存槑:

| 瀛楁 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `request_id` | 鍚?| 骞傜瓑閿? 鍚屼竴鐢ㄦ埛涓嬬浉鍚?request_id 鍙彈鐞嗕竴娆? 閲嶅鎻愪氦鐩存帴杩斿洖鍘?message_id |
| `title` | 鍚?| 娑堟伅鏍囬 |
| `content` | 鍚?涓?template_code 鑷冲皯涓€涓? | 娑堟伅姝ｆ枃 |
| `template_code` | 鍚?涓?content 鑷冲皯涓€涓? | 妯℃澘 code; 鎻愪緵鏃朵互妯℃澘鍐呭涓烘鏂? 骞剁敤姣忎釜鏀朵欢浜虹殑 `params` 娓叉煋 `{{var}}` 鍗犱綅绗?缂哄け鍙傛暟鏇挎崲涓虹┖涓?; title 涓虹┖鏃跺彇妯℃澘 name |
| `channel` | 鍚?| 娓犻亾: `webhook` / `email` / `apns` / `fcm` / `inapp` / `auto`銆傜己鐪佹垨 `auto` 鏃? 浼樺厛鍙栨ā鏉跨殑 `channel_type`, 鍚﹀垯榛樿 `webhook`; 鑻ヤ紭鍏堟笭閬撶啍鏂? 鑷姩鎸?webhook > email > apns > fcm > inapp 椤哄簭閫夌涓€涓彲鐢ㄦ笭閬撻檷绾?|
| `priority` | 鍚?| 浼樺厛绾?閫忎紶, 渚涢槦鍒楀弬鑰? |
| `recipients` | 鏄?| 鏀朵欢浜烘暟缁? 鑷冲皯 1 涓?|

`recipients` 鍏冪礌:

| 瀛楁 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `target` | 鏄?| 鏀朵欢浜虹洰鏍囥€俥mail 娓犻亾鍙暀绌?涓虹┖鏃朵娇鐢ㄦ笭閬撻厤缃殑榛樿鏀朵欢浜?; inapp 娓犻亾涓烘敹浠堕偖绠辨垨鐗规畩鍊?`all`(鍙戠粰璇ョ敤鎴疯嚜宸? 褰撳墠"绉熸埛"鍗冲崟涓€鐢ㄦ埛); 鍏朵綑娓犻亾涓哄悇娓犻亾瑕佹眰鐨勮澶囨爣璇?鍦板潃 |
| `params` | 鍚?| 妯℃澘娓叉煋鍙傛暟(浠呴厤鍚?template_code 浣跨敤) |

娓犻亾璇存槑:

- `webhook`: 鍙戦€佸埌鏈鎴锋敞鍐岀殑鍥炶皟鍦板潃, 鑷冲皯涓€涓闃呰繑鍥?2xx 鍗虫垚鍔熴€?- `email`: SMTP 鍙戦€? 闇€鍏堥厤缃偖浠舵笭閬? 鏀朵欢浜虹暀绌哄垯浣跨敤娓犻亾閰嶇疆鐨勯粯璁ゆ敹浠朵汉銆?- `apns`: Apple 鎺ㄩ€? 闇€閰嶇疆 Team ID/Key ID/Bundle ID/.p8; 400/403/404/410 瑙嗕负璁惧 token 澶辨晥涓嶉噸璇曘€?- `fcm`: Firebase 鎺ㄩ€? 闇€閰嶇疆鏈嶅姟璐﹀彿 JSON銆?- `inapp`: 绔欏唴淇? 鍐欏叆骞冲彴鏀朵欢绠? 鏀朵欢浜轰负閭鎴?`all`銆?- `auto`: 鎸夋ā鏉挎笭閬?榛樿 webhook 璺敱, 鐔旀柇鏃惰嚜鍔ㄩ檷绾с€?
鎴愬姛鍝嶅簲(HTTP 202):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "message_id": 1001,
    "duplicate": false,
    "count": 1
  }
}
```

- `message_id`: 鏈姹傚彈鐞嗕骇鐢熺殑棣栦釜娑堟伅 ID(骞傜瓑閲嶅璇锋眰鏃惰繑鍥炲師 message_id, 涓?`duplicate=true`)銆?- `duplicate`: 鏄惁涓哄箓绛夐噸澶嶈姹?鏈紶 request_id 鎭掍负 false)銆?- `count`: 瀹為檯鍏ラ槦鎴愬姛鐨勬秷鎭潯鏁?閫愭敹浠朵汉璁?銆?
澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒; template_code 涓?content 閮芥湭鎻愪緵; 妯℃澘涓嶅瓨鍦ㄦ垨涓嶅彲鐢? 鏀朵欢浜轰笉鑳戒负绌?|
| 40100 | 缂哄皯鎴栨棤鏁堝嚟璇?|
| 42900 | 璇锋眰棰戠巼瓒呴檺, 璇风◢鍚庡啀璇?|
| 50000 | 鍙戦€佸け璐?鍚笭閬撴湭娉ㄥ唽/鏈厤缃? |

### 4.2 GET /api/v1/messages

鍒嗛〉鏌ヨ娑堟伅鍒楄〃銆?
Query 鍙傛暟:

| 鍙傛暟 | 榛樿 | 璇存槑 |
|------|------|------|
| `current` | 1 | 椤电爜 |
| `pageSize` | 20 | 姣忛〉鏉℃暟 |
| `channel` | 绌?| 鎸夋笭閬撹繃婊?webhook/email/apns/fcm/inapp) |
| `status` | 绌?| 鎸夌姸鎬佽繃婊?瑙?4.3 鐘舵€佹満) |
| `recipient` | 绌?| 鎸夋敹浠朵汉杩囨护 |
| `since` | 绌?| 璧峰鏃堕棿(RFC3339) |
| `until` | 绌?| 缁撴潫鏃堕棿(RFC3339) |

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      {
        "message_id": 1001,
        "tenant_id": 1,
        "request_id": "optional-id-001",
        "channel": "webhook",
        "title": "璁㈠崟閫氱煡",
        "content": "鎮ㄧ殑璁㈠崟宸插彂璐?,
        "recipient": "user@example.com",
        "status": "SUCCESS",
        "error": "",
        "retry_count": 0,
        "cost_ms": 120,
        "created_at": "2026-08-27T10:00:00+08:00",
        "updated_at": "2026-08-27T10:00:00+08:00"
      }
    ],
    "total": 1,
    "success": true
  }
}
```

- `total`: 绗﹀悎鏉′欢鐨勬€绘潯鏁? 渚涘墠绔垎椤点€?
澶辫触璇存槑: 40100 鍑瘉鏃犳晥; 50000 鏌ヨ澶辫触銆?
### 4.3 GET /api/v1/messages/:id

鏌ヨ鍗曟潯娑堟伅璇︽儏銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200): `data` 涓哄崟鏉℃秷鎭璞? 瀛楁鍚?4.2 鍒楄〃椤?鍚?`error` 瀛楁, 璁板綍鏈€杩戜竴娆″け璐ュ師鍥?銆?
娑堟伅鐘舵€佹満:

```
PENDING 鈫?SENDING 鈫?SUCCESS
                    鈫?FAILED
                    鈫?RETRYING 鈫?(閲嶈瘯) 鈫?SENDING / DEAD
                    鈫?DEAD
                    鈫?CANCELLED
```

| 鐘舵€?| 鍚箟 |
|------|------|
| `PENDING` | 鎺掗槦涓? 绛夊緟娑堣垂鑰呰棰?|
| `SENDING` | 鍙戦€佷腑(宸茶棰? |
| `SUCCESS` | 閫佽揪鎴愬姛 |
| `FAILED` | 鍙戦€佸け璐?|
| `RETRYING` | 閲嶈瘯涓?鍚屾閲嶈瘯 3 娆℃垨绉熺害鍥炴敹澶嶄綅) |
| `DEAD` | 姝讳俊(璁ら瓒呴檺鎴栫啍鏂敊璇? 涓嶉噸璇? 浜哄伐澶勭悊) |
| `CANCELLED` | 宸插彇娑?|

澶辫触璇存槑: 40000 鏃犳晥鐨勬秷鎭?ID; 40100 鍑瘉鏃犳晥; 40400 娑堟伅涓嶅瓨鍦?鍚秺鏉冭闂粬绉熸埛娑堟伅)銆?
---

## 5. API Key 绠＄悊(/api-keys)

鍏ㄩ儴璺敱鎸傚湪 `/api/v1/api-keys` 涓? 闇€鐢ㄦ埛 JWT銆?
### 5.1 GET /api/v1/api-keys

鍒楀嚭褰撳墠鐢ㄦ埛鐨勫叏閮?API Key銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      {
        "id": 3,
        "tenant_id": 1,
        "name": "鐢熶骇鏈嶅姟鍣?,
        "key_prefix": "mp_ab12",
        "scopes": "[\"messages:send\",\"messages:read\"]",
        "status": "active",
        "expires_at": null,
        "last_used_at": "2026-08-26T10:00:00+08:00",
        "created_at": "2026-08-25T10:00:00+08:00"
      }
    ],
    "total": 1,
    "success": true
  }
}
```

- `key_prefix`: 鏄庢枃 key 鐨勫睍绀虹敤鍓嶇紑(瀹屾暣鏄庢枃涓嶅啀杩斿洖)銆?- `scopes`: 鏉冮檺 JSON 鏁扮粍瀛楃涓层€?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 5.2 POST /api/v1/api-keys

鍒涘缓 API Key銆傛槑鏂囦粎姝や竴娆¤繑鍥? 璇风珛鍗充繚瀛樸€?
璇锋眰浣?

```json
{
  "name": "鐢熶骇鏈嶅姟鍣?,
  "scopes": "[\"messages:send\",\"messages:read\"]",
  "expires_in": 365
}
```

瀛楁璇存槑:

| 瀛楁 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `name` | 鏄?| Key 鍚嶇О |
| `scopes` | 鍚?| 鏉冮檺鍒楄〃(JSON 鏁扮粍瀛楃涓? 褰撳墠鐢辨湇鍔＄瀛樺偍閫忎紶, 瀹為檯鏍￠獙浠ユ渶鏂颁唬鐮佷负鍑? |
| `expires_in` | 鍚?| 杩囨湡澶╂暟, 0 鎴栦笉浼犱负姘镐箙 |

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "key": {
      "id": 3,
      "tenant_id": 1,
      "name": "鐢熶骇鏈嶅姟鍣?,
      "key_prefix": "mp_ab12",
      "scopes": "[\"messages:send\",\"messages:read\"]",
      "status": "active",
      "expires_at": "2027-08-25T10:00:00+08:00",
      "last_used_at": null,
      "created_at": "2026-08-25T10:00:00+08:00"
    },
    "plain_key": "mp_ab12cdef..."
  }
}
```

- `plain_key`: 瀹屾暣鏄庢枃(浠?`mp_` 寮€澶?, **浠呭垱寤烘椂杩斿洖, 鏈嶅姟绔彧瀛樺搱甯? 涔嬪悗鏃犳硶鎵惧洖**銆?
澶辫触璇存槑: 40000 鍙傛暟閿欒; 40100 鏈櫥褰? 50000 鍒涘缓澶辫触銆?
### 5.3 DELETE /api/v1/api-keys/:id

鍚婇攢 API Key銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40000 鏃犳晥鐨?key ID; 40100 鏈櫥褰? 50000 鍚婇攢澶辫触銆?
---

## 6. 鍏煎 Key(/compat-keys)

Server閰?鍏煎鎺ュ叆鐢ㄣ€傚叏閮ㄨ矾鐢辨寕鍦?`/api/v1/compat-keys` 涓? 闇€鐢ㄦ埛 JWT銆?
### 6.1 GET /api/v1/compat-keys

鍒楀嚭鍏煎 key 鍒楄〃銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      {
        "id": 1,
        "tenant_id": 1,
        "external_key": "SCTabcdef0123456789...",
        "source": "serverchan_v1",
        "default_channel": "webhook",
        "description": "鑰佸鎴风鎺ュ叆",
        "status": "active",
        "last_used_at": null,
        "created_at": "2026-08-25T10:00:00+08:00",
        "updated_at": "2026-08-25T10:00:00+08:00"
      }
    ],
    "total": 1,
    "success": true
  }
}
```

- `source`: `serverchan_v1`(Server閰?, sc 褰㈡€?鎴?`serverchan_v2`(Server閰甭稵urbo鐗? sctapi 褰㈡€?銆?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 6.2 POST /api/v1/compat-keys

鍒涘缓鍏煎 key銆?
璇锋眰浣?

```json
{
  "source": "serverchan_v1",
  "external_key": "SCTcustomKey",
  "default_channel": "webhook",
  "description": "鐢熶骇鏈嶅姟鍣?
}
```

瀛楁璇存槑:

| 瀛楁 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `source` | 鏄?| `serverchan_v1` 鎴?`serverchan_v2`(鍏跺畠鍊?400) |
| `external_key` | 鍚?| 澶栭儴 key(鍗宠€佸鎴风浣跨敤鐨?SCKEY/SendKey), 鐣欑┖鑷姩鐢熸垚 32 浣嶉殢鏈轰覆 |
| `default_channel` | 鍚?| 璇?key 娑堟伅鐨勯粯璁ゆ笭閬? 缂虹渷 `webhook` |
| `description` | 鍚?| 澶囨敞 |

鎴愬姛鍝嶅簲(HTTP 200): `data` 涓哄垱寤虹殑 CompatKey 瀵硅薄, 瀛楁鍚?6.1 鍒楄〃椤?鍚?`external_key`)銆?
澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒; source 涓嶆槸 serverchan_v1/serverchan_v2 |
| 40100 | 鏈櫥褰?|
| 40900 | 鍒涘缓澶辫触, key 鍙兘宸插瓨鍦?|

### 6.3 DELETE /api/v1/compat-keys/:id

鍒犻櫎鍏煎 key銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40000 鏃犳晥鐨?key ID; 40100 鏈櫥褰? 50000 鍒犻櫎澶辫触銆?
### 6.4 鍏煎璋冪敤鍦板潃(鍏紑, 鏃犻渶鐧诲綍)

鑰佸鎴风闆舵敼鍔ㄦ帴鍏? 璇﹁绗?14 绔?

- `GET /api/sc/{SCKEY}.send`(Server閰? 褰㈡€?
- `POST /api/sctapi/{SendKey}.send`(Server閰甭稵urbo鐗?褰㈡€?

---

## 7. 鍥炶皟璁㈤槄(/callbacks)

Webhook 娑堟伅鍙戦€佸悗, 骞冲彴浼?POST 鍒版敞鍐岀殑鍥炶皟鍦板潃銆傚叏閮ㄨ矾鐢辨寕鍦?`/api/v1/callbacks` 涓? 闇€鐢ㄦ埛 JWT銆?
### 7.1 GET /api/v1/callbacks

鍒楀嚭鍥炶皟璁㈤槄銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      {
        "id": 2,
        "tenant_id": 1,
        "url": "https://example.com/hook",
        "events": "success,failed",
        "status": "active",
        "created_at": "2026-08-25T10:00:00+08:00",
        "updated_at": "2026-08-25T10:00:00+08:00"
      }
    ],
    "total": 1,
    "success": true
  }
}
```

- `events`: 璁㈤槄鐨勪簨浠? 閫楀彿鍒嗛殧鐨勫瓧绗︿覆(濡?`success,failed`)銆俙secret` 涓烘晱鎰熷瓧娈? 涓嶅洖浼犮€?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 7.2 POST /api/v1/callbacks

娉ㄥ唽鍥炶皟璁㈤槄銆?
璇锋眰浣?

```json
{
  "url": "https://example.com/hook",
  "secret": "my-secret",
  "events": ["success", "failed"]
}
```

瀛楁璇存槑:

| 瀛楁 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `url` | 鏄?| 鍥炶皟鍦板潃, 蹇呴』鏄悎娉?URL |
| `secret` | 鍚?| 绛惧悕瀵嗛挜, 鐢ㄤ簬鏍￠獙鍥炶皟鏉ユ簮 |
| `events` | 鍚?| 璁㈤槄鐨勪簨浠舵暟缁? 缂虹渷 `["success","failed"]` |

鎴愬姛鍝嶅簲(HTTP 200): `data` 涓哄垱寤虹殑璁㈤槄瀵硅薄(瀛楁鍚?7.1, 娉ㄦ剰 `events` 鍦ㄥ搷搴斾腑涓洪€楀彿鍒嗛殧瀛楃涓? `secret` 涓嶅洖浼?銆?
澶辫触璇存槑: 40000 鍙傛暟閿欒(url 闈炴硶); 40100 鏈櫥褰? 50000 鍒涘缓澶辫触銆?
### 7.3 DELETE /api/v1/callbacks/:id

鍒犻櫎鍥炶皟璁㈤槄銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40000 鏃犳晥鐨?ID; 40100 鏈櫥褰? 50000 鍒犻櫎澶辫触銆?
### 7.4 鍥炶皟绛惧悕璇存槑

Webhook 鍙戦€佹椂鐨勮姹備笌绛惧悕(浠ヤ唬鐮佷负鍑?:

- 鏂规硶: `POST`, 璇锋眰澶?`Content-Type: application/json`, `User-Agent: AXmiPusher/1.0`銆?- 杞借嵎:

```json
{
  "message_id": 1001,
  "title": "璁㈠崟閫氱煡",
  "content": "鎮ㄧ殑璁㈠崟宸插彂璐?,
  "recipient": "user@example.com",
  "timestamp": 1785200000
}
```

- 绛惧悕: 璁㈤槄璁剧疆浜?`secret` 鏃? 璇锋眰澶存惡甯?`X-MP-Signature`, 鍊间负瀵?*鍘熷璇锋眰浣?JSON 瀛楃涓?**鍋?HMAC-SHA256 鍚庤浆灏忓啓 hex 鐨勭粨鏋? 瀵嗛挜涓鸿闃呯殑 `secret`銆備笟鍔℃柟鍙敤 `HMAC-SHA256(secret, body)` 楠岀闃蹭吉閫?

```text
X-MP-Signature = hex( HMAC-SHA256( secret, body ) )
```

- 缁撴灉鍒ゅ畾: 杩斿洖 2xx 瑙嗕负閫佽揪鎴愬姛; 浠讳竴璁㈤槄鎴愬姛鍗宠娓犻亾鎴愬姛(閫愪釜灏濊瘯, 闈炲叏閮ㄨ姹?2xx)銆?
---

## 8. 妯℃澘(/templates)

娑堟伅妯℃澘, 鍒涘缓鍗崇敓鏁?鏃犲鏍?銆傚叏閮ㄨ矾鐢辨寕鍦?`/api/v1/templates` 涓? 闇€鐢ㄦ埛 JWT銆?
### 8.1 GET /api/v1/templates

鍒嗛〉鏌ヨ妯℃澘鍒楄〃銆?
Query 鍙傛暟: `current`(榛樿 1)銆乣pageSize`(榛樿 20)銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      {
        "id": 5,
        "tenant_id": 1,
        "code": "order_notify",
        "name": "璁㈠崟閫氱煡",
        "content": "鎮ㄥソ {{name}}, 鎮ㄧ殑璁㈠崟宸插彂璐?,
        "channel_type": "webhook",
        "status": "active",
        "created_by": 1,
        "created_at": "2026-08-25T10:00:00+08:00",
        "updated_at": "2026-08-25T10:00:00+08:00"
      }
    ],
    "total": 1,
    "success": true
  }
}
```

澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 8.2 POST /api/v1/templates

鍒涘缓妯℃澘(鍒涘缓鍗崇敓鏁? status 鎭掍负 active)銆?
璇锋眰浣?

```json
{
  "code": "order_notify",
  "name": "璁㈠崟閫氱煡",
  "content": "鎮ㄥソ {{name}}, 鎮ㄧ殑璁㈠崟宸插彂璐?,
  "channel_type": "webhook"
}
```

瀛楁璇存槑:

| 瀛楁 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `code` | 鏄?| 妯℃澘 code(鍙戦€佹椂鎸?code 寮曠敤; 绉熸埛鍐呭敮涓€) |
| `name` | 鏄?| 妯℃澘鍚嶇О |
| `content` | 鏄?| 妯℃澘鍐呭, 鏀寔 `{{var}}` 鍗犱綅绗?|
| `channel_type` | 鍚?| 榛樿娓犻亾, 缂虹渷 `webhook` |

鎴愬姛鍝嶅簲(HTTP 200): `data` 涓哄垱寤虹殑妯℃澘瀵硅薄(瀛楁鍚?8.1)銆?
澶辫触璇存槑: 40000 鍙傛暟閿欒鎴栧垱寤哄け璐?濡?code 閲嶅); 40100 鏈櫥褰曘€?
### 8.3 GET /api/v1/templates/:id

妯℃澘璇︽儏銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200): `data` 涓烘ā鏉垮璞?瀛楁鍚?8.1)銆?
澶辫触璇存槑: 40000 鏃犳晥鐨勬ā鏉?ID; 40100 鏈櫥褰? 40400 妯℃澘涓嶅瓨鍦ㄣ€?
### 8.4 PUT /api/v1/templates/:id

鏇存柊妯℃澘(鏇存柊鍚庣洿鎺ョ敓鏁? 鏃犲鏍?銆?
璇锋眰浣? 鍚?8.2(鍥涗釜瀛楁)銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40000 鍙傛暟閿欒鎴栨洿鏂板け璐? 40100 鏈櫥褰曘€?
### 8.5 DELETE /api/v1/templates/:id

鍒犻櫎妯℃澘銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40000 鏃犳晥鐨勬ā鏉?ID; 40100 鏈櫥褰? 50000 鍒犻櫎澶辫触銆?
---

## 9. 娓犻亾閰嶇疆(/channels)

娓犻亾涓哄钩鍙板唴缃殑 5 绫诲彂閫侀€氶亾銆傚叏閮ㄨ矾鐢辨寕鍦?`/api/v1/channels` 涓? 闇€鐢ㄦ埛 JWT銆?
### 9.1 GET /api/v1/channels

鍒楀嚭鍏ㄩ儴娓犻亾鍙婂綋鍓嶇敤鎴风殑閰嶇疆鐘舵€併€?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      { "type": "webhook", "name": "Webhook 鍥炶皟", "desc": "POST 鍒颁笟鍔℃柟鍥炶皟鍦板潃", "configured": true },
      { "type": "email", "name": "閭欢", "desc": "SMTP 鍙戦€?闇€鍦ㄦ笭閬撻厤缃腑璁剧疆 SMTP 涓庨粯璁ゆ敹浠朵汉)", "configured": false },
      { "type": "apns", "name": "APNs (iOS)", "desc": "Apple 鎺ㄩ€? 闇€ Team ID/Key ID/Bundle ID/.p8", "configured": false },
      { "type": "fcm", "name": "FCM (Android)", "desc": "Firebase 鎺ㄩ€? 闇€鏈嶅姟璐﹀彿 JSON", "configured": false },
      { "type": "inapp", "name": "绔欏唴淇?, "desc": "骞冲彴鍐呮敹浠剁, 鏃犻渶澶栭儴閰嶇疆", "configured": true }
    ],
    "total": 5,
    "success": true
  }
}
```

瀛楁璇存槑:

- `type`: 娓犻亾绫诲瀷, 鏋氫妇 `webhook` / `email` / `apns` / `fcm` / `inapp`銆?- `configured`: 鏄惁鏈夊彲鐢ㄩ厤缃€俙webhook`/`inapp` 鎭掍负 `true`(鏃犻渶澶栭儴閰嶇疆); `email`/`apns`/`fcm` 闇€鍦?9.3 涓厤缃€?- 鑻ュ綋鍓嶇敤鎴峰凡涓烘煇娓犻亾淇濆瓨杩囬厤缃? 璇ヨ棰濆杩斿洖 `id`(娓犻亾閰嶇疆璁板綍 ID)銆乣status`銆乣updated_at`銆?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 9.2 GET /api/v1/channels/health

娓犻亾鍋ュ悍鐪嬫澘: 鐔旀柇鐘舵€?+ 24 灏忔椂閫佽揪缁熻銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      {
        "type": "webhook",
        "name": "Webhook 鍥炶皟",
        "breaker_state": "closed",
        "breaker_failures": 0,
        "msg_24h": 12,
        "success_24h": 11,
        "success_rate": 91.7,
        "last_success_at": "2026-08-27T09:30:00+08:00",
        "last_failure_at": null
      }
    ],
    "total": 5,
    "success": true
  }
}
```

瀛楁璇存槑:

- `breaker_state`: 鐔旀柇鐘舵€? `closed`(闂悎)/ `open`(鐔旀柇)/ `half_open`(鍗婂紑, 鍐峰嵈鍚庢斁琛屼竴涓帰閽?銆?- `breaker_failures`: 绱澶辫触娆℃暟銆?- `msg_24h` / `success_24h` / `success_rate`: 杩?24 灏忔椂娑堟伅鎬婚噺銆佹垚鍔熼噺銆佹垚鍔熺巼(鐧惧垎姣? 淇濈暀 1 浣嶅皬鏁?銆?- `last_success_at` / `last_failure_at`: 鏈€杩戞垚鍔?澶辫触鏃堕棿, 鍙负 null銆?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 缁熻澶辫触銆?
### 9.3 PUT /api/v1/channels/:type

淇濆瓨鏌愭笭閬撶殑绉熸埛閰嶇疆(`type` 浠呮敮鎸?`email` / `apns` / `fcm`; `webhook`/`inapp` 鏃犻渶閰嶇疆, 浼犲叾瀹冨€?400)銆?
璇锋眰浣? `config` 涓烘笭閬撶鏈夐厤缃殑 JSON 瀵硅薄銆?
email 娓犻亾:

```json
{
  "config": {
    "host": "smtp.example.com",
    "port": 465,
    "user": "sender@example.com",
    "password": "smtp-password",
    "from": "sender@example.com",
    "recipient": "default-receiver@example.com"
  }
}
```

- `port`: 465 闅愬紡 TLS; 587/25 璧?STARTTLS(鑻ユ敮鎸?銆傜鍙ｅ吋瀹规暟瀛楁垨瀛楃涓层€?- `recipient`: 榛樿鏀朵欢浜? 鍙戦€佹椂鏈寚瀹氭敹浠朵汉鏃跺厹搴曚娇鐢ㄣ€?
apns 娓犻亾:

```json
{
  "config": {
    "team_id": "TEAMID12345",
    "key_id": "KEYID12345",
    "bundle_id": "com.example.app",
    "key_p8": "-----BEGIN PRIVATE KEY-----...",
    "sandbox": true
  }
}
```

- `sandbox`: `true` = 寮€鍙戠幆澧?娌欑洅), `false` = 鐢熶骇鐜銆?
fcm 娓犻亾:

```json
{
  "config": {
    "project_id": "my-project",
    "client_email": "firebase-adminsdk@my-project.iam.gserviceaccount.com",
    "private_key": "-----BEGIN PRIVATE KEY-----..."
  }
}
```

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true, "type": "email" }
}
```

澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 涓嶆敮鎸佺殑娓犻亾绫诲瀷; 鍙傛暟閿欒; 閰嶇疆蹇呴』鏄悎娉?JSON |
| 40100 | 鏈櫥褰?|
| 50000 | 淇濆瓨澶辫触 |

### 9.4 DELETE /api/v1/channels/:type

鍒犻櫎绉熸埛娓犻亾閰嶇疆(鍥炲埌骞冲彴榛樿)銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40000 鏃犳晥鐨勭被鍨? 40100 鏈櫥褰? 40400 璇ユ笭閬撴病鏈夌鎴烽厤缃€?
---

## 10. 濂楅/璁㈤槄/鏀粯(/pay)

鍏ㄩ儴璺敱鎸傚湪 `/api/v1/pay` 涓? 闇€鐢ㄦ埛 JWT(浠?`POST /api/v1/pay/notify` 涓?`GET /api/v1/pay/return` 涓哄叕寮€鏀粯鍥炶皟, 瑙?10.5)銆?
### 10.1 GET /api/v1/pay/plans

鍙敤濂楅鍒楄〃銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      {
        "id": 1,
        "name": "鍏嶈垂鐗?,
        "price": 0,
        "duration_days": 30,
        "quota": "{\"monthly_messages\":1000,\"channels\":[\"webhook\",\"inapp\"]}",
        "description": "浣撻獙濂楅",
        "status": "active",
        "sort_order": 0,
        "created_at": "2026-08-01T10:00:00+08:00",
        "updated_at": "2026-08-01T10:00:00+08:00"
      }
    ],
    "total": 1,
    "success": true
  }
}
```

- `quota`: 閰嶉 JSON 瀛楃涓? 绾﹀畾鍚?`monthly_messages`(鏈堝害娑堟伅涓婇檺)涓?`channels`(鍙敤娓犻亾鍒楄〃)绛夐敭, 鍏蜂綋閿互濂楅瀹氫箟涓哄噯銆?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 10.2 GET /api/v1/pay/subscription

褰撳墠璁㈤槄涓庡搴斿椁愩€?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "subscription": {
      "id": 2,
      "tenant_id": 1,
      "plan_id": 1,
      "start_at": "2026-08-01T10:00:00+08:00",
      "end_at": "2026-08-31T10:00:00+08:00",
      "status": "active",
      "created_at": "2026-08-01T10:00:00+08:00",
      "updated_at": "2026-08-01T10:00:00+08:00"
    },
    "plan": { "...": "鍚?10.1 鐨勫椁愮粨鏋? }
  }
}
```

- `subscription.status`: `active`(鐢熸晥涓?/ `expired`(宸茶繃鏈?銆?- 鏃犺闃呮椂杩斿洖 `{"subscription": null, "plan": null}`銆?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 10.3 POST /api/v1/pay/orders

鍒涘缓鏀粯璁㈠崟銆?
璇锋眰浣?

```json
{
  "plan_id": 2,
  "type": "alipay"
}
```

瀛楁璇存槑:

- `plan_id`(蹇呭～): 濂楅 ID(椤讳负鐢熸晥濂楅)銆?- `type`(鍙€?: 鏀粯鏂瑰紡, `alipay`(鏀粯瀹? 榛樿)/ `wxpay`(寰俊)銆?
鎴愬姛鍝嶅簲(HTTP 200), 浠樿垂濂楅:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "order": {
      "id": 10,
      "tenant_id": 1,
      "plan_id": 2,
      "type": "alipay",
      "out_trade_no": "MP20260827100000000001",
      "epay_trade_no": "",
      "amount": 9.9,
      "status": "pending",
      "paid_at": null,
      "expired_at": "2026-08-27T10:30:00+08:00",
      "created_at": "2026-08-27T10:00:00+08:00",
      "updated_at": "2026-08-27T10:00:00+08:00"
    },
    "pay_url": "https://epay.example.com/submit.php?pid=..."
  }
}
```

- `order.status`: `pending`(寰呮敮浠?/ `paid`(宸叉敮浠?/ `closed`(宸插叧闂?銆?- `pay_url`: 鏄撴敮浠樿烦杞摼鎺? 娴忚鍣ㄦ墦寮€瀹屾垚鏀粯銆?
鎴愬姛鍝嶅簲(HTTP 200), 鍏嶈垂濂楅(price=0): 涓嶄緷璧栨槗鏀粯, 鐩存帴婵€娲昏闃? 杩斿洖"宸叉敮浠?璁㈠崟涓旀棤 `pay_url`:

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "order": {
      "plan_id": 1,
      "amount": 0,
      "status": "paid",
      "type": "free"
    },
    "pay_url": ""
  }
}
```

澶辫触璇存槑:

| 涓氬姟鐮?| 鍦烘櫙 |
|--------|------|
| 40000 | 鍙傛暟閿欒; 鏄撴敮浠樺皻鏈厤缃? 濂楅涓嶅瓨鍦ㄦ垨宸蹭笅鏋?|
| 40100 | 鏈櫥褰?|
| 50000 | 鍒涘缓璁㈠崟澶辫触 |

### 10.4 GET /api/v1/pay/orders/:id

鏌ヨ璁㈠崟鐘舵€?鍓嶇杞鏀粯缁撴灉)銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200): `data` 涓鸿鍗曞璞? 瀛楁鍚?10.3(`status` 涓?`paid` 琛ㄧず鍒拌处, 濂楅宸茬敓鏁?銆?
澶辫触璇存槑: 40000 鏃犳晥鐨勮鍗?ID; 40100 鏈櫥褰? 40400 璁㈠崟涓嶅瓨鍦ㄣ€?
### 10.5 鏀粯鍥炶皟(鍏紑)

- `POST /api/v1/pay/notify`: 鏄撴敮浠樻湇鍔＄寮傛閫氱煡, 骞冲彴楠岀鍚庢縺娲昏闃呫€傝繑鍥炵函鏂囨湰 `success`(鍋滄閲嶈瘯)鎴?`fail`(涓婃父浼氶噸璇?, **涓嶉伒寰粺涓€鍝嶅簲鏍煎紡**銆?- `GET /api/v1/pay/return?out_trade_no=...`: 鏀粯瀹屾垚鍚庣殑娴忚鍣ㄥ洖璺抽〉, 杩斿洖 HTML 椤甸潰銆?
---

## 11. 鎵归噺浠诲姟(/batch-tasks)

澶у悕鍗曞垎鎵瑰彂閫併€傚叏閮ㄨ矾鐢辨寕鍦?`/api/v1/batch-tasks` 涓? 闇€鐢ㄦ埛 JWT銆?
### 11.1 POST /api/v1/batch-tasks

鍒涘缓鎵归噺浠诲姟骞跺紓姝ュ惎鍔?鍚庡彴姣?100 涓敹浠朵汉涓€鐗囧垎鎵瑰叆闃?銆?
璇锋眰浣?

```json
{
  "name": "鏈堝害钀ラ攢",
  "template_code": "promo",
  "title": "淇冮攢娲诲姩",
  "content": "鍏ㄥ満 5 鎶?,
  "channel": "webhook",
  "priority": "normal",
  "recipients": [
    { "target": "a@example.com", "params": { "name": "寮犱笁" } },
    { "target": "b@example.com" }
  ]
}
```

瀛楁璇存槑:

| 瀛楁 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `name` | 鏄?| 浠诲姟鍚嶇О |
| `template_code` | 鍚?| 妯℃澘 code(涓?content 鑷冲皯涓€涓? |
| `title` | 鍚?| 鏍囬 |
| `content` | 鍚?| 姝ｆ枃(涓?template_code 鑷冲皯涓€涓? |
| `channel` | 鍚?| 娓犻亾(鍚?4.1) |
| `priority` | 鍚?| 浼樺厛绾?|
| `recipients` | 鏄?| 鏀朵欢浜烘暟缁? 鑷冲皯 1 涓?鍏冪礌缁撴瀯鍚?4.1) |

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "id": 7,
    "tenant_id": 1,
    "name": "鏈堝害钀ラ攢",
    "status": "running",
    "total": 2,
    "success": 0,
    "failed": 0,
    "config": "{\"template_code\":\"promo\",\"title\":\"淇冮攢娲诲姩\",\"content\":\"鍏ㄥ満 5 鎶榎",\"channel\":\"webhook\",\"priority\":\"normal\",\"recipients\":[...]}",
    "error": "",
    "created_at": "2026-08-27T10:00:00+08:00",
    "updated_at": "2026-08-27T10:00:00+08:00"
  }
}
```

- `status`: `pending` / `running` / `done` / `failed` / `cancelled`銆?- `config`: 浠诲姟鍙傛暟 JSON 瀛楃涓层€?- 鍒涘缓鍚庝换鍔″湪鍚庡彴寮傛鎵ц, 鐢?11.2 / 11.3 杞杩涘害(`success` / `failed` 瀹炴椂鏇存柊)銆?
澶辫触璇存槑: 40000 鍙傛暟閿欒鎴栧垱寤哄け璐?濡傛敹浠朵汉涓嶈兘涓虹┖銆乼emplate_code 涓?content 閮芥湭鎻愪緵); 40100 鏈櫥褰曘€?
### 11.2 GET /api/v1/batch-tasks

鍒嗛〉鏌ヨ浠诲姟鍒楄〃銆?
Query 鍙傛暟: `current`(榛樿 1)銆乣pageSize`(榛樿 20)銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [ { "...": "浠诲姟瀵硅薄, 鍚?11.1" } ],
    "total": 1,
    "success": true
  }
}
```

澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 11.3 GET /api/v1/batch-tasks/:id

浠诲姟璇︽儏銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200): `data` 涓轰换鍔″璞?鍚?11.1, 鍚疄鏃?`success` / `failed` 杩涘害)銆?
澶辫触璇存槑: 40000 鏃犳晥鐨勪换鍔?ID; 40100 鏈櫥褰? 40400 浠诲姟涓嶅瓨鍦ㄣ€?
### 11.4 POST /api/v1/batch-tasks/:id/cancel

鍙栨秷浠诲姟(宸插彂閫侀儴鍒嗕繚鐣?銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40000 鍙栨秷澶辫触(濡備换鍔″凡缁撴潫鎴栨鍦ㄥ彇娑?; 40100 鏈櫥褰曘€?
---

## 12. 绔欏唴淇?/inbox)

骞冲彴鍐呮敹浠剁(娓犻亾 `inapp` 鐨勬秷鎭惤鍦ㄨ繖閲?銆傚叏閮ㄨ矾鐢辨寕鍦?`/api/v1/inbox` 涓? 闇€鐢ㄦ埛 JWT銆?
### 12.1 GET /api/v1/inbox

褰撳墠鐢ㄦ埛鐨勬敹浠剁鍒楄〃銆?
Query 鍙傛暟: `current`(榛樿 1)銆乣pageSize`(榛樿 20, 涓婇檺 100)銆乣read`(鍙€? `true` 鍙湅宸茶, `false` 鍙湅鏈)銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "data": [
      {
        "id": 12,
        "tenant_id": 1,
        "user_id": 1,
        "user_email": "user@example.com",
        "title": "璁㈠崟閫氱煡",
        "content": "鎮ㄧ殑璁㈠崟宸插彂璐?,
        "is_read": false,
        "read_at": null,
        "created_at": "2026-08-27T10:00:00+08:00"
      }
    ],
    "total": 3,
    "success": true
  }
}
```

- `is_read`: 鏄惁宸茶; `read_at`: 宸茶鏃堕棿(null 琛ㄧず鏈)銆?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 12.2 GET /api/v1/inbox/unread-count

鏈绔欏唴淇℃暟閲忋€?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "unread": 3 }
}
```

澶辫触璇存槑: 40100 鏈櫥褰? 50000 鏌ヨ澶辫触銆?
### 12.3 PUT /api/v1/inbox/:id/read

鏍囪鍗曟潯宸茶銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40000 鏃犳晥鐨?ID; 40100 鏈櫥褰? 40400 娑堟伅涓嶅瓨鍦?鍚秺鏉冭闂粬浜烘秷鎭?銆?
### 12.4 PUT /api/v1/inbox/read-all

鍏ㄩ儴鏍囪宸茶銆?
鏃犺姹備綋銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": { "ok": true }
}
```

澶辫触璇存槑: 40100 鏈櫥褰曘€?
---

## 13. 缁熻(/stats)

鍏ㄩ儴璺敱鎸傚湪 `/api/v1/stats` 涓? 闇€鐢ㄦ埛 JWT銆?
### 13.1 GET /api/v1/stats/messages

鎸夌姸鎬佺粺璁℃秷鎭暟閲忋€?
Query 鍙傛暟:

| 鍙傛暟 | 榛樿 | 璇存槑 |
|------|------|------|
| `since` | 褰撳墠鏃堕棿鍓?24 灏忔椂 | 璧峰鏃堕棿(RFC3339) |
| `until` | 褰撳墠鏃堕棿 | 缁撴潫鏃堕棿(RFC3339) |

鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "since": "2026-08-26T10:00:00+08:00",
    "until": "2026-08-27T10:00:00+08:00",
    "status": {
      "PENDING": 0,
      "SENDING": 0,
      "SUCCESS": 11,
      "FAILED": 1,
      "RETRYING": 0,
      "DEAD": 0,
      "CANCELLED": 0
    }
  }
}
```

- `status`: 鍚勬秷鎭姸鎬佺殑璁℃暟(閿负 4.3 鐘舵€佹満涓殑澶у啓鐘舵€? 璁℃暟涓?0 鐨勯敭涔熷彲鑳藉瓨鍦?銆?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 缁熻澶辫触銆?
### 13.2 GET /api/v1/stats/overview

姒傝缁熻: 鍙戦€佹€婚噺銆佹垚鍔?澶辫触鏁般€佹垚鍔熺巼銆?
Query 鍙傛暟: `since`(榛樿褰撳墠鏃堕棿鍓?24 灏忔椂, RFC3339)銆?
鎴愬姛鍝嶅簲(HTTP 200):

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "total": 12,
    "success": 11,
    "failed": 1,
    "success_rate": 91.7,
    "period": {
      "since": "2026-08-26T10:00:00+08:00",
      "until": "2026-08-27T10:00:00+08:00"
    }
  }
}
```

- `failed`: 缁熻鍙ｅ緞涓?`FAILED` 涓?`DEAD` 涔嬪拰銆?- `success_rate`: 鐧惧垎姣? 淇濈暀 1 浣嶅皬鏁般€?
澶辫触璇存槑: 40100 鏈櫥褰? 50000 缁熻澶辫触銆?
---

## 14. Server閰?鍏煎灞?鍏紑)

鑰?Server閰?瀹㈡埛绔浂鏀瑰姩鎺ュ叆銆傝矾鐢变负鍏紑绔偣(鏃犲钩鍙伴壌鏉?, 鐢卞吋瀹?key 鏄犲皠鍒板搴旂敤鎴枫€?
### 14.1 Server閰? 褰㈡€?v1)

```
GET /api/sc/{SCKEY}.send
```

- `SCKEY` 涓哄吋瀹?key 鐨?`external_key`(鍒涘缓鏃舵寚瀹氭垨鑷姩鐢熸垚)銆?- 鍙傛暟鏀寔琛ㄥ崟鎴?query:

| 鍙傛暟 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `text` | 鏄?| 娑堟伅鏍囬 |
| `desp` | 鍚?| 娑堟伅鍐呭(鍙负绌? |

- 鏍囬涓庡唴瀹归兘涓虹┖鏃跺け璐ャ€?- 鎴愬姛鍝嶅簲(HTTP 200, Server閰卞師鐗堟牸寮?:

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "pushid": 1001
  }
}
```

- 澶辫触鍝嶅簲(HTTP 200, 鍘熺増鏍煎紡, `pushid` 涓哄钩鍙版秷鎭?ID):

```json
{
  "errno": 40001,
  "errmsg": "invalid key"
}
```

### 14.2 Server閰甭稵urbo鐗?褰㈡€?v2)

```
POST /api/sctapi/{SendKey}.send
```

- `SendKey` 涓哄吋瀹?key 鐨?`external_key`銆?- 鍙傛暟涓?`application/x-www-form-urlencoded`:

| 鍙傛暟 | 蹇呭～ | 璇存槑 |
|------|------|------|
| `title` | 鍚?| 娑堟伅鏍囬(涓虹┖鏃跺洖閫€浣跨敤 `short`) |
| `desp` | 鍚?| 娑堟伅鍐呭(鍙负绌? |
| `short` | 鍚?| 鐭爣棰? 浠呭綋 `title` 涓虹┖鏃朵綔涓烘爣棰?|

- 鏍囬涓庡唴瀹归兘涓虹┖鏃跺け璐ャ€?- 鎴愬姛鍝嶅簲(HTTP 200, 鍘熺増鏍煎紡):

```json
{
  "code": 0,
  "message": "",
  "data": {
    "pushid": 1001,
    "readkey": ""
  }
}
```

- 澶辫触鍝嶅簲(HTTP 200, 鍘熺増鏍煎紡):

```json
{
  "code": 40001,
  "message": "invalid key"
}
```

### 14.3 璇存槑

- 鍏煎灞傛秷鎭粯璁よ蛋璇ュ吋瀹?key 鐨?`default_channel`(鍒涘缓鏃惰缃? 榛樿 `webhook`), 杩涘叆骞冲彴鏍稿績鍙戦€侀摼璺? 鐘舵€佸彲鍦ㄧ敤鎴蜂腑蹇冩秷鎭垪琛ㄤ腑鏌ョ湅銆?- 鎴愬姛涓庡け璐ュ潎杩斿洖 HTTP 200, 鍝嶅簲浣撲弗鏍肩収鎶勫師鐗? 瀹㈡埛绔彲鏃犳劅鍒囨崲銆?
---

## 15. 闄勫綍

### 15.1 娑堟伅鐘舵€佹満

```
PENDING 鈫?SENDING 鈫?SUCCESS
                    鈫?FAILED
                    鈫?RETRYING 鈫?(閲嶈瘯鍚? SENDING / DEAD
                    鈫?DEAD
                    鈫?CANCELLED
```

- `PENDING` 鎺掗槦涓?鈫?娑堣垂鑰呰棰嗗彉 `SENDING`(璁ら鍗冲埛鏂扮绾? 閲嶅惎涓嶄涪娑堟伅)銆?- 澶辫触鍙噸璇? 鍚屾閲嶈瘯 3 娆? 瓒呮椂鏈畬鎴愮敱绉熺害鍥炴敹(ReapStale)澶嶄綅閲嶈瘯, 杈炬渶澶у皾璇曟鏁?鐢遍槦鍒楅厤缃?`MP_QUEUE_MAX_CLAIM_ATTEMPTS` 鎺у埗)缃?`DEAD`銆?- **鐔旀柇閿欒涓嶉噸璇曠洿鎺?`DEAD`**: 娓犻亾鐔旀柇鏃堕噸璇曟棤鎰忎箟銆?- 娑堟伅鐢熷懡鍛ㄦ湡浜嬩欢(created/sending/success/failed/retry/dead)鍏ㄩ儴钀藉簱, 渚涘璁′笌缁熻銆?
### 15.2 鐔旀柇

- 鎸?褰掑睘鐢ㄦ埛, 娓犻亾)缁村害闅旂銆?- 杩炵画澶辫触 3 娆?鈫?鐔旀柇(`open`), 鍐峰嵈 30 绉?鈫?鍗婂紑(`half_open`)鏀捐涓€涓帰閽堣姹? 鎺㈤拡鎴愬姛鍥炲埌闂悎(`closed`), 澶辫触绔嬪嵆閲嶆柊鐔旀柇銆?- `auto` 娓犻亾璺敱鏃朵細璺宠繃宸茬啍鏂笭閬? 鑷姩闄嶇骇(瑙?4.1)銆?- 鐔旀柇鐘舵€佸彲閫氳繃 `GET /api/v1/channels/health` 鏌ョ湅銆?
### 15.3 闄愭祦

- 鍙戦€佸彈鐞嗕晶鎸夊綊灞炵敤鎴?绉熸埛)闄愭祦, 鍥哄畾绐楀彛 1 鍒嗛挓(Redis 瀹炵幇; 鏈厤缃?Redis 鏃堕檷绾у唴瀛樹护鐗屾《)銆?- 瓒呭嚭闄愬埗杩斿洖 42900銆傞檺娴侀槇鍊?姣忓垎閽熸潯鏁?鐢辩鐞嗗悗鍙扮郴缁熻缃皟鏁淬€?
### 15.4 骞傜瓑

- 鏈哄埗: 鍚岀敤鎴?+ 鐩稿悓 `request_id` 鍙彈鐞嗕竴娆°€?- DB 鍞竴绱㈠紩(tenant_id + request_id)鍏滃簳, Redis 缂撳瓨鍔犻€?鍛戒腑鐩存帴杩斿洖鍘?`message_id`, TTL 24 灏忔椂)銆?- 閲嶅璇锋眰鍝嶅簲涓殑 `duplicate=true`, `message_id` 涓洪娆″彈鐞嗙殑 message_id銆?- 鍙湁鎴愬姛鍏ラ槦鑷冲皯涓€鏉℃墠鍐欏叆骞傜瓑璁板綍; 鍏ラ槦澶辫触涓嶅啓, 鍏佽閲嶈瘯銆?
### 15.5 甯歌閿欒鐮侀€熸煡

| 涓氬姟鐮?| 鍏稿瀷鍦烘櫙 |
|--------|----------|
| 0 | 鎴愬姛 |
| 40000 | 鍙傛暟閿欒: 瀛楁缂哄け/闈炴硶銆侀獙璇佺爜閿欒銆佹ā鏉跨己澶便€佹笭閬撶被鍨嬩笉鏀寔銆乧onfirm_password 涓嶄竴鑷?|
| 40100 | 鏈璇? 缂哄皯鍑瘉銆佸嚟璇佹棤鏁堟垨宸茶繃鏈熴€佺櫥褰曢偖绠辨垨瀵嗙爜閿欒銆乀OTP 楠岃瘉澶辫触 |
| 40300 | 鏃犳潈闄?褰撳墠鐢ㄦ埛涓績 API 杈冨皯瑙﹀彂) |
| 40400 | 璧勬簮涓嶅瓨鍦? 娑堟伅/妯℃澘/璁㈠崟/浠诲姟/绔欏唴淇′笉瀛樺湪, 鎴栬秺鏉冭闂粬浜鸿祫婧?|
| 40900 | 鍐茬獊: 閭宸叉敞鍐屻€佹ā鏉?code 閲嶅銆佸吋瀹?key 宸插瓨鍦?|
| 42900 | 闄愭祦: 娉ㄥ唽楠岃瘉鐮?60s 鍐峰嵈銆佹秷鎭彂閫侀鐜囪秴闄?|
| 50000 | 鏈嶅姟绔敊璇? 鏁版嵁搴撳紓甯搞€丼MTP 鏈厤缃€佹槗鏀粯鏈厤缃瓑 |
| 50300 | 骞冲彴灏氭湭瀹夎, 璇峰厛璁块棶 /install 瀹屾垚瀹夎 |

---

*鏈枃妗ｅ瓧娈典互浠撳簱浠ｇ爜涓哄噯, 鑻ヤ笌杩愯鐗堟湰瀛樺湪鍑哄叆, 璇蜂互鏈€鏂颁唬鐮佷负鍑嗐€傜鐞嗗悗鍙?API(/admin/*)涓嶅湪鏈枃妗ｈ寖鍥淬€?
