# 本地 mock SMTP 服务器: 监听 2525 端口, 接受 SMTP 对话并把邮件正文写入 data/smtp.log
$listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 2525)
$listener.Start()
Add-Content -Path "D:\Code\messagepusher\data\smtp.log" -Value "=== smtp mock started $(Get-Date -Format o) ==="
while ($true) {
    try {
        $client = $listener.AcceptTcpClient()
        $stream = $client.GetStream()
        $reader = [System.IO.StreamReader]::new($stream)
        $writer = [System.IO.StreamWriter]::new($stream)
        $writer.AutoFlush = $true
        $writer.WriteLine("220 mock-smtp ESMTP")
        $inData = $false
        $mailBody = [System.Text.StringBuilder]::new()
        while ($client.Connected) {
            $line = $reader.ReadLine()
            if ($null -eq $line) { break }
            if ($inData) {
                if ($line -eq ".") {
                    $inData = $false
                    Add-Content -Path "D:\Code\messagepusher\data\smtp.log" -Value "--- MESSAGE ---"
                    Add-Content -Path "D:\Code\messagepusher\data\smtp.log" -Value $mailBody.ToString()
                    Add-Content -Path "D:\Code\messagepusher\data\smtp.log" -Value "--- END ---"
                    $mailBody.Clear() | Out-Null
                    $writer.WriteLine("250 2.0.0 OK")
                } else {
                    [void]$mailBody.AppendLine($line)
                }
                continue
            }
            $cmd = $line.ToUpper()
            if ($cmd.StartsWith("EHLO") -or $cmd.StartsWith("HELO")) { $writer.WriteLine("250-mock-smtp"); $writer.WriteLine("250 AUTH PLAIN") }
            elseif ($cmd.StartsWith("AUTH")) { $writer.WriteLine("235 2.7.0 Accepted") }
            elseif ($cmd.StartsWith("MAIL FROM")) { $writer.WriteLine("250 2.1.0 OK") }
            elseif ($cmd.StartsWith("RCPT TO")) { $writer.WriteLine("250 2.1.5 OK") }
            elseif ($cmd.StartsWith("DATA")) { $inData = $true; $writer.WriteLine("354 End data with <CR><LF>.<CR><LF>") }
            elseif ($cmd.StartsWith("QUIT")) { $writer.WriteLine("221 2.0.0 Bye"); break }
            elseif ($cmd.StartsWith("RSET")) { $writer.WriteLine("250 2.0.0 OK") }
            else { $writer.WriteLine("250 2.0.0 OK") }
        }
        $client.Close()
    } catch {
        Add-Content -Path "D:\Code\messagepusher\data\smtp.log" -Value "ERROR: $_"
    }
}
