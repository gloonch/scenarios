## توضیح خط‌به‌خط و Flow کد P04/main.go (درک عملی Channels و Context در Go)

این برنامه چندین URL را به صورت همزمان (Concurrent) دانلود می‌کند و نتایج را از طریق Channel جمع‌آوری می‌کند. برای مدیریت Timeout از `context.Context`، برای همگام‌سازی از `sync.WaitGroup`، و برای انتقال نتایج از `chan Result` استفاده شده است. این کد نمونه‌ای از الگوی Producer-Consumer با گوروتین‌ها است.

### خروجی نهایی چه کاری انجام می‌دهد؟
- چندین URL به صورت همزمان دانلود می‌شوند.
- هر دانلود دارای Timeout جداگانه است (3 ثانیه).
- نتایج از طریق Channel جمع‌آوری می‌شوند.
- در پایان آمار کلی (موفق/ناموفق/مجموع بایت‌ها) چاپ می‌شود.

---

### Flow کلی برنامه
- تنظیم فرمت لاگ‌ها و تعریف لیست URLها.
- ایجاد Context اصلی با قابلیت Cancel.
- ایجاد Channel برای نتایج و WaitGroup برای همگام‌سازی.
- برای هر URL:
  - راه‌اندازی گوروتین دانلود با `wg.Add(1)`.
  - داخل گوروتین: دانلود با Timeout، ارسال نتیجه به Channel.
- گوروتین جداگانه برای بستن Channel بعد از اتمام همه‌ی کارها.
- حلقه‌ی خواندن نتایج از Channel و جمع‌آوری آمار.

---

### توضیح خط‌به‌خط (با اشاره به شماره خطوط)

#### L1–L12: Imports و تعریف ساختار Result
- L1: `package main` - تعریف پکیج اصلی.
- L3–L11: Import کتابخانه‌های مورد نیاز:
  - `context`: برای مدیریت Timeout و Cancel.
  - `io`: برای خواندن Response Body.
  - `log`: برای لاگ‌گیری.
  - `net/http`: برای درخواست‌های HTTP.
  - `sync`: برای WaitGroup.
  - `time`: برای اندازه‌گیری زمان.

- L13–L19: ساختار `Result`
  - `URL`: آدرس دانلود شده.
  - `Bytes`: تعداد بایت‌های دانلود شده.
  - `Err`: خطای احتمالی.
  - `Elapsed`: زمان سپری‌شده.
  - `StatusCode`: کد وضعیت HTTP.

#### L21–L25: تنظیمات اولیه main
- L21: `log.SetFlags(log.Ltime | log.Lmicroseconds)`
  - تنظیم فرمت لاگ‌ها برای نمایش زمان دقیق.
- L22–L26: تعریف لیست URLها
  - `mobile.ir`: سایت سریع.
  - `httpbin.org/delay/2`: تأخیر 2 ثانیه‌ای.
  - `httpbin.org/delay/5`: تأخیر 5 ثانیه‌ای (احتمالاً Timeout خواهد شد).

#### L28–L30: ایجاد Context و Channel
- L28: `parent, cancelAll := context.WithCancel(context.Background())`
  - ایجاد Context اصلی با قابلیت Cancel برای همه‌ی گوروتین‌ها.
- L29: `defer cancelAll()`
  - تضمین Cleanup در پایان برنامه.
- L30: `results := make(chan Result)`
  - ایجاد Channel برای انتقال نتایج از گوروتین‌ها به main.

#### L32–L33: تعریف WaitGroup
- L32: `var wg sync.WaitGroup`
  - برای شمارش و انتظار تا اتمام همه‌ی گوروتین‌های دانلود.

#### L35–L46: راه‌اندازی گوروتین‌های دانلود
- L35: `log.Printf("[main] launching %d downloads", len(urls))`
  - لاگ تعداد دانلودهای شروع شده.
- L36: حلقه روی همه‌ی URLها
- L37: `wg.Add(1)`
  - قبل از راه‌اندازی هر گوروتین، شمارنده‌ی WaitGroup افزایش می‌یابد.
- L38–L45: راه‌اندازی گوروتین دانلود
  - L38: `go func(url string) {`
    - گوروتین جدید با پارامتر `url` (از Capturing متغیر حلقه جلوگیری شده).
  - L39: `defer wg.Done()`
    - تضمین کاهش شمارنده‌ی WaitGroup در پایان گوروتین.
  - L40: `res := fetchWithTimeout(parent, url, 3*time.Second)`
    - فراخوانی تابع دانلود با Timeout 3 ثانیه‌ای.
  - L41–L44: `select` برای ارسال نتیجه
    - `case results <- res:`: ارسال نتیجه به Channel.
    - `case <-parent.Done()`: اگر Context Cancel شده باشد، نتیجه ارسال نمی‌شود.
  - L45: `}(u)` - بستن گوروتین با پارامتر URL.

#### L48–L51: گوروتین بستن Channel
- L48: `go func() {`
  - گوروتین جداگانه برای بستن Channel.
- L49: `wg.Wait()`
  - انتظار تا اتمام همه‌ی گوروتین‌های دانلود.
- L50: `close(results)`
  - بستن Channel بعد از اتمام همه‌ی کارها.
- L51: `}()` - بستن گوروتین.

#### L53–L54: تعریف متغیرهای آمار
- L53: `var ok, failed int`
  - شمارنده‌های موفق و ناموفق.
- L54: `var totalBytes int64`
  - مجموع بایت‌های دانلود شده.

#### L56–L70: حلقه‌ی خواندن نتایج از Channel
- L56: `log.Printf("[main] waiting for results (selecting on results channel)")`
  - لاگ شروع انتظار برای نتایج.
- L57: `for r := range results`
  - حلقه روی Channel تا بسته شدن آن.
- L58–L63: پردازش خطاها
  - L58: `if r.Err != nil`
    - بررسی وجود خطا در نتیجه.
  - L59: لاگ خطا با جزئیات.
  - L60: `failed++` - افزایش شمارنده‌ی ناموفق.
  - L61–L62: کامنت برای Cancel کردن همه‌ی کارها در صورت خطا.
  - L63: `continue` - ادامه با نتیجه بعدی.
- L64–L69: پردازش نتایج موفق
  - L64: لاگ نتیجه موفق با جزئیات.
  - L65: `ok++` - افزایش شمارنده‌ی موفق.
  - L66: `totalBytes += r.Bytes` - اضافه کردن به مجموع بایت‌ها.

#### L72: لاگ نهایی
- L72: `log.Printf("[main] done. ok=%d failed=%d totalBytes=%d", ok, failed, totalBytes)`
  - چاپ آمار نهایی برنامه.

#### L74–L115: تابع `fetchWithTimeout`
- L74: `func fetchWithTimeout(parent context.Context, url string, perReq time.Duration) Result`
  - تابع دانلود با Context، URL، و Timeout.

- L75–L76: ایجاد Context با Timeout
  - L75: `ctx, cancel := context.WithTimeout(parent, perReq)`
    - Context جدید با Timeout مشخص.
  - L76: `defer cancel()`
    - Cleanup Context در پایان تابع.

- L78: `start := time.Now()`
  - ثبت زمان شروع برای محاسبه مدت زمان.

- L79–L82: ایجاد درخواست HTTP
  - L79: `req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)`
    - درخواست GET با Context.
  - L80–L82: بررسی خطا و بازگشت در صورت وجود.

- L84–L87: تنظیم HTTP Client
  - L84: `client := &http.Client{`
  - L85: `Timeout: 0,`
    - Timeout صفر یعنی استفاده از Context برای Timeout.

- L89: `log.Printf("[fetch] start url=%s (deadline=%s)", url, start.Add(perReq).Format(time.RFC3339Nano))`
  - لاگ شروع دانلود با Deadline.

- L91–L95: اجرای درخواست HTTP
  - L91: `resp, err := client.Do(req)`
    - ارسال درخواست.
  - L92–L95: بررسی خطا و بازگشت در صورت وجود.

- L96: `defer resp.Body.Close()`
  - تضمین بسته شدن Response Body.

- L98–L99: خواندن محتوای Response
  - L98: `n, copyErr := io.Copy(io.Discard, resp.Body)`
    - خواندن کل محتوا بدون ذخیره (فقط برای شمارش بایت‌ها).
  - L99: `elapsed := time.Since(start)`
    - محاسبه زمان سپری‌شده.

- L101–L106: بررسی Cancel شدن Context
  - L101: `select {`
  - L102: `case <-ctx.Done():`
    - اگر Context Cancel شده باشد.
  - L103: `return Result{URL: url, Err: ctx.Err(), Elapsed: elapsed, StatusCode: resp.StatusCode}`
    - بازگشت خطای Context.
  - L104: `default:`
  - L105: `// no problem`
    - اگر Context Cancel نشده باشد، ادامه می‌دهد.

- L108–L111: بررسی خطای کپی
  - L108: `if copyErr != nil`
  - L109: `return Result{URL: url, Err: copyErr, Elapsed: elapsed, StatusCode: resp.StatusCode}`
    - بازگشت خطای کپی.

- L110–L115: بازگشت نتیجه موفق
  - L110: `return Result{`
  - L111: `URL: url,`
  - L112: `StatusCode: resp.StatusCode,`
  - L113: `Bytes: n,`
  - L114: `Elapsed: elapsed,`
  - L115: `}`

---

### چرا Channel؟ چرا Context؟ چرا WaitGroup؟
- **Channel (`results`)**: برای انتقال نتایج از گوروتین‌ها به main به صورت Thread-safe. Channel به صورت خودکار همگام‌سازی می‌کند.
- **Context**: برای مدیریت Timeout و Cancel کردن درخواست‌ها. `context.WithTimeout` تضمین می‌کند درخواست‌ها بیش از زمان مشخص طول نکشند.
- **WaitGroup**: برای شمارش گوروتین‌های فعال و بستن Channel در زمان مناسب.

### نکات مهم همزمانی (Best Practices)
- **Context Propagation**: Context اصلی به همه‌ی گوروتین‌ها پاس می‌شود تا امکان Cancel کردن وجود داشته باشد.
- **Channel Closing**: Channel توسط گوروتین جداگانه‌ای بسته می‌شود که منتظر اتمام همه‌ی کارهاست.
- **Resource Cleanup**: استفاده از `defer` برای Cleanup Context و Response Body.
- **Error Handling**: خطاها در ساختار `Result` ذخیره و به main منتقل می‌شوند.

### اجرای برنامه
```bash
go run P04/main.go
```

نمونه‌ای از خروجی:
```
[main] launching 3 downloads
[fetch] start url=https://mobile.ir (deadline=2024-01-01T12:00:03.000000000Z)
[fetch] start url=https://httpbin.org/delay/2 (deadline=2024-01-01T12:00:03.000000000Z)
[fetch] start url=https://httpbin.org/delay/5 (deadline=2024-01-01T12:00:03.000000000Z)
[main] waiting for results (selecting on results channel)
[result] URL=https://mobile.ir status=200 bytes=12345 elapsed=500ms
[result] URL=https://httpbin.org/delay/2 status=200 bytes=6789 elapsed=2.1s
[result] URL=https://httpbin.org/delay/5 ERR=context deadline exceeded elapsed=3s
[main] done. ok=2 failed=1 totalBytes=19134
```

### تغییرات احتمالی برای آزمایش
- **تغییر Timeout**: مقدار `3*time.Second` را تغییر دهید.
- **اضافه کردن URL**: URLهای جدید به لیست اضافه کنید.
- **Cancel کردن در صورت خطا**: خط L61–L62 را از کامنت خارج کنید.
- **تغییر تعداد Worker**: می‌توانید تعداد گوروتین‌های همزمان را محدود کنید.

---

### جمع‌بندی
این کد نمونه‌ای از الگوی Producer-Consumer با گوروتین‌ها است که از Channel برای انتقال داده، Context برای مدیریت Timeout، و WaitGroup برای همگام‌سازی استفاده می‌کند. این الگو برای دانلود همزمان فایل‌ها، API calls، یا هر عملیات I/O همزمان مناسب است.
