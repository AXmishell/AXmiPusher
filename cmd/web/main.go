// 前端托管主程序(双端口): 用户中心 + 管理后台独立端口, 各自反代 /api 到主程序。
//
// 端口规划(环境变量可覆盖):
//   MP_USER_PORT   用户中心端口   默认 19876   (路由: / 用户中心, /api 反代)
//   MP_ADMIN_PORT  管理后台端口   默认 19877   (路由: /{admin_path}/ 管理后台, /api 反代)
//   MP_API_TARGET  后端 API 地址   默认 http://127.0.0.1:8080(主程序)
//   MP_USER_DIST   用户中心 dist   默认 web/user/dist
//   MP_ADMIN_DIST  管理后台 dist   默认 web/admin/dist
//   MP_ADMIN_PATH  管理后台随机路径 默认 b322aa9602150d0c(须与 admin 构建 base 一致)
//
// 主程序(api :8080)负责统一路径: / 用户中心, /{admin_path}/ 管理后台, /api API。
// 本程序提供两个独立端口直连前端, 便于内网访问/独立部署, 不依赖主程序托管。
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	userPort := env("MP_USER_PORT", "19876")
	adminPort := env("MP_ADMIN_PORT", "19877")
	userDist := env("MP_USER_DIST", "web/user/dist")
	adminDist := env("MP_ADMIN_DIST", "web/admin/dist")
	adminPath := strings.Trim(env("MP_ADMIN_PATH", "b322aa9602150d0c"), "/")
	apiTarget := env("MP_API_TARGET", "http://127.0.0.1:8080")

	apiURL, err := url.Parse(apiTarget)
	if err != nil {
		log.Fatalf("MP_API_TARGET 无效: %v", err)
	}

	adminPrefix := "/" + adminPath + "/"

	// 用户中心端口: / 用户中心 SPA + /api 反代。
	userMux := http.NewServeMux()
	userMux.Handle("/api/", httputil.NewSingleHostReverseProxy(apiURL))
	userMux.Handle("/", spaHandler(userDist, ""))

	// 管理后台端口: /{admin_path}/ 管理后台 SPA + /api 反代; 根路径重定向到 admin 前缀。
	adminMux := http.NewServeMux()
	adminMux.Handle("/api/", httputil.NewSingleHostReverseProxy(apiURL))
	adminMux.Handle(adminPrefix, spaHandler(adminDist, adminPrefix))
	adminMux.Handle("/"+adminPath, http.RedirectHandler(adminPrefix, http.StatusMovedPermanently))
	adminMux.Handle("/", http.RedirectHandler(adminPrefix, http.StatusMovedPermanently))

	// 双端口并发监听。
	go func() {
		log.Printf("用户中心  : http://0.0.0.0:%s  (dist: %s)", userPort, userDist)
		if err := http.ListenAndServe(":"+userPort, userMux); err != nil {
			log.Fatalf("用户中心端口 %s 退出: %v", userPort, err)
		}
	}()
	log.Printf("管理后台  : http://0.0.0.0:%s%s  (dist: %s, API→%s)", adminPort, adminPrefix, adminDist, apiTarget)
	if err := http.ListenAndServe(":"+adminPort, adminMux); err != nil {
		log.Fatalf("管理后台端口 %s 退出: %v", adminPort, err)
	}
}

// spaHandler 静态文件优先; 目录/无扩展名未命中回退 index.html(SPA 路由);
// 带扩展名的静态资源缺失返回 404(避免浏览器把 HTML 当 JS/CSS 解析)。
// stripPrefix 非空时, 先按前缀剥离 URL 再映射 dist(http.StripPrefix 保证 FileServer 命中)。
func spaHandler(distDir, stripPrefix string) http.Handler {
	fs := http.StripPrefix(stripPrefix, http.FileServer(http.Dir(distDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(r.URL.Path, "/")
		if stripPrefix != "" {
			rel = strings.TrimPrefix(rel, strings.TrimPrefix(stripPrefix, "/"))
		}
		full := filepath.Join(distDir, filepath.Clean(rel))
		if _, err := os.Stat(full); err != nil {
			if filepath.Ext(rel) != "" {
				http.NotFound(w, r)
				return
			}
			// SPA 回退: 直接返回 index.html。
			http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
