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

## Mount Point Logic

- Tận dụng chính hàm disk.Partitions(false) của thư viện gopsutil. Hàm này:

- Trên Windows: Nó sẽ quét và trả về danh sách các ổ đĩa đang có.

- Trên Linux: Nó sẽ quét file hệ thống và trả về tất cả các Mount Point đang được gắn vào cây thư mục (/, /data, /home).

# 15/6/2026

- Sự khác biệt giữa cpu.Percent(time.Second, false) và cpu.Percent(0, false)

cpu.Percent(time.Second, false): Trả về phần trăm CPU được sử dụng tại thời điểm gọi, sau đó sleep 1s, rồi lại trả về kết quả

cpu.Percent(0, false): Trả về phần trăm CPU trung bình được sử dụng trong khoảng thời gian giữa hai lần gọi hàm.

Xử lý khi hệ thống bị rớt mạng hoặc backend trả về lỗi

**Khi mất mạng**:

- Dùng Mutex Lock để khóa bảo vệ bộ nhớ (tránh xung đột dữ liệu giữa các luồng).

- Đẩy dữ liệu vào Memory Queue (RAM) theo cơ chế FIFO, tối đa 300 bản ghi, để không làm mất dữ liệu và không làm tràn RAM của máy.

**Khi có mạng lại**:

- Một Goroutine chạy ngầm phát hiện ra server đã sống lại.
- Kích hoạt chế độ gửi bù: Rút từng bản ghi trong RAM ra gửi, giãn cách nhau 1-2 giây để bảo vệ backend không bị sập vì quá tải.

# 21/6/2026: Tính IOPS

- `disk.IOCounters()`: Tổng số lần đọc (ReadCount) và Tổng số lần ghi (WriteCount) tích lũy kể từ lúc máy tính khởi động.

```
B1: Tại thời điểm T1: gọi disk.IOCounters(), lấy ReadCount_1 và WriteCount_1.
B2: Chờ 1 khoảng thời gian ΔT
B3: Tại thời điểm T2: gọi disk.IOCOunters() lần 2, lấy ReadCount_2 và WriteCount_2.
B4: Công thức tính
- Read IOPS = (ReadCount_2 - ReadCount_1) / ΔT
- Write IOPS = (WriteCount_2 - WriteCount_1) / ΔT

```

# 23/6/2026: bảo mật trong agent
