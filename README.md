## سناریوهای Go — Concurrency, Channels, Context, Mutex, Pub/Sub

این مخزن مجموعه‌ای از سناریوهای آموزشی Go است که با تمرکز بر همزمانی، الگوهای ارتباط بین گوروتین‌ها، مدیریت زمان/لغو با Context، قفل‌ها (Mutex/RWMutex)، و ادغام با Redis (Cache + Pub/Sub) طراحی شده‌اند.


### راه‌اندازی سرویس‌ها (برای سناریوهای شبکه‌ای)
در صورت نیاز به Redis/Kafka (به‌خصوص برای P06):
```bash
docker compose up -d
```
سرویس‌ها:
- Redis روی پورت `6379`
- Zookeeper روی پورت `2181`
- Kafka روی پورت `9092` (در حال حاضر در کدها استفاده نشده، اما در compose حاضر است)

---

### P06 — Redis Price Cache + Pub/Sub (Publisher/Subscriber)
- پکیج: `P06/pricecache` (API سطح پایین Cache/Subscribe)
- برنامه‌ها:
  - Publisher: `P06/pricepub/main.go`
  - Subscriber: `P06/pricesub/main.go`
- نحوه‌ی راه‌اندازی سرویس‌ها:
```bash
docker compose up -d   # Redis/Kafka
```
- اجرای Publisher:
```bash
go run P06/pricepub/main.go \ 
  -redis=localhost:6379 \ 
  -symbols=BTC,ETH,ADA \ 
  -rate=500ms \ 
  -ttl=5s \ 
  -ch=prices
```
- اجرای Subscriber:
```bash
go run P06/pricesub/main.go \ 
  -redis=localhost:6379 \ 
  -symbol=BTC \ 
  -interval=2s \ 
  -channel=prices
```
- رفتار:
  - Publisher برای هر نماد با یک random walk قیمت تولید می‌کند، در Redis با TTL ذخیره و روی کانال Pub/Sub منتشر می‌کند (Set+Publish اتمیک با `TxPipeline`).
  - Subscriber پیام‌های زنده را از Pub/Sub لاگ می‌کند و به‌صورت ادواری cache را می‌خواند تا انقضای TTL را مشاهده کنید.

---

## ساختار پوشه‌ها
```
scenarios/
  P01/  # محاسبات آماری + تست/بنچمارک
  P02/  # WaitGroup + Mutex روی فایل‌ها
  P03/  # Mutex vs RWMutex + تست/بنچمارک
  P04/  # Context/Timeout + Channel در HTTP
  P05/  # Pipeline چندمرحله‌ای با Channels/Context
  P06/  # Redis cache + Pub/Sub (publisher/subscriber)
  docker-compose.yml  # سرویس‌های کمکی (Redis/ZooKeeper/Kafka)
  go.mod
```

## نکات آموزشی کلّی
- **Context**: برای مدیریت مهلت‌ها و لغو تمیز در مرز گوروتین‌ها
- **Channels**: ارتباط امن داده و اعمال backpressure طبیعی
- **Mutex/RWMutex**: حفاظت از state مشترک؛ RWMutex برای read-heavy مناسب‌تر است
- **Atomic**: شمارنده‌های سبک و خواندن/نوشتن بدون قفل
- **تست/بنچمارک**: ارزیابی عملکرد و صحت سناریوها

## اشکال‌زدایی و اجرای محلی
- لاگ‌ها با `log.SetFlags(log.Ltime|log.Lmicroseconds)` زمان دقیق را نمایش می‌دهند.
- برای مشاهده‌ی رفتارهای زمانی، تاخیرها (Sleep/Ticker) را تغییر دهید.
- پیش از اجرای سناریوهای وابسته به سرویس، از در دسترس بودن آن‌ها اطمینان حاصل کنید (`docker compose ps`).
