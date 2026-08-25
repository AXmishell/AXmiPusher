// 前端托管主程序(双端口): 用户中心 + 管理后台独立端口, 各自反代 /api 到主程序。
//
// 端口固化在 config.yaml(安装向导写入, 默认: 主程序 8080 / 用户中心 19876 / 管理后台 19877):
//   server:
//     port: 8080
//   web:
//     user_port: 19876
//     admin_port: 19877
//     api_target: http://127.0.0.1:8080
//     user_dist: web/user/dist
//     admin_dist: web/admin/dist
//   admin:
//     random_path: <随机串>
//
// 优先级: MP_* 环境变量 > config.yaml > 默认值(与主程序一致)。
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"messagepusher/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	userPort := cfg.Web.UserPort
	adminPort := cfg.Web.AdminPort
	userDist := cfg.Web.UserDist
	adminDist := cfg.Web.AdminDist
	apiTarget := cfg.Web.APITarget
	adminPath := strings.Trim(cfg.Admin.RandomPath, "/")

	// 空值兜底(配置未写时)。
	if userDist == "" {
		userDist = "web/user/dist"
	}
	if adminDist == "" {
		adminDist = "web/admin/dist"
	}
	if apiTarget == "" {
		apiTarget = "http://127.0.0.1:8080"
	}
	if adminPath == "" {
		adminPath = "b322aa9602150d0c"
	}

	apiURL, err := url.Parse(apiTarget)
	if err != nil {
		log.Fatalf("api_target 无效: %v", err)
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
		log.Printf("用户中心  : http://0.0.0.0:%d  (dist: %s)", userPort, userDist)
		if err := http.ListenAndServe(":"+strconv.Itoa(userPort), userMux); err != nil {
			log.Fatalf("用户中心端口 %d 退出: %v", userPort, err)
		}
	}()
	log.Printf("管理后台  : http://0.0.0.0:%d%s  (dist: %s, API→%s)", adminPort, adminPrefix, adminDist, apiTarget)
	if err := http.ListenAndServe(":"+strconv.Itoa(adminPort), adminMux); err != nil {
		log.Fatalf("管理后台端口 %d 退出: %v", adminPort, err)
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
