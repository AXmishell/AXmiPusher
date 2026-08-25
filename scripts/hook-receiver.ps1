# 本地 Webhook 接收器: 监听 9090 端口, 收到 POST 后写入收到的请求到 data/hook.log
$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add("http://localhost:9090/")
$listener.Start()
Add-Content -Path "D:\Code\messagepusher\data\hook.log" -Value "receiver started $(Get-Date -Format o)"
while ($listener.IsListening) {
    try {
        $ctx = $listener.GetContext()
        $req = $ctx.Request
        $reader = New-Object System.IO.StreamReader($req.InputStream)
        $body = $reader.ReadToEnd()
        $line = "$(Get-Date -Format o) | $($req.HttpMethod) $($req.Url.AbsolutePath) | $($req.Headers['X-MP-Signature']) | $body"
        Add-Content -Path "D:\Code\messagepusher\data\hook.log" -Value $line
        $ctx.Response.StatusCode = 200
        $resp = '{"ok":true}'
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($resp)
        $ctx.Response.OutputStream.Write($bytes, 0, $bytes.Length)
        $ctx.Response.Close()
    } catch {
        Add-Content -Path "D:\Code\messagepusher\data\hook.log" -Value "ERROR: $_"
    }
}
