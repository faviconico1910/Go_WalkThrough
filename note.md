## gopsutil

- Lấy CPU usage: cpuPercent, err := cpu.Percent(time.Second, false)
- Lấy RAM usage: ramUsage, err := mem.VirtualMemory()

## Định nghĩa Struct:

type struct_name struct {
member1 datatype;
member2 datatype;
member3 datatype;
...
}

- Từ struct -> Json: Dùng marshal
- Từ json -> struct: Unmarshal

## Goroutines & Channels

- Goroutine: các luồng thực thi ảo cực kì nhẹ

Go<function name>(<tham số>)

- Channel: cung cấp cho các Goroutine

```Tạo channel: <tên channel> := make(chan <kiểu>)

Gửi vào channel: <tên channel> <- <giá trị>

Lấy dữ liệu ra từ channel: <biến lưu trữ giá trị> <- <tên channel>
```

## CPU usage type

- CPU User: thời gian để CPU xử lý các tác vụ liên quan đến ứng dụng đang chạy trên máy, ví dụ như database server

- CPU System: thời gian CPU được sử dụng bởi kernel, ví dụ như tương tác với phần cứng, cấp phát bộ nhớ, giao tiếp giữa các tiến trình của hệ điều hành, drivers, file system, scheduler

- CPU IOWait: Thời gian CPU phải chờ đợi thao tác IO, như việc đọc và lưu xuống ổ đĩa

## Memory

- Free memory: Là RAM hoàn toàn không hoạt động. Trong hệ điều hành hiện đại như Linux, lượng Free memory thường rất thấp vì hệ thống ưu tiên dùng RAM dư thừa để lưu trữ tạm thời (cache) nhằm tăng tốc độ truy cập ứng dụng.
- Available memory: Đo lường chính xác lượng RAM thực sự có thể sử dụng cho các tác vụ mới mà không làm hệ thống bị treo.
  Công thức tính:

  `Available memory (in %) = ((Free+Cached)/TotalMem)*100. Trong thư viện, đã có sẵn free + cached + buffer là 1 trường Available.`

- Mount Point: Các folder như /, /home, /var, ...
- Mount: Hành động lấy ổ đĩa vật lí và gắn vào một thư mục, ví dụ mount ổ SSD 1T riêng vào /data, thì dữ liệu được lưu trong /data sẽ không ở trong ổ đĩa chính mà sẽ được lưu thẳng xuống SSD 1T

## Network

- TCP (Transmission Control Protocol): TCP bắt buộc hai phía phải bắt tay 3 bước để xác nhận kết nối, giúp đảm bảo dữ liệu không bị mất, không bị lỗi và đến đúng thứ tự.

- Port ở trạng thái LISTEN: Nghĩa là có một phần mềm/dịch vụ (như Web Server, Database) đang chạy ngầm trên hệ điều hành, chủ động mở cổng đó ra và chờ các yêu cầu kết nối gửi tới từ client. Nếu không có phần mềm nào LISTEN ở cổng đó, mọi yêu cầu kết nối gửi tới sẽ bị hệ điều hành từ chối (Connection Refused). Sau LISTEN là ESTABLISHED.

- Kiểm tra kết nối TCP (vài cách thường dùng)

  ping

  telnet: kiểm tra server có mở port 8080: telnet 192.168.1.10 8080

  nc/netcat: nc -vz 192.168.1.10 8080, nc -vz 192.168.1.10 22 80 443 8080 (kiểm tra nhiều port)

  Powershell: Test-NetConnection google.com -Port 443

Ví dụ trong Go: kiểm tra xem server có đang mở port 3306 không? net.DialTimeout("tcp", "192.168.1.10:3306", time.Second\*3)

## CHECK OS Linux/Windows

- Dùng runtime.GOOS
