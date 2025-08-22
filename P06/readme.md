## P06 — توضیح خط‌به‌خط استفاده از Redis برای Pub/Sub (Price Cache)

در این سناریو از Redis برای ساخت یک سیستم Pub/Sub سبک استفاده شده است که قیمت نمادها (مثل BTC/ETH/ADA) را منتشر می‌کند و مشترک‌ها (Subscribers) آن‌ها را به صورت زنده دریافت می‌کنند. علاوه‌بر پیام‌های زنده، آخرین قیمت هر نماد در Redis با TTL ذخیره می‌شود تا مشترک‌ها بتوانند از کش نیز بخوانند.

### راه‌اندازی سریع
- سرویس‌ها را اجرا کنید:
```bash
docker compose up -d   # Redis روی 6379
```
- Publisher:
```bash
go run P06/pricepub/main.go -redis=localhost:6379 -symbols=BTC,ETH,ADA -rate=500ms -ttl=5s -ch=prices
```
- Subscriber:
```bash
go run P06/pricesub/main.go -redis=localhost:6379 -symbol=BTC -interval=2s -channel=prices
```

---

## معماری
- Publisher برای هر نماد، قیمت را به‌روزرسانی می‌کند و در یک تراکنش Redis هم «در کش» می‌نویسد و هم «روی کانال Pub/Sub» منتشر می‌کند.
- Subscriber همزمان دو رفتار دارد:
  - شنیدن پیام‌های زنده از کانال Pub/Sub
  - خواندن دوره‌ای از کش برای مشاهده انقضای TTL

---

## پکیج `pricecache` — API سطح پایین برای Redis
فایل: `P06/pricecache/pricecache.go`

- L13–L17: ساختار `Cache`
  - `redis *redis.Client`: کلاینت Redis
  - `ttl time.Duration`: مدت اعتبار کش
  - `channel string`: نام کانال Pub/Sub
- L19–L21: سازنده `New(redis, ttl, channel)`
- L23–L25: تابع `keyFor(symbol)`
  - کلید کش را مثل `price:BTC` می‌سازد.

- L27–L41: `UpsertPrice(ctx, p Price) error`
  - L30–L33: تبدیل `Price` به JSON برای ذخیره/انتشار
  - L35–L39: استفاده از `TxPipelined` برای انجام اتمیک دو عملیات:
    - `Set(key, payload, ttl)`: ذخیره JSON با TTL
    - `Publish(channel, payload)`: انتشار همان JSON روی کانال
  - نکته: اتمیک بودن باعث می‌شود «نوشتن کش» و «انتشار پیام» با هم انجام شوند و مشترک‌ها پیام ناسازگار نبینند.

- L43–L58: `GetPrice(ctx, symbol)`
  - `Get` کلید؛ تبدیل `redis.Nil` به `ErrNotFound`
  - Unmarshal از JSON به `Price`

- L60–L83: `Subscribe(ctx, handle func(Price) error)`
  - `redis.Subscribe(ctx, c.channel)` → شیء PubSub
  - `sub.Channel()` → کانال Go از پیام‌های Redis
  - حلقه‌ی `for` با `select`:
    - اگر `ctx.Done()` → خروج تمیز
    - اگر `msg` رسید → Unmarshal JSON به `Price` و فراخوانی `handle(p)`
  - نکته: این تابع تا لغو Context یا بسته‌شدن کانال بلاک می‌ماند.

فایل: `P06/pricecache/model.go`
- L5–L9: ساختار `Price` با تگ‌های JSON: `symbol`, `price`, `at`

فایل: `P06/pricecache/errors.go`
- L6: `ErrNotFound` برای زمانی که کلید کش وجود ندارد/منقضی شده است.

---

## برنامه Publisher — `P06/pricepub/main.go`
هدف: تولید قیمت برای چند نماد و «Set+Publish» اتمیک هر مقدار در Redis.

- L20–L25: فلگ‌ها
  - `-redis`: آدرس Redis
  - `-symbols`: فهرست نمادها (CSV)
  - `-rate`: نرخ به‌روزرسانی هر نماد
  - `-ttl`: مدت اعتبار هر قیمت در کش
  - `-ch`: نام کانال Pub/Sub
- L28–L35: ساخت کلاینت Redis و `pricecache.New`
- L37: لاگ شروع با نمایش پارامترها
- L39–L43: ایجاد `time.Ticker` جدا برای هر نماد
- L45–L49: `last` به عنوان آخرین قیمت هر نماد (برای Random Walk)
- L51–L73: حلقه‌ی اصلی
  - `select` روی `ctx.Done()` برای خروج تمیز با سیگنال
  - برای هر نماد، وقتی `ticker.C` تیک زد:
    - L61–L63: به‌روزرسانی قیمت به شکل Random Walk و ساخت `Price{...}`
    - L63–L67: `pc.UpsertPrice(ctx, p)` → اتمیک Set+Publish
    - لاگ وضعیت (موفق/خطا)

- L76–L86: `splitAndTrim` — تبدیل CSV به لیست نمادها (Upper-case)
- L88–L94: `round` — رندکردن قیمت برای لاگ‌خوانایی

نتیجه: با هر تیک برای هر نماد، یک پیام JSON روی کانال منتشر می‌شود و همان Payload در کش ذخیره می‌گردد.

---

## برنامه Subscriber — `P06/pricesub/main.go`
هدف: دریافت زنده‌ی پیام‌ها از Pub/Sub و مشاهده‌ی کش برای TTL.

- L19–L23: فلگ‌ها
  - `-redis`: آدرس Redis
  - `-symbol`: نمادی که به صورت دوره‌ای از کش خوانده می‌شود
  - `-interval`: فاصله‌ی بین خواندن‌های کش
  - `-channel`: نام کانال Pub/Sub
- L27–L33: ساخت کلاینت Redis و `pricecache.New(redis, 0, channel)`
  - `ttl=0` یعنی این برنامه خودش چیزی در کش نمی‌نویسد؛ فقط می‌خواند و Subscribe می‌کند.

- L35–L44: راه‌اندازی Listener Pub/Sub در یک گوروتین
  - `priceCache.Subscribe(ctx, handler)`
  - در `handler`: لاگ `LIVE symbol=price @ time` برای پیام‌های زنده
  - اگر `context.Canceled` نبود و خطایی آمد، لاگ خطا

- L47–L63: خواندن دوره‌ای از کش
  - `Ticker` با `interval`
  - هر بار: `priceCache.GetPrice(ctx, symbol)`
    - اگر خطا (مثلاً TTL منقضی شده) → `CACHE miss`
    - اگر موفق → لاگ `CACHED symbol=price @ time`

نتیجه: مشترک هم پیام‌های لحظه‌ای Pub/Sub را می‌بیند و هم وضعیت کش را برای مشاهده‌ی انقضای TTL بررسی می‌کند.

---

## فرمت پیام‌ها
- Payloadها JSON هستند و ساختارشان `Price` است:
```json
{"symbol":"BTC","price":12345.67,"at":"2024-01-01T12:34:56Z"}
```
- همان Payload هم در کش ذخیره می‌شود و هم روی کانال منتشر می‌گردد.

---

## نکات کلیدی پیاده‌سازی Pub/Sub با Redis
- Set+Publish اتمیک با `TxPipelined`: از عدم ناسازگاری بین کش و پیام جلوگیری می‌کند.
- لغو تمیز با `context`: هر دو برنامه با سیگنال سیستم (`SIGINT/SIGTERM`) خاتمه‌ی امن دارند.
- استفاده از `PubSub.Channel()`: تبدیل Pub/Sub Redis به کانال Go برای پردازش ساده در حلقه‌ی `select`.
- TTL برای کش: مشترک می‌تواند با خواندن دوره‌ای، انقضای داده را مشاهده کند (هم‌زمان با دریافت پیام‌های زنده).

---

## دستورات نمونه
- اجرای Publisher با نرخ 500ms و TTL پنج ثانیه:
```bash
go run P06/pricepub/main.go -redis=localhost:6379 -symbols=BTC,ETH,ADA -rate=500ms -ttl=5s -ch=prices
```
- اجرای Subscriber و مانیتور نماد BTC هر 2 ثانیه از کش:
```bash
go run P06/pricesub/main.go -redis=localhost:6379 -symbol=BTC -interval=2s -channel=prices
```
