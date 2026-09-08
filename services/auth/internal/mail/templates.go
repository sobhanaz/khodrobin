package mail

import (
	"fmt"
	"html/template"
	"strings"
)

// Persian, RTL, and inline-styled — mail clients strip <style> blocks, so every
// rule has to live on the element. The palette is the site's own tokens from
// web/assets/css/main.css: bg #07080b, surface #11141c, ink #f4f6fa, accent
// #ff2e4d, good #3ddc84, dims #a4adbe / #6b7488.
//
// Two things a dark email has to declare or clients will "fix" it for you:
// color-scheme (so Gmail/Apple don't invert the dark card into a light one)
// and a hidden preheader (the sentence that shows in the inbox list next to
// the subject — without one, clients pull a random body line instead).
const shellTmpl = `<!doctype html>
<html lang="fa" dir="rtl">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="dark">
<meta name="supported-color-schemes" content="dark">
<title>خودروبین</title>
</head>
<body style="margin:0;padding:0;background-color:#07080b;color:#f4f6fa;
  font-family:'Vazirmatn','IRANSans',Tahoma,system-ui,sans-serif;
  -webkit-font-smoothing:antialiased;color-scheme:dark">
  <div style="display:none;max-height:0;overflow:hidden;mso-hide:all;font-size:0;
       color:transparent;line-height:0;visibility:hidden">%s</div>
  <div style="padding:32px 16px">
    <div style="max-width:520px;margin:0 auto;background:#11141c;
         border:1px solid rgba(255,255,255,.09);border-radius:16px;
         box-shadow:0 12px 32px rgba(0,0,0,.35)">
      <div style="padding:28px">
        <div style="margin-bottom:22px">
          <span style="display:inline-block;width:28px;height:28px;border-radius:8px;
               background:#ff2e4d;color:#ffffff;font-size:16px;font-weight:800;
               text-align:center;line-height:28px;vertical-align:middle">خ</span>
          <span style="display:inline-block;font-size:19px;font-weight:900;color:#f4f6fa;
               vertical-align:middle;margin-right:9px">خودروبین</span>
          <div style="font-size:12px;color:#6b7488;margin-top:6px">ترب برای خودروی دست‌دوم</div>
          <div style="height:3px;width:44px;background:#ff2e4d;border-radius:2px;margin-top:14px"></div>
        </div>
        %s
        <hr style="border:0;border-top:1px solid rgba(255,255,255,.08);margin:24px 0">
        <div style="font-size:12px;color:#a4adbe;line-height:2">
          اگر این درخواست از طرف شما نبوده، این ایمیل را نادیده بگیرید؛ هیچ تغییری اعمال نمی‌شود.
        </div>
        <div style="font-size:11px;color:#6b7488;margin-top:8px">
          خودروبین &middot; همه‌ی آگهی‌های یک ماشین، در یک ردیف
        </div>
      </div>
    </div>
  </div>
</body>
</html>`

func shell(preheader, body string) string {
	return fmt.Sprintf(shellTmpl, preheader, body)
}

// button renders the primary CTA centred, plus a plain-text fallback under it.
//
// The fallback is not paranoia: some clients render styled links as plain
// text, and a verification link the reader cannot reach is a lost account.
// Both carry the same href, both escaped — a redirect parameter can reach a
// verification URL, so the href is attacker-influenced and treated as such.
func button(href, label string) string {
	h := template.HTMLEscapeString(href)
	return fmt.Sprintf(`<div style="text-align:center;margin-top:6px">
  <a href="%s" target="_blank" rel="noopener" style="display:inline-block;
       background:#ff2e4d;color:#ffffff;text-decoration:none;font-size:14px;
       font-weight:700;padding:12px 26px;border-radius:12px;line-height:1.4">%s</a>
  <div style="margin-top:12px;font-size:12px">
    <a href="%s" style="color:#a4adbe;text-decoration:underline">یا این لینک را در مرورگر باز کن</a>
  </div>
</div>`, h, label, h)
}

// bodyText is the shared paragraph style for the message itself.
const bodyText = `<div style="font-size:14px;line-height:2.1;color:#a4adbe;margin-bottom:8px">%s</div>`

func para(text string) string {
	return fmt.Sprintf(bodyText, text)
}

// heading is the message title. It names the action, not the service: someone
// scanning an inbox has to know what this mail wants from them in one glance.
func heading(text string) string {
	return fmt.Sprintf(`<div style="font-size:17px;font-weight:800;color:#f4f6fa;margin-bottom:12px">%s</div>`, text)
} // Verify is the address-confirmation message.
// It offers both paths on purpose: the button for people who read mail on a
// phone (where a link tap sometimes lands in the client's built-in browser
// instead of our site), and a six-digit code for anyone who would rather type
// it — including the person whose client strips link buttons entirely.
func Verify(link, code string) (string, string) {
	body := shell(
		"برای فعال‌سازی هشدار قیمت و ذخیره‌ی جست‌وجوها، روی دکمه‌ی تأیید بزن یا کد ۶ رقمی را وارد کن — این لینک ۲۴ ساعت اعتبار دارد.",
		strings.Join([]string{
			heading("ایمیلت را تأیید کن"),
			para("برای فعال‌سازی هشدار قیمت و ذخیره‌ی جست‌وجوها، روی دکمه‌ی زیر بزن. " +
				"این لینک ۲۴ ساعت اعتبار دارد و بعد از آن منقضی می‌شود."),
			button(link, "تأیید ایمیل"),
			codePanel(code, link),
		}, ""),
	)
	return "تأیید ایمیل — خودروبین", body
}

// codePanel is the alternative to the button: a six-digit code and where to
// type it. The link is the same verification URL minus its token — landing on
// the token again would just re-verify via the link path.
// The code is digits, so no escaping is needed; the page URL is escaped like
// any other href.
func codePanel(code, link string) string {
	page := link
	if i := strings.IndexByte(page, '?'); i >= 0 {
		page = page[:i]
	}
	href := template.HTMLEscapeString(page)
	return fmt.Sprintf(`<div style="border:1px solid rgba(255,255,255,.08);border-radius:12px;
  background:#161a24;padding:16px;margin-top:20px;text-align:center">
  <div style="font-size:12px;color:#a4adbe;line-height:2">یا این کد را در <a href="%s" style="color:#f4f6fa;text-decoration:underline">صفحه‌ی تأیید</a> وارد کن</div>
  <div style="font-family:ui-monospace,Menlo,monospace;direction:ltr;font-size:30px;
       font-weight:700;color:#f4f6fa;letter-spacing:10px;margin-top:6px">%s</div>
  <div style="font-size:11px;color:#6b7488;line-height:2;margin-top:8px">این کد ۱۵ دقیقه اعتبار دارد.</div>
</div>`, href, code)
}

// Welcome is sent the moment an address becomes verified.
//
// It exists to close the loop on registration: the verify page says "done",
// and this is the same news arriving in the place the person actually reads.
// It also names the two things the account is now good for, because a feature
// nobody is told about might as well not exist.
func Welcome(link string) (string, string) {
	body := shell(
		"حسابت فعال شد — حالا می‌توانی جست‌وجو ذخیره کنی و برای تغییر قیمت‌شان هشدار بگیری.",
		strings.Join([]string{
			heading("به خودروبین خوش آمدی"),
			para("ایمیلت تأیید شد و حسابت فعال است. از این به بعد می‌توانی جست‌وجوهایت را ذخیره کنی " +
				"و برای تغییر قیمت میانه‌شان هشدار بگیری."),
			button(link, "شروع جست‌وجو"),
			`<div style="font-size:11px;color:#6b7488;line-height:2;margin-top:2px">` +
				`هشدار قیمت فقط با ایمیل فرستاده می‌شود — و لغوش هر وقت بخواهی یک کلیک است.</div>`,
		}, ""),
	)
	return "به خودروبین خوش آمدی", body
}

// PasswordChanged is the security notice after a reset.
//
// Most of the time it is read by the person who just reset their own password,
// and it costs them one glance. Its real job is the rare other case: a password
// that changed without the owner doing it is the single best early warning an
// account compromise gives, and the message tells them the two moves that still
// work — reset again, and contact us.
func PasswordChanged(loginLink, forgotLink string) (string, string) {
	body := shell(
		"رمز عبور حساب خودروبین‌ات عوض شد — اگر خودت نبوده‌ای، همین حالا اقدام کن.",
		strings.Join([]string{
			heading("رمز عبورت عوض شد"),
			para("رمز عبور حساب خودروبین‌ات به‌تازگی تغییر کرده. اگر خودت این کار را کردی، کاری لازم نیست."),
			button(loginLink, "ورود به حساب"),
			fmt.Sprintf(`<div style="border:1px solid rgba(255,255,255,.08);border-radius:10px;
  background:#161a24;padding:12px 14px;margin-top:20px;font-size:12px;color:#a4adbe;line-height:2">
  اگر این کار را تو نکرده‌ای، <a href="%s" style="color:#f4f6fa;text-decoration:underline">همین حالا رمزت را بازیابی کن</a>.
</div>`, template.HTMLEscapeString(forgotLink)),
		}, ""),
	)
	return "رمز عبور تغییر کرد — خودروبین", body
}

// SavedSearchCreated confirms that a price alert is armed.
//
// An alert people do not know is on gets deleted as spam when it finally fires.
// Naming the search and the threshold in the confirmation is what makes the
// later PriceAlert mail — which states the same two things — look expected.
func SavedSearchCreated(query string, pct float64, link string) (string, string) {
	body := shell(
		"هشدار قیمت برای جست‌وجوی ذخیره‌شده‌ات فعال شد — وقتی میانه تغییر کند خبرت می‌کنیم.",
		strings.Join([]string{
			heading("هشدار قیمت فعال شد"),
			fmt.Sprintf(`<div style="font-size:14px;line-height:2.1;color:#a4adbe;margin-bottom:18px">`+
				`برای جست‌وجوی «%s» هشدار گذاشتی: اگر قیمت میانه بیش از %s تغییر کند، بهت ایمیل می‌زنیم.</div>`,
				template.HTMLEscapeString(query), pctFmt(pct)),
			button(link, "دیدن نتایج"),
			`<div style="font-size:11px;color:#6b7488;line-height:2;margin-top:2px">` +
				`لغو هشدار یک کلیک است، از صفحه‌ی حساب.</div>`,
		}, ""),
	)
	return "هشدار قیمت فعال شد — خودروبین", body
}

// pctFmt renders the alert threshold the same way the price panel renders
// percentages: Latin digits, because the number sits inside a mono LTR block.
func pctFmt(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}

// Reset is the password-reset message.
func Reset(link string) (string, string) {
	body := shell(
		"برای انتخاب رمز تازه روی دکمه‌ی زیر بزن — این لینک یک ساعت اعتبار دارد و فقط یک بار کار می‌کند.",
		strings.Join([]string{
			heading("تغییر رمز عبور"),
			para("برای انتخاب رمز تازه روی دکمه‌ی زیر بزن. " +
				"این لینک یک ساعت اعتبار دارد و فقط یک بار کار می‌کند."),
			button(link, "انتخاب رمز تازه"),
		}, ""),
	)
	return "تغییر رمز عبور — خودروبین", body
}

// ConfirmSubscription is the double opt-in mail for the newsletter.
//
// It is the only thing an unconfirmed address ever receives. The register page
// promises «هیچ ایمیل تبلیغاتی‌ای نمی‌فرستیم», and this message is what makes
// that promise checkable rather than decorative: without the click there is no
// list entry, so an address typed in by somebody else costs one mail and stops.
//
// The opt-out link rides along even though nobody is subscribed yet, because
// «ignore this» is the wrong answer to give a person who did not ask to be
// here and wants to be sure.
func ConfirmSubscription(confirmLink, unsubscribeLink string) (string, string) {
	body := shell(
		"با یک کلیک عضویتت را در خبرنامه‌ی قیمت تأیید کن — تا آن موقع هیچ ایمیلی نمی‌فرستیم.",
		strings.Join([]string{
			heading("عضویت در خبرنامه‌ی قیمت"),
			para("برای گرفتن خبر تغییر قیمت خودروی دست‌دوم، عضویتت را با دکمه‌ی زیر تأیید کن. " +
				"تا وقتی روی این دکمه نزنی، هیچ ایمیلی برایت نمی‌فرستیم."),
			button(confirmLink, "تأیید عضویت"),
			fmt.Sprintf(`<div style="border:1px solid rgba(255,255,255,.08);border-radius:10px;
  background:#161a24;padding:12px 14px;margin-top:20px;font-size:12px;color:#a4adbe;line-height:2">
  این نشانی را شما وارد نکرده‌اید؟
  <a href="%s" style="color:#f4f6fa;text-decoration:underline">حذف کامل این نشانی</a>.</div>`,
				template.HTMLEscapeString(unsubscribeLink)),
		}, ""),
	)
	return "تأیید عضویت در خبرنامه — خودروبین", body
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
	body := shell(
		"میانه‌ی بازار برای یک جست‌وجوی ذخیره‌شده‌ات تغییر کرد — ببین چقدر.",
		strings.Join([]string{
			fmt.Sprintf(`<div style="font-size:17px;font-weight:800;color:#f4f6fa;margin-bottom:12px">قیمت %s تغییر کرد</div>`,
				template.HTMLEscapeString(carName)),
			fmt.Sprintf(`<div style="font-size:14px;line-height:2.1;color:#a4adbe;margin-bottom:18px">
  میانه‌ی بازار برای جست‌وجوی «%s» %s پیدا کرده است.</div>`,
				template.HTMLEscapeString(query), dir),
			fmt.Sprintf(`<div style="border:1px solid rgba(255,255,255,.08);border-radius:12px;
  background:#161a24;padding:14px 16px;margin-bottom:18px">
  <div style="font-size:11px;color:#6b7488;margin-bottom:6px">قیمت میانه</div>
  <div style="font-family:ui-monospace,Menlo,monospace;direction:ltr;text-align:left;
       font-size:15px;color:#a4adbe">
    %s <span style="color:#6b7488">&rarr;</span>
    <span style="color:%s;font-weight:700">%s</span>
    <span style="color:%s">(%+.1f%%)</span>
  </div>
</div>`, comma(oldMedian), colour, comma(newMedian), colour, pct),
			button(link, "دیدن آگهی‌ها"),
			`<div style="font-size:11px;color:#6b7488;line-height:2;margin-top:2px">
  بر اساس جست‌وجوی ذخیره‌شده‌ات فرستاده شده؛ لغو آن یک کلیک است.
</div>`,
		}, ""),
	)
	return fmt.Sprintf("%s: %s قیمت — خودروبین", carName, dir), body
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
