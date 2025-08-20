## توضیح خط‌به‌خط و Flow کد P05/main.go (Pipeline با Channels، Context، RWMutex و Atomic)

این برنامه یک خط‌لوله (Pipeline) داده‌ای را با چند مرحله پیاده‌سازی می‌کند:
- Stage0: تولید داده‌ی سنسور (Reading)
- Stage1: فیلتر مقادیر نامعتبر و زیر آستانه
- Stage2: تبدیل دما از سلسیوس به فارنهایت
- Stage3: ذخیره‌سازی Thread-safe و گزارش‌گیری
- مصرف‌کننده (Consumer): اتصال مراحل و نوشتن در Store

در همه‌ی مراحل از `context.Context` برای لغو (Cancelation) امن، از Channel برای انتقال داده با Backpressure طبیعی، از `sync.RWMutex` برای دسترسی همزمان به ذخیره، و از `sync/atomic` برای شمارش Thread-safe استفاده می‌شود.

### خروجی نهایی چه کاری انجام می‌دهد؟
- ۲۰۰ نمونه دما از سنسور شبیه‌سازی می‌شود (هر ۲۰۰ms یک نمونه).
- مقادیر خارج از بازه‌ی [-50..100] یا کمتر از آستانه‌ی ۳۰°C حذف می‌شوند.
- باقی داده‌ها به فارنهایت تبدیل شده و در Store ذخیره می‌شوند.
- در پایان تعداد «نگه‌داشته‌شده» و «حذف‌شده» چاپ می‌شود و یک Snapshot نمونه نمایش داده می‌شود.

---

### Flow کلی برنامه
- Stage0 (Sensor): تولید پیوسته‌ی `Reading` روی Channel خروجی.
- Stage1 (Filter): دریافت از Stage0، حذف داده‌های نامعتبر، ارسال باقی‌مانده به خروجی.
- Stage2 (Transform): دریافت Reading و تولید `Processed` با مقدار فارنهایت.
- Stage3 (Store + Consumer): خواندن از خروجی Transform و ذخیره‌ی Thread-safe در حافظه.
- در پایان `wg.Wait()` تا اتمام مصرف‌کننده صبر می‌کند، سپس گزارش چاپ می‌شود.

---

### توضیح خط‌به‌خط (با اشاره به شماره خطوط)

#### L1–L11: Package و Imports
- تعریف پکیج `main` و وارد کردن کتابخانه‌های موردنیاز:
  - `context`: مدیریت لغو و مهلت‌ها (timeout/deadline)
  - `fmt`, `log`: چاپ و لاگ‌گیری
  - `math/rand`: تولید نویز تصادفی دما
  - `sync`, `sync/atomic`: همگام‌سازی و شمارش Thread-safe
  - `time`: کار با زمان و Ticker

#### L13–L25: ساختار داده‌ها
- L13–L19: `Reading`
  - `SensorID`, `Sequence`, `Celsius`, `At`
- L22–L25: `Processed`
  - شامل `Reading` و فیلد اضافی `Fahrenheit`

#### L27–L63: Stage0 - تابع `Sensor`
- L29: امضای تابع: `Sensor(ctx, sensorID, n, interval) <-chan Reading`
  - خروجی: Channel بدون‌بافر برای ایجاد Backpressure طبیعی
- L30: `out := make(chan Reading)`
- L31–L33: راه‌اندازی گوروتین تولیدکننده + `defer close(out)`
- L34–L35: `time.NewTicker(interval)` برای زمان‌بندی تولید
- L37–L60: حلقه‌ی اصلی با `select`
  - L39–L41: در صورت `ctx.Done()`، لاگ «canceled» و خروج
  - L43–L58: روی تیک هر بازه:
    - `seq++` و تولید مقدار با نویز: `25 + rand.NormFloat64()*8`
    - ساخت `Reading` و ارسال امن روی Channel با `select` (بررسی لغو)
    - شرط پایان تولید: اگر `n>0` و `seq>=n`، لاگ و خروج
- L62–L63: بازگرداندن Channel خروجی

#### L65–L93: Stage1 - تابع `Filter`
- L67: امضا: `Filter(ctx, in, min, max, threshold, dropped *int64) <-chan Reading`
- L68–L71: گوروتین فیلتر و `defer close(out)`
- L71–L90: `for r := range in` دریافت از ورودی تا بسته‌شدن
  - L73–L78: بررسی لغو Context
  - L80–L83: قواعد فیلتر:
    - اگر `r.Celsius` خارج از `[min..max]` یا کمتر از `threshold` بود: `atomic.AddInt64(dropped, 1)` و `continue`
  - L84–L88: ارسال Reading معتبر به خروجی یا خروج در صورت لغو
- L90–L91: لاگ بسته‌شدن ورودی و بستن خروجی

#### L95–L118: Stage2 - تابع `Transform`
- L96: امضا: `Transform(ctx, in) <-chan Processed`
- L98–L116: گوروتین تبدیل + بستن خروجی در پایان
  - L100–L106: بررسی لغو Context
  - L107–L113: تبدیل C→F: `f = r.Celsius*9/5 + 32` و ساخت `Processed`
  - L109–L113: ارسال امن با بررسی لغو
- L115–L116: لاگ بسته‌شدن ورودی

#### L120–L150: Stage3 - Store (ذخیره‌سازی Thread-safe)
- L122–L126: ساختار `Store`
  - `mu sync.RWMutex`: برای خواندن همزمان و نوشتن انحصاری
  - `data []Processed`: آرایه‌ی داده‌ها
  - `count int64`: شمارنده‌ی Thread-safe کل اقلام ذخیره‌شده
- L128–L132: سازنده `NewStore()` با ظرفیت اولیه 1024
- L134–L139: متد `Append(p Processed)`
  - `mu.Lock()` → `append` → `mu.Unlock()`
  - سپس `atomic.AddInt64(&s.count, 1)` (خارج از قفل برای کاهش زمان نگه‌داشت قفل)
- L141–L147: متد `Snapshot()`
  - `RLock()` → کپی امن از Slice → `RUnlock()`
- L149–L150: متد `Count()`
  - `atomic.LoadInt64(&s.count)` برای خواندن بدون قفل

#### L151–L164: Consumer - تابع `Consume`
- L152–L154: `wg.Done()` با `defer`
- L154–L162: حلقه روی ورودی `in`
  - `select` برای بررسی لغو Context
  - در حالت عادی: `st.Append(p)`
- L163–L164: لاگ پایان مصرف پس از بسته‌شدن ورودی

#### L166–L209: تابع `main` (سیم‌کشی Pipeline)
- L168: تنظیم فرمت لاگ با میکروثانیه‌ها
- L169: بذر تصادفی `rand.Seed`
- L171–L172: `parent, cancel := context.WithCancel(context.Background())` و `defer cancel()`
- L174–L176: Stage0: سنسور با ۲۰۰ نمونه، هر ۲۰۰ms
- L177: تعریف شمارنده‌ی حذف‌شده‌ها: `var dropped int64`
- L178–L180: Stage1: فیلتر با بازه‌ی معتبر `[-50..100]` و آستانه‌ی `30°C`
- L181–L183: Stage2: تبدیل به فارنهایت
- L184–L189: Stage3: ساخت Store و راه‌اندازی Consumer با `WaitGroup`
- L190–L191: انتظار تا اتمام Consumer (`wg.Wait()`)
- L193–L199: گزارش‌گیری و چاپ نتایج:
  - `totalKept := store.Count()`
  - `totalDropped := atomic.LoadInt64(&dropped)`
  - چاپ گزارش و نمایش نمونه‌ی اول/آخر از Snapshot (اگر وجود داشته باشد)
- L208–L209: لاگ خروج main

---

### چرا Channel؟ چرا Context؟ چرا RWMutex؟ چرا Atomic؟
- **Channel (بدون‌بافر)**: برای انتقال امن بین گوروتین‌ها و ایجاد Backpressure طبیعی؛ مصرف‌کننده‌ی کند سرعت تولید را محدود می‌کند.
- **Context**: امکان لغو تمیز و Propagation سیگنال لغو به تمام مراحل.
- **RWMutex در Store**: خواندن‌های همزمان (Snapshot) بدون بلاک کردن هم را ممکن می‌کند؛ نوشتن انحصاری است.
- **Atomic**: شمارش سریع و Thread-safe بدون نیاز به گرفتن قفل؛ مناسب برای شمارنده‌های ساده.

### نکات مهم همزمانی (Best Practices)
- بستن Channel‌ها تنها در سمت تولیدکننده انجام می‌شود.
- حداقل‌سازی ناحیه‌ی Critical: نگه‌داشتن قفل فقط در لحظه‌ی `append`.
- استفاده از `select` برای ارسال/دریافت همراه با بررسی لغو Context.
- برگشت تمیز با `defer` برای `close(...)`، `cancel()` و `wg.Done()`.

### اجرای برنامه
```bash
go run P05/main.go
```

نمونه‌ای از خروجی (مقادیر واقعی تصادفی است):
```
[sensor] produced 200 readings, closing...
[filter] input closed > closing out
[transform] input closed > closing out
[store] input closed > done

=== REPORT ===
Kept:   87 readings (stored)
Dropped:113 readings (filtered out)
First kept: seq=3, C=31.24, F=88.23
Last  kept: seq=200, C=45.02, F=113.04
[main] exit
```

### تغییرات احتمالی برای آزمایش
- **بی‌نهایت کردن تولید**: در `main` مقدار `n` را در `Sensor` به `<=0` تغییر دهید.
- **تغییر نرخ تولید**: پارامتر `interval` را افزایش/کاهش دهید.
- **قواعد فیلتر**: بازه‌ی مجاز (`min/max`) و آستانه (`threshold`) را تغییر دهید.
- **ظرفیت Store**: ظرفیت اولیه‌ی Slice یا نوع قفل را تغییر/بررسی کنید.

---

### جمع‌بندی
این کد نمونه‌ای روشن از طراحی Pipeline در Go است که با استفاده از Channel، Context، RWMutex و Atomic، جریان داده را به‌شکل همزمان، امن و قابل‌لغو مدیریت می‌کند. ساختار مرحله‌ای باعث سادگی توسعه، تست و تغییر مستقل هر بخش می‌شود. 