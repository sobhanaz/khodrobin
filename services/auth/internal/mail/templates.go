package mail

import (
	"fmt"
	"html/template"
	"strings"
)

// Persian, RTL, and inline-styled — mail clients strip <style> blocks, so every
// rule has to live on the element.
const shell = `<!doctype html>
<html lang="fa" dir="rtl"><body style="margin:0;background:#0b0d12;padding:32px 16px;
  font-family:Tahoma,'IRANSans',system-ui,sans-serif;color:#f4f6fa">
  <div style="max-width:520px;margin:0 auto;background:#11141c;border:1px solid rgba(255,255,255,.08);
              border-radius:16px;padding:28px">
    <div style="font-size:20px;font-weight:900;margin-bottom:4px">خودروبین</div>
    <div style="font-size:12px;color:#6b7488;margin-bottom:20px">ترب برای خودروی دست‌دوم</div>
    %s
    <hr style="border:0;border-top:1px solid rgba(255,255,255,.08);margin:24px 0">
    <div style="font-size:11px;color:#6b7488;line-height:1.9">
      اگر این درخواست از طرف شما نبوده، این ایمیل را نادیده بگیرید؛ هیچ تغییری اعمال نمی‌شود.
    </div>
  </div>
</body></html>`

func button(href, label string) string {
	return fmt.Sprintf(`<a href="%s" style="display:inline-block;background:#ff2e4d;color:#fff;
	  text-decoration:none;font-weight:700;padding:12px 22px;border-radius:10px">%s</a>`,
		template.HTMLEscapeString(href), label)
}

// Verify is the address-confirmation message.
func Verify(link string) (string, string) {
	body := fmt.Sprintf(shell, strings.Join([]string{
		`<div style="font-size:16px;font-weight:700;margin-bottom:10px">ایمیلت را تأیید کن</div>`,
		`<div style="font-size:14px;line-height:2;color:#a4adbe;margin-bottom:20px">`,
		`برای فعال‌سازی هشدار قیمت و ذخیره‌ی جست‌وجوها، روی دکمه‌ی زیر بزن. این لینک ۲۴ ساعت اعتبار دارد.`,
		`</div>`,
		button(link, "تأیید ایمیل"),
	}, ""))
	return "تأیید ایمیل — خودروبین", body
}

// Reset is the password-reset message.
func Reset(link string) (string, string) {
	body := fmt.Sprintf(shell, strings.Join([]string{
		`<div style="font-size:16px;font-weight:700;margin-bottom:10px">تغییر رمز عبور</div>`,
		`<div style="font-size:14px;line-height:2;color:#a4adbe;margin-bottom:20px">`,
		`برای انتخاب رمز تازه روی دکمه بزن. این لینک یک ساعت اعتبار دارد و فقط یک بار کار می‌کند.`,
		`</div>`,
		button(link, "انتخاب رمز تازه"),
	}, ""))
	return "تغییر رمز عبور — خودروبین", body
}

// PriceAlert is the message a saved search produces when a median moves.
//
// The reason the mail exists is stated in it — which car, which direction, how
// much — because an alert that does not say why it fired trains people to
// ignore alerts.
func PriceAlert(carName, query string, oldMedian, newMedian int64, link string) (string, string) {
	dir, colour := "کاهش", "#3ddc84"
	if newMedian > oldMedian {
		dir, colour = "افزایش", "#ff2e4d"
	}
	pct := 0.0
	if oldMedian != 0 {
		pct = float64(newMedian-oldMedian) / float64(oldMedian) * 100
	}
	body := fmt.Sprintf(shell, strings.Join([]string{
		fmt.Sprintf(`<div style="font-size:16px;font-weight:700;margin-bottom:10px">قیمت %s تغییر کرد</div>`,
			template.HTMLEscapeString(carName)),
		fmt.Sprintf(`<div style="font-size:14px;line-height:2;color:#a4adbe;margin-bottom:16px">`+
			`میانه‌ی بازار برای جست‌وجوی «%s» %s پیدا کرد.</div>`,
			template.HTMLEscapeString(query), dir),
		fmt.Sprintf(`<div style="font-family:monospace;direction:ltr;text-align:left;font-size:15px;`+
			`background:#161a24;border-radius:10px;padding:14px;margin-bottom:18px">`+
			`%s &rarr; <span style="color:%s;font-weight:700">%s</span> `+
			`<span style="color:%s">(%+.1f%%)</span></div>`,
			comma(oldMedian), colour, comma(newMedian), colour, pct),
		button(link, "دیدن آگهی‌ها"),
	}, ""))
	return fmt.Sprintf("%s %s قیمت — خودروبین", carName, dir), body
}

// comma groups thousands. Prices are long, and an unbroken run of ten digits is
// unreadable at a glance.
func comma(n int64) string {
	s := fmt.Sprintf("%d", n)
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}
