#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
AXmiPusher Android APK 正式签名 CLI 工具(纯标准库, 零第三方依赖, 跨平台)。

一键封装 AXmiPusher(RN Android 客户端)的完整签名工作流:
  keytool 生成 keystore → keystore.properties 配置签名 → gradlew assembleRelease 构建签名 APK
  → apksigner 验证 → base64 导出(供 GitHub Secret ANDROID_KEYSTORE_BASE64)。

用法示例(6 个子命令, 均在 mobile/ 目录下执行):

  python scripts/android_signer.py generate                      # 生成新 keystore 并完成全部配置
  python scripts/android_signer.py sign --version 1.2.3          # 构建并签名 release APK
  python scripts/android_signer.py verify                        # 验证 APK 签名是否为正式 keystore
  python scripts/android_signer.py base64 --output out.b64       # 导出 keystore 的单行 base64
  python scripts/android_signer.py secrets                       # 打印 GitHub Secrets 配置指引
  python scripts/android_signer.py backup                        # 备份签名材料(keystore/base64/README)

安全说明:
  - 密码绝不写入任何日志/临时文件, 仅在 generate 时打印一次; 其他命令只指引密码来源。
  - base64 文件为单行无换行, 可直接用于 GitHub Secret ANDROID_KEYSTORE_BASE64。
"""

import argparse
import base64
import glob
import hashlib
import os
import platform
import re
import secrets
import shutil
import subprocess
import sys
import time
from pathlib import Path

# ---------------------------------------------------------------- 路径常量
# 脚本位于 mobile/scripts/, 据此自动定位项目根与 android 工程(跨平台)
SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_ROOT = SCRIPT_DIR.parent            # mobile/
ANDROID_DIR = PROJECT_ROOT / "android"
APP_DIR = ANDROID_DIR / "app"

DEFAULT_KEYSTORE = APP_DIR / "release.keystore"
DEFAULT_KEYSTORE_PROPS = APP_DIR / "keystore.properties"
DEFAULT_APK = APP_DIR / "build" / "outputs" / "apk" / "release" / "app-release.apk"
DEFAULT_BACKUP_DIR = Path.home() / "AXmiPusher-ReleaseSigning"
BASE64_FILENAME = "axmipusher-android.keystore.base64"

DEFAULT_ALIAS = "axmipusher"

# 密码字符集(24 位字母数字, 兼容 keytool 与各类构建工具对密码的字符限制)
_PASS_ALPHABET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

# 输出前缀
P_OK = "[OK]"
P_WARN = "[警告]"
P_ERR = "[错误]"

# ---------------------------------------------------------------- 基础助手


def is_windows():
    """是否 Windows 平台。"""
    return platform.system() == "Windows"


def force_utf8_io():
    """强制 stdout/stderr 使用 UTF-8 输出。

    Windows 控制台默认代码页(如 GBK)无法编码 ✓ 等 Unicode 字符且中文易乱码,
    统一输出 UTF-8 保证跨平台显示一致(现代终端 / Windows Terminal 均支持)。
    """
    for stream in (sys.stdout, sys.stderr):
        try:
            stream.reconfigure(encoding="utf-8", errors="replace")
        except (AttributeError, ValueError, OSError):
            pass


def die(msg):
    """打印错误并以非零码退出。"""
    print(P_ERR + " " + msg)
    sys.exit(1)


def _timestamp():
    """当前时间戳, 用于备份文件名。"""
    return time.strftime("%Y%m%d-%H%M%S")


def decode_bytes(b):
    """将子进程字节输出解码为文本(兼容 UTF-8 / 系统代码页 GBK / Latin-1)。"""
    if not b:
        return ""
    for enc in ("utf-8", "gbk", "latin-1"):
        try:
            return b.decode(enc)
        except (UnicodeDecodeError, LookupError):
            continue
    return b.decode("utf-8", errors="replace")


def run_capture(args, cwd=None, env=None):
    """执行命令并捕获输出, 返回 (returncode, stdout文本, stderr文本)。"""
    proc = subprocess.run(args, capture_output=True,
                          cwd=str(cwd) if cwd else None, env=env)
    return proc.returncode, decode_bytes(proc.stdout), decode_bytes(proc.stderr)


def run_stream(args, cwd=None, env=None):
    """执行命令, 输出直接透传到终端(用于 gradle 长构建), 返回 returncode。"""
    return subprocess.run(args, cwd=str(cwd) if cwd else None, env=env).returncode


def gen_password(length=24):
    """生成 length 位字母数字密码(secrets 密码学安全随机)。"""
    return "".join(secrets.choice(_PASS_ALPHABET) for _ in range(length))


def file_sha256(path):
    """计算文件 SHA-256(分块读取, 兼容大文件)。"""
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def read_keystore_props(props_path):
    """读取 keystore.properties, 返回 dict; 缺失或不可读返回 None。"""
    p = Path(props_path)
    if not p.is_file():
        return None
    text = None
    for enc in ("utf-8", "gbk"):
        try:
            text = p.read_text(encoding=enc)
            break
        except (OSError, UnicodeDecodeError):
            continue
    if text is None:
        return None
    props = {}
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, _, v = line.partition("=")
        props[k.strip()] = v.strip()
    return props or None


def keystore_to_base64(ks_path):
    """读 keystore 字节并返回单行 base64 文本(无换行)。"""
    return base64.b64encode(Path(ks_path).read_bytes()).decode("ascii")


def write_base64_file(b64, out_path):
    """将 base64 文本单行写入文件(不含换行符)。"""
    out_path.write_text(b64, encoding="ascii")


# ---------------------------------------------------------------- 工具探测

def find_keytool():
    """定位 keytool: $JAVA_HOME → 常见安装路径(含 glob) → PATH 兜底。"""
    ext = ".exe" if is_windows() else ""
    candidates = []
    jh = os.environ.get("JAVA_HOME")
    if jh:
        candidates.append(Path(jh) / "bin" / ("keytool" + ext))
    patterns = [
        "/usr/lib/jvm/*/bin/keytool",                       # Linux 常见 JVM 目录
        r"C:\Program Files\Android\Android Studio\jbr\bin\keytool.exe",
        r"C:\Program Files\Java\*\bin\keytool.exe",
        r"D:\AppData\Android\Android Studio\jbr\bin\keytool.exe",
        r"D:\AppData\Android\jdk-17\jdk-17.0.20.1+1\bin\keytool.exe",
    ]
    for pat in patterns:
        if "*" in pat:
            candidates.extend(Path(p) for p in glob.glob(pat))
        else:
            candidates.append(Path(pat))
    # 最后用 PATH 兜底(Android Studio 命令行环境常见)
    which = shutil.which("keytool" + ext) or shutil.which("keytool")
    if which:
        candidates.append(Path(which))
    for c in candidates:
        if c.is_file():
            return c
    return None


def _parse_sdk_version(name):
    """把 build-tools 目录名(如 37.0.0)解析为可比较的版本元组。"""
    parts = []
    for s in str(name).split("."):
        try:
            parts.append(int(s))
        except ValueError:
            parts.append(0)
    return tuple(parts)


def _sdk_root_candidates():
    """候选 Android SDK 根目录: 环境变量优先, 其次平台默认目录。"""
    cands = []
    for v in ("ANDROID_HOME", "ANDROID_SDK_ROOT"):
        if os.environ.get(v):
            cands.append(Path(os.environ[v]))
    if is_windows():
        la = os.environ.get("LOCALAPPDATA")
        if la:
            cands.append(Path(la) / "Android" / "Sdk")
        # 常见自定义安装目录(与 keytool 探测中的 D:\\AppData 路径对应)
        cands.append(Path(r"D:\AppData\Android\Sdk"))
    elif sys.platform == "darwin":
        cands.append(Path.home() / "Library" / "Android" / "sdk")
    else:
        cands.append(Path.home() / "Android" / "Sdk")
    return cands


def find_apksigner():
    """定位 apksigner: SDK 根目录下 build-tools/<版本号最大>/apksigner(.bat)。"""
    exe = "apksigner.bat" if is_windows() else "apksigner"
    best = None
    best_ver = None
    for sdk in _sdk_root_candidates():
        bt = sdk / "build-tools"
        if not bt.is_dir():
            continue
        for d in bt.iterdir():
            if not d.is_dir():
                continue
            p = d / exe
            if p.is_file():
                ver = _parse_sdk_version(d.name)
                if best_ver is None or ver > best_ver:
                    best_ver, best = ver, p
    return best


def _jdk_major(jdk_root):
    """读取 JDK 的 release 文件得到主版本号; 失败返回 None。"""
    try:
        rel = (Path(jdk_root) / "release").read_text(encoding="utf-8", errors="ignore")
    except OSError:
        return None
    m = re.search(r'JAVA_VERSION="(\d+)', rel)
    return int(m.group(1)) if m else None


def find_jdk_home():
    """定位可用的 JDK 根目录(优先 Java 17, 因 RN gradle 插件强制要求 17 工具链)。

    顺序: $JAVA_HOME(已设置则直接使用) → 常见 JDK 安装路径(优先 17 版本)。
    返回 JDK 根目录 Path 或 None。
    """
    ext = ".exe" if is_windows() else ""
    jh = os.environ.get("JAVA_HOME")
    if jh and (Path(jh) / "bin" / ("java" + ext)).is_file():
        return Path(jh)

    roots = []
    for pat in [
        r"D:\AppData\Android\jdk-17\jdk-17.0.20.1+1",   # 本机已知 JDK 17
        r"C:\Program Files\Java\*",
        r"D:\AppData\Android\Android Studio\jbr",
        r"C:\Program Files\Android\Android Studio\jbr",
        "/usr/lib/jvm/*",                                # Linux 常见 JVM 目录
    ]:
        if "*" in pat:
            roots.extend(Path(p) for p in glob.glob(pat))
        else:
            roots.append(Path(pat))

    # 第一轮: 优先 17 版本(满足 RN gradle 插件工具链要求, 避免触发在线下载 17)
    for r in roots:
        if (r / "bin" / ("java" + ext)).is_file() and _jdk_major(r) == 17:
            return r
    # 第二轮: 任意可用 JDK 兜底(仍比没有 JAVA_HOME 强)
    for r in roots:
        if (r / "bin" / ("java" + ext)).is_file():
            return r
    return None


def find_gradlew():
    """定位 gradlew: 脚本自定位项目根(mobile/), gradle 工程为项目根/android。"""
    g = ANDROID_DIR / ("gradlew.bat" if is_windows() else "gradlew")
    return g if g.is_file() else None


# ---------------------------------------------------------------- 正则常量(verify 用)
_CERT_DN_RE = re.compile(r"certificate\s+DN:\s*(.+)")
_CERT_SHA256_RE = re.compile(r"certificate\s+SHA-256\s+digest:\s*([0-9a-fA-F]{64})")
_KEYTOOL_FP_RE = re.compile(r"([0-9A-Fa-f]{2}(?::[0-9A-Fa-f]{2}){31})")


def normalize_fp(fp):
    """把 '9D:2E:CE:...' 归一化为小写无分隔十六进制。"""
    return re.sub(r"[^0-9a-fA-F]", "", fp).lower()


# ---------------------------------------------------------------- 子命令: generate

def prompt_dname():
    """交互式输入证书 DN(隐私: 不固化任何默认值, 仅 CN 必填, 其余字段可跳过)。"""
    print(P_OK + " 开始输入证书 DN(将写入 APK 签名证书; 隐私起见脚本不内置默认值):")
    print("   CN 必填(通常填公司/组织/应用名), 其余字段回车即跳过。")
    cn = input("   CN(组织/名称, 必填): ").strip()
    if not cn:
        die("CN 必填, 已取消生成。")
    parts = [f"CN={cn}"]
    for key, label in [("OU", "组织单位 OU"), ("O", "组织 O"), ("L", "城市 L"), ("ST", "省/州 ST")]:
        v = input(f"   {label}(回车跳过): ").strip()
        if v:
            parts.append(f"{key}={v}")
    c = input("   C(国家代码, 2 位, 回车跳过): ").strip()
    if c:
        parts.append(f"C={c}")
    dname = ", ".join(parts)
    print(P_OK + f" 证书 DN: {dname}")
    return dname


def cmd_generate(args):
    """生成新 keystore 并完成全部配置(keystore.properties + base64 + GitHub Secret 表)。"""
    ks = Path(args.keystore)
    backup_dir = Path(args.backup_dir)
    keytool = find_keytool()
    if keytool is None:
        die("未找到 keytool。请设置 JAVA_HOME 或安装 JDK 17+, 或安装 Android Studio(自带 JBR)。")

    if args.password and len(args.password) < 6:
        die("--password 长度不足: keytool 要求密码至少 6 位。")

    # 已存在则询问式保护: keytool 无法覆盖已存在的不同密码 keystore
    if ks.exists():
        print(P_WARN + f" keystore 已存在: {ks}")
        print(P_WARN + " 重新生成将覆盖现有正式签名, 可能导致已安装用户无法升级!")
        try:
            ans = input(" 确认继续? 输入 yes 继续, 其他任意键取消: ").strip()
        except (EOFError, KeyboardInterrupt):
            die("已取消")
        if ans.lower() != "yes":
            die("已取消(未做任何修改)")
        # 备份旧 keystore 再删除, 避免 keytool 拒绝覆盖, 也防止旧签名永久丢失
        backup_dir.mkdir(parents=True, exist_ok=True)
        old_bak = backup_dir / ("release.keystore.old-" + _timestamp())
        shutil.copy2(ks, old_bak)
        print(P_OK + f" 旧 keystore 已备份到: {old_bak}")
        ks.unlink()

    password = args.password or gen_password()
    print(P_OK + f" keystore 密码: {password}  (仅显示这一次, 请妥善保存)")

    # DN 隐私: 不固化在脚本里, 未显式传入 --dname 时交互式输入(仅 CN 必填)
    dname = args.dname or prompt_dname()

    cmd = [str(keytool), "-genkeypair", "-v",
           "-keystore", str(ks),
           "-alias", args.alias,
           "-keyalg", "RSA", "-keysize", "2048",
           "-validity", "36500",
           "-storepass", password, "-keypass", password,
           "-dname", dname]
    print(P_OK + " 正在生成 keystore(RSA 2048, 100 年有效期)...")
    rc, out, err = run_capture(cmd)
    if rc != 0:
        print(P_ERR + " keytool 生成失败:")
        print(err.strip() or out.strip())
        sys.exit(1)
    print(P_OK + f" keystore 已生成: {ks}")

    # 在 keystore 同目录写 keystore.properties(build.gradle 读取后自动签名)
    props_path = ks.parent / "keystore.properties"
    props_text = (f"storeFile={ks.name}\n"
                  f"storePassword={password}\n"
                  f"keyAlias={args.alias}\n"
                  f"keyPassword={password}\n")
    props_path.write_text(props_text, encoding="utf-8")
    print(P_OK + f" 已写入签名配置: {props_path}")

    # 生成 base64 到备份目录(单行, 供 GitHub Secret ANDROID_KEYSTORE_BASE64)
    backup_dir.mkdir(parents=True, exist_ok=True)
    b64_file = backup_dir / BASE64_FILENAME
    write_base64_file(keystore_to_base64(ks), b64_file)
    print(P_OK + f" 已导出单行 base64: {b64_file}")

    # 打印 GitHub Secret 配置表(生成时打印一次, 含密码)
    print()
    print(P_OK + " 签名配置完成! GitHub Secret 配置表(仓库 Settings → Secrets and variables → Actions):")
    print(f"    ANDROID_KEYSTORE_BASE64    = {b64_file} 文件内容(单行 base64, 直接复制粘贴)")
    print(f"    ANDROID_KEYSTORE_PASSWORD  = {password}")
    print(f"    ANDROID_KEY_ALIAS          = {args.alias}")
    print(f"    ANDROID_KEY_PASSWORD       = {password}")


# ---------------------------------------------------------------- 子命令: sign

def cmd_sign(args):
    """构建并签名 release APK(实际走 gradlew assembleRelease, build.gradle 已集成签名)。"""
    gradlew = find_gradlew()
    if gradlew is None:
        die(f"未找到 gradle 包装脚本({ANDROID_DIR / 'gradlew'})。请确认脚本位于 mobile/scripts/ 下。")

    # 默认加 --no-daemon, 避免后台守护进程占用内存/锁
    cmd = [str(gradlew), "assembleRelease", "--no-daemon"]
    env = os.environ.copy()
    # 关键: 未设置 JAVA_HOME 时自动探测 JDK(优先 17)。
    # 否则 gradlew 会用 PATH 上的 java(可能非 17), RN gradle 插件要求 17 工具链,
    # 会触发 foojay 在线下载 JDK 17 → 无代理环境下长时间卡死。
    if not env.get("JAVA_HOME"):
        jdk = find_jdk_home()
        if jdk:
            env["JAVA_HOME"] = str(jdk)
            env["PATH"] = str(jdk / "bin") + os.pathsep + env.get("PATH", "")
            print(P_OK + f" 检测到 JDK: {jdk}(RN 构建需要 Java 17 工具链)")
        else:
            print(P_WARN + " 未检测到 JDK, 使用系统 PATH 上的 java; 若报 17 工具链错误请设置 JAVA_HOME 指向 JDK 17")
    if args.version:
        env["APP_VERSION"] = args.version
        print(P_OK + f" 注入 APP_VERSION={args.version}(versionName/versionCode 随之生成)")

    print(P_OK + f" 开始构建 release APK(工作目录: {ANDROID_DIR}, 输出如下)...")
    rc = run_stream(cmd, cwd=ANDROID_DIR, env=env)
    if rc != 0:
        die(f"gradle 构建失败(退出码 {rc})。请检查上方构建日志。")

    if DEFAULT_APK.is_file():
        size_mb = DEFAULT_APK.stat().st_size / 1048576
        print(P_OK + f" 构建完成: {DEFAULT_APK} ({size_mb:.1f} MB)")
        print(P_OK + " 可运行以下命令验证签名: python scripts/android_signer.py verify")
    else:
        print(P_WARN + " 构建成功但未找到 APK 产物, 请检查输出路径。")


# ---------------------------------------------------------------- 子命令: verify

def cmd_verify(args):
    """验证 APK 签名: apksigner 输出证书信息, 与本地 keystore 指纹对比。"""
    apk = Path(args.apk)
    ks = Path(args.keystore)
    if not apk.is_file():
        die(f"APK 不存在: {apk}\n请先构建: python scripts/android_signer.py sign")

    apksigner = find_apksigner()
    if apksigner is None:
        die("未找到 apksigner。请设置 ANDROID_HOME / ANDROID_SDK_ROOT, 或安装 Android SDK 到平台默认目录。")

    rc, out, err = run_capture([str(apksigner), "verify", "--print-certs", str(apk)])
    if rc != 0:
        print(P_ERR + " apksigner 验证失败(APK 签名无效或工具报错):")
        print(err.strip() or out.strip())
        sys.exit(1)

    text = out + "\n" + err
    dn_m = _CERT_DN_RE.search(text)
    sha_m = _CERT_SHA256_RE.search(text)
    if not dn_m or not sha_m:
        print(P_WARN + " 未能从 apksigner 输出解析出证书信息, 原文如下:")
        print(text.strip())
        sys.exit(1)
    apk_dn = dn_m.group(1).strip()
    apk_sha = sha_m.group(1).strip().lower()
    print(P_OK + f" APK 签名证书: {apk_dn}")
    print(P_OK + f" 证书 SHA-256: {apk_sha}")

    # keystore 不存在 → 跳过对比, 仅展示 APK 自身签名信息
    if not ks.is_file():
        print(P_WARN + f" 未找到 keystore({ks}), 跳过与本地 keystore 对比。")
        return

    # 从 keystore.properties 读密码做指纹对比; 缺失或密码不符时降级为仅展示
    props = read_keystore_props(ks.parent / "keystore.properties")
    if props is None:
        print(P_WARN + " 未找到 keystore.properties, 无法读取 keystore 密码, 跳过对比。")
        print(P_WARN + " 提示: 可运行 'python scripts/android_signer.py generate' 重建签名配置。")
        return
    storepass = props.get("storePassword")
    alias = props.get("keyAlias")
    if not storepass or not alias:
        print(P_WARN + " keystore.properties 缺少 storePassword/keyAlias, 跳过对比。")
        return

    keytool = find_keytool()
    if keytool is None:
        print(P_WARN + " 未找到 keytool, 无法对比 keystore 指纹。")
        return
    rc2, out2, err2 = run_capture([str(keytool), "-list", "-keystore", str(ks),
                                   "-storepass", storepass, "-alias", alias])
    if rc2 != 0:
        first_line = (err2.strip() or out2.strip()).splitlines()
        print(P_WARN + " keytool 读取 keystore 失败(可能密码不符):")
        print("  " + (first_line[0][:120] if first_line else "未知错误"))
        print(P_WARN + " 提示: 密码不符或 keystore 损坏时请用 'generate' 重建。")
        return
    fp_m = _KEYTOOL_FP_RE.search(out2 + "\n" + err2)
    if not fp_m:
        print(P_WARN + " 未能从 keytool 输出解析出 SHA-256 指纹, 跳过对比。")
        return
    ks_sha = normalize_fp(fp_m.group(1))

    if ks_sha == apk_sha:
        print(P_OK + f" 正式签名 ✓ {apk_dn}")
    else:
        print(P_WARN + f" 非当前 keystore 签名(APK 与本地 keystore 指纹不一致, 如 debug 签名的 CN=...androiddebugkey)")
        print(P_WARN + f" 本地 keystore SHA-256: {ks_sha}")


# ---------------------------------------------------------------- 子命令: base64

def cmd_base64(args):
    """导出 keystore 的单行 base64 文本文件(用于 GitHub Secret ANDROID_KEYSTORE_BASE64)。"""
    ks = Path(args.keystore)
    if not ks.is_file():
        die(f"keystore 不存在: {ks}\n请先运行 'generate' 生成。")
    out_path = Path(args.output) if args.output else (DEFAULT_BACKUP_DIR / BASE64_FILENAME)
    out_path.parent.mkdir(parents=True, exist_ok=True)

    b64 = keystore_to_base64(ks)
    write_base64_file(b64, out_path)
    print(P_OK + f" 已导出单行 base64: {out_path}")
    print(P_OK + f" 内容长度: {len(b64)} 字符(单行, 无换行)")
    print(P_OK + f" keystore SHA-256: {file_sha256(ks)}")
    print(P_OK + " 可直接复制文件内容到 GitHub Secret ANDROID_KEYSTORE_BASE64。")


# ---------------------------------------------------------------- 子命令: secrets

def cmd_secrets(args):
    """打印 GitHub Secrets 配置指引(名称与值来源, 不重复打印密码)。"""
    props = read_keystore_props(Path(args.keystore).parent / "keystore.properties")
    if props is None:
        die("未找到 keystore.properties, 无法生成配置指引。\n请先运行 'generate' 重建签名配置。")
    storepass = props.get("storePassword")
    keypass = props.get("keyPassword")
    alias = props.get("keyAlias") or args.alias
    if not storepass or not keypass:
        die("keystore.properties 缺少 storePassword/keyPassword, 请用 'generate' 重建。")

    b64_file = Path(args.backup_dir) / BASE64_FILENAME
    b64_len = None
    if b64_file.is_file():
        try:
            b64_len = len(b64_file.read_text(encoding="ascii"))
        except (OSError, UnicodeDecodeError):
            b64_len = None

    print("GitHub Secrets 配置指引(仓库 Settings → Secrets and variables → Actions, 供 release.yml 正式签名):")
    print(f"  ANDROID_KEYSTORE_BASE64    = {b64_file} 文件内容(单行 base64)" +
          (f", 当前 {b64_len} 字符" if b64_len is not None else ", 尚未生成"))
    print("  ANDROID_KEYSTORE_PASSWORD  = keystore.properties 中的 storePassword")
    print(f"  ANDROID_KEY_ALIAS          = keystore.properties 中的 keyAlias(当前: {alias})")
    print("  ANDROID_KEY_PASSWORD       = keystore.properties 中的 keyPassword")
    print("提示: 密码仅生成时打印一次, 请从 keystore.properties 或备份目录 README.md 中复制(本命令不重复打印密码)。")
    if b64_len is None:
        print(P_WARN + " 尚未导出 base64, 可先运行: python scripts/android_signer.py base64")


# ---------------------------------------------------------------- 子命令: backup

def _creds_table(props, alias_fallback):
    """构造凭据表 + GitHub Secret 配置表(Markdown)。"""
    if props is None:
        storepass, keypass = "未知(keystore.properties 缺失)", "未知"
        alias = alias_fallback
    else:
        storepass = props.get("storePassword") or "未知"
        keypass = props.get("keyPassword") or "未知"
        alias = props.get("keyAlias") or alias_fallback
    return f"""## 凭据表
| 项目 | 值 |
|---|---|
| storeFile | release.keystore |
| storePassword | {storepass} |
| keyAlias | {alias} |
| keyPassword | {keypass} |

## GitHub Secret 配置表
| Secret | 值来源 |
|---|---|
| ANDROID_KEYSTORE_BASE64 | {BASE64_FILENAME} 文件内容(单行, 无换行) |
| ANDROID_KEYSTORE_PASSWORD | 上表 storePassword |
| ANDROID_KEY_ALIAS | 上表 keyAlias |
| ANDROID_KEY_PASSWORD | 上表 keyPassword |"""


def _build_readme(props):
    """生成完整备份 README 内容。"""
    ks = Path(args.keystore)
    sha = file_sha256(ks)
    return f"""# AXmiPusher Android 正式签名备份

生成时间: {time.strftime("%Y-%m-%d %H:%M:%S")}
keystore SHA-256: {sha}

## 文件清单
- release.keystore                         — 正式签名 keystore(核心机密)
- {BASE64_FILENAME}            — keystore 的单行 base64(供 GitHub Secret ANDROID_KEYSTORE_BASE64)
- README.md                                — 本说明文件(凭据 / 配置 / 验证)

{_creds_table(props, DEFAULT_ALIAS)}

## 验证命令
cd <项目根>/mobile
python scripts/android_signer.py verify    # 应输出「正式签名 ✓」(DN 为生成时填写的信息)

## ⚠ 丢失 / 泄露警告
- 丢失 keystore: 无法再对已安装用户推送升级, 只能换包名重新发布。
- 泄露 keystore / 密码: 他人可用同一身份伪造签名版本, 冒充官方更新。
- 请将此目录离线加密保管; 切勿提交到 Git 仓库、上传公开网盘或粘贴到聊天工具。
"""


def _append_readme_update(readme, props):
    """在已存在的 README.md 末尾追加/更新凭据部分。"""
    block = f"""
---
## 凭据更新记录
更新时间: {time.strftime("%Y-%m-%d %H:%M:%S")}

{_creds_table(props, DEFAULT_ALIAS)}
"""
    with open(readme, "a", encoding="utf-8") as f:
        f.write(block)


def cmd_backup(args):
    """一键备份签名材料(keystore + base64 + README)到备份目录。"""
    ks = Path(args.keystore)
    if not ks.is_file():
        die(f"keystore 不存在: {ks}\n请先运行 'generate' 生成。")
    backup_dir = Path(args.backup_dir)
    backup_dir.mkdir(parents=True, exist_ok=True)

    # 1. 复制 keystore
    ks_dst = backup_dir / "release.keystore"
    shutil.copy2(ks, ks_dst)
    print(P_OK + f" 已复制 keystore → {ks_dst}")

    # 2. 生成单行 base64
    b64_file = backup_dir / BASE64_FILENAME
    write_base64_file(keystore_to_base64(ks), b64_file)
    print(P_OK + f" 已生成单行 base64 → {b64_file}")

    # 3. 写入 README(已存在则追加/更新凭据部分)
    props = read_keystore_props(ks.parent / "keystore.properties")
    if props is None:
        print(P_WARN + " 未找到 keystore.properties, README 凭据表将标记为未知(请用 'generate' 重建)。")
    readme = backup_dir / "README.md"
    if readme.is_file():
        _append_readme_update(readme, props)
        print(P_OK + f" README.md 已存在, 已追加/更新凭据部分: {readme}")
    else:
        readme.write_text(_build_readme(props), encoding="utf-8")
        print(P_OK + f" 已写入备份说明: {readme}")

    print(P_WARN + " 请将备份目录妥善保管(离线/加密存储), 切勿上传到公开网络或提交 Git!")


# ---------------------------------------------------------------- 参数解析

def build_parser():
    parser = argparse.ArgumentParser(
        prog="android_signer.py",
        description="AXmiPusher Android APK 正式签名 CLI 工具(纯标准库, 跨平台)。"
                    "封装 keytool 生成 keystore / gradle 签名构建 / apksigner 验证 / base64 导出 / 备份。",
        epilog="示例: python scripts/android_signer.py generate",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", metavar="子命令")

    # generate
    p_gen = sub.add_parser(
        "generate",
        help="生成新 keystore 并完成全部配置(keystore.properties + base64 + GitHub Secret 表)",
        description="生成新 keystore 并完成全部配置: 运行 keytool 生成 RSA 2048 / 100 年有效期 keystore, "
                    "在同目录写入 keystore.properties, 导出 base64 到备份目录, 打印 GitHub Secret 配置表。",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p_gen.set_defaults(func=cmd_generate)
    p_gen.add_argument("--keystore", default=str(DEFAULT_KEYSTORE),
                       help=f"keystore 路径(默认: {DEFAULT_KEYSTORE})")
    p_gen.add_argument("--alias", default=DEFAULT_ALIAS,
                       help=f"key 别名(默认: {DEFAULT_ALIAS})")
    p_gen.add_argument("--password", default=None,
                       help="keystore/密钥密码(默认: 自动生成 24 位字母数字密码并仅打印一次)")
    p_gen.add_argument("--backup-dir", default=str(DEFAULT_BACKUP_DIR),
                       help=f"备份目录, 存放 base64 与旧 keystore 备份(默认: {DEFAULT_BACKUP_DIR})")
    p_gen.add_argument("--dname", default=None,
                       help="证书 DN(如 'CN=MyOrg, O=MyOrg'); 不传则交互式输入(仅 CN 必填, 隐私起见不内置默认值)")

    # sign
    p_sign = sub.add_parser(
        "sign",
        help="构建并签名 release APK(gradlew assembleRelease, build.gradle 已集成签名)",
        description="构建并签名 release APK: 在 android/ 下执行 gradlew assembleRelease。"
                    "产物: android/app/build/outputs/apk/release/app-release.apk。",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p_sign.set_defaults(func=cmd_sign)
    p_sign.add_argument("--version", default=None, metavar="X.Y.Z",
                        help="版本号, 注入 APP_VERSION 环境变量给 gradle(如 1.2.3 → versionCode 10203)")
    p_sign.add_argument("--no-daemon", action="store_true",
                        help="构建时加 --no-daemon(默认已启用, 该参数仅为显式声明)")

    # verify
    p_ver = sub.add_parser(
        "verify",
        help="验证 APK 签名(apksigner, 与本地 keystore 指纹对比)",
        description="验证 APK 签名: 用 apksigner 打印证书 DN 与 SHA-256; 若本地 keystore 存在, "
                    "再用 keytool 取 keystore 指纹对比, 一致则输出「正式签名 ✓」, 否则提示非当前 keystore 签名。",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p_ver.set_defaults(func=cmd_verify)
    p_ver.add_argument("--apk", default=str(DEFAULT_APK),
                       help=f"待验证 APK 路径(默认: {DEFAULT_APK})")
    p_ver.add_argument("--keystore", default=str(DEFAULT_KEYSTORE),
                       help=f"本地 keystore 路径, 不存在时跳过对比(默认: {DEFAULT_KEYSTORE})")

    # base64
    p_b64 = sub.add_parser(
        "base64",
        help="导出 keystore 的单行 base64 文件(用于 GitHub Secret ANDROID_KEYSTORE_BASE64)",
        description="导出 keystore 的 base64 单行文本文件(不含换行), 直接用于 GitHub Secret ANDROID_KEYSTORE_BASE64。",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p_b64.set_defaults(func=cmd_base64)
    p_b64.add_argument("--keystore", default=str(DEFAULT_KEYSTORE),
                       help=f"keystore 路径(默认: {DEFAULT_KEYSTORE})")
    p_b64.add_argument("--output", default=None,
                       help=f"输出文件路径(默认: {DEFAULT_BACKUP_DIR / BASE64_FILENAME})")

    # secrets
    p_sec = sub.add_parser(
        "secrets",
        help="打印 GitHub Secrets 配置指引(名称与值来源)",
        description="打印 GitHub Secrets 配置指引: 从 keystore.properties 读密码, 从 base64 文件读长度, "
                    "打印 4 个 Secret 的名称与值来源(不重复打印密码本身)。",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p_sec.set_defaults(func=cmd_secrets)
    p_sec.add_argument("--keystore", default=str(DEFAULT_KEYSTORE),
                       help=f"keystore 路径, 其同目录须有 keystore.properties(默认: {DEFAULT_KEYSTORE})")
    p_sec.add_argument("--alias", default=DEFAULT_ALIAS,
                       help=f"key 别名, keystore.properties 缺失 keyAlias 时兜底(默认: {DEFAULT_ALIAS})")
    p_sec.add_argument("--backup-dir", default=str(DEFAULT_BACKUP_DIR),
                       help=f"备份目录, 读取 base64 文件长度(默认: {DEFAULT_BACKUP_DIR})")

    # backup
    p_bak = sub.add_parser(
        "backup",
        help="一键备份签名材料到备份目录(keystore + base64 + README)",
        description="一键备份签名材料: 复制 keystore 为 release.keystore, 生成单行 base64, 写入 README.md"
                    "(文件清单 / 凭据表 / Secret 配置表 / 验证命令 / 丢失泄露警告)。",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p_bak.set_defaults(func=cmd_backup)
    p_bak.add_argument("--keystore", default=str(DEFAULT_KEYSTORE),
                       help=f"keystore 路径(默认: {DEFAULT_KEYSTORE})")
    p_bak.add_argument("--backup-dir", default=str(DEFAULT_BACKUP_DIR),
                       help=f"备份目录(默认: {DEFAULT_BACKUP_DIR})")

    return parser


def main():
    force_utf8_io()
    parser = build_parser()
    args = parser.parse_args()
    if not getattr(args, "func", None):
        parser.print_help()
        sys.exit(0)
    try:
        args.func(args)
    except KeyboardInterrupt:
        die("已取消")


if __name__ == "__main__":
    main()
