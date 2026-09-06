"""The Iranian car market, as the three sources actually spell it.

Every alias here was taken from captured data, not from a catalogue. Divar
writes English ("Peugeot 206 5"), Bama writes a Persian brand with an English
model slug ("پژو، 206 SD"), Hamrah-Mechanic writes English slugs
("peugeot"/"206"). All three need to land on the same key.

Keys are ASCII slugs so they are safe in URLs, index names and cache keys.
Display names stay Persian because the product is Persian.
"""
from __future__ import annotations

# brand slug -> (Persian display name, aliases seen in the wild)
BRANDS: dict[str, tuple[str, tuple[str, ...]]] = {
    "pride":    ("پراید",      ("pride", "پراید", "saipa pride")),
    "peugeot":  ("پژو",        ("peugeot", "پژو", "پيژو")),
    "samand":   ("سمند",       ("samand", "سمند")),
    "tiba":     ("تیبا",       ("tiba", "تیبا")),
    "quik":     ("کوییک",      ("quik", "quick", "کوییک", "کوئیک")),
    "dena":     ("دنا",        ("dena", "دنا")),
    "tara":     ("تارا",       ("tara", "تارا")),
    "shahin":   ("شاهین",      ("shahin", "شاهین")),
    "rana":     ("رانا",       ("rana", "runna", "رانا")),
    "saina":    ("ساینا",      ("saina", "ساینا")),
    "arisan":   ("آریسان",     ("arisan", "آریسان")),
    "zamyad":   ("زامیاد",     ("zamyad", "زامیاد")),
    "renault":  ("رنو",        ("renault", "رنو", "tondar", "تندر", "l90", "ال ۹۰")),
    "kia":      ("کیا",        ("kia", "کیا")),
    "hyundai":  ("هیوندای",    ("hyundai", "هیوندای", "هیوندا")),
    "toyota":   ("تویوتا",     ("toyota", "تویوتا")),
    "mazda":    ("مزدا",       ("mazda", "مزدا")),
    "nissan":   ("نیسان",      ("nissan", "نیسان")),
    "honda":    ("هوندا",      ("honda", "هوندا")),
    "bmw":      ("بی‌ام‌و",     ("bmw", "ب ام و", "بی ام و", "بی‌ام‌و")),
    "benz":     ("بنز",        ("benz", "mercedes", "بنز")),
    "mvm":      ("ام‌وی‌ام",    ("mvm", "ام وی ام", "ام‌وی‌ام")),
    "chery":    ("چری",        ("chery", "چری", "fownix", "فونیکس", "arizo", "آریزو", "tiggo", "تیگو")),
    "jac":      ("جک",         ("jac", "جک")),
    "kmc":      ("کی‌ام‌سی",    ("kmc", "کی ام سی", "کرمان موتور")),
    "byd":      ("بی‌وای‌دی",   ("byd", "بی وای دی")),
    "mg":       ("ام‌جی",      ("mg", "ام جی")),
    "dongfeng": ("دانگ‌فنگ",   ("dongfeng", "دانگ فنگ", "دانگفنگ")),
    "haima":    ("هایما",      ("haima", "هایما")),
    "suzuki":   ("سوزوکی",     ("suzuki", "سوزوکی")),
    "lexus":    ("لکسوس",      ("lexus", "لکسوس")),
    "geely":    ("جیلی",       ("geely", "جیلی")),
    "foton":    ("فوتون",      ("foton", "فوتون")),
}

# (brand, model slug) -> (Persian display, aliases). Longest alias wins, so
# "206 sd" is matched before "206".
MODELS: dict[tuple[str, str], tuple[str, tuple[str, ...]]] = {
    ("peugeot", "206-sd"): ("۲۰۶ SD", ("206 sd", "206sd", "۲۰۶ اس دی", "206 صندوق")),
    ("peugeot", "206"):    ("۲۰۶", ("206", "۲۰۶")),
    ("peugeot", "207"):    ("۲۰۷", ("207", "۲۰۷")),
    ("peugeot", "405"):    ("۴۰۵", ("405", "۴۰۵")),
    ("peugeot", "pars"):   ("پارس", ("pars", "پارس", "persia")),
    ("peugeot", "2008"):   ("۲۰۰۸", ("2008", "۲۰۰۸")),
    ("peugeot", "roa"):    ("روآ", ("roa", "روآ")),

    ("pride", "111"):      ("۱۱۱", ("111", "۱۱۱")),
    ("pride", "131"):      ("۱۳۱", ("131", "۱۳۱")),
    ("pride", "132"):      ("۱۳۲", ("132", "۱۳۲")),
    ("pride", "141"):      ("۱۴۱", ("141", "۱۴۱")),
    ("pride", "151"):      ("۱۵۱", ("151", "۱۵۱")),
    ("pride", "sedan"):    ("صندوق‌دار", ("sedan", "صندوقدار", "صندوق دار", "صندوق‌دار")),
    ("pride", "hatchback"):("هاچبک", ("hatchback", "هاچبک")),

    ("samand", "soren"):   ("سورن", ("soren", "سورن")),
    ("samand", "lx"):      ("LX", ("lx", "ال ایکس")),
    ("samand", "ef7"):     ("EF7", ("ef7", "ای اف ۷")),

    ("tiba", "2"):         ("تیبا ۲", ("tiba 2", "تیبا 2", "تیبا ۲", "تیبا۲")),
    ("tiba", "1"):         ("تیبا", ("tiba", "تیبا")),

    ("quik", "r"):         ("کوییک R", ("quik r", "کوییک r", "کوییک آر")),
    ("quik", "s"):         ("کوییک S", ("quik s", "کوییک s")),
    ("quik", "base"):      ("کوییک", ("quik", "کوییک")),

    ("dena", "plus"):      ("دنا پلاس", ("dena plus", "دنا پلاس", "دناپلاس")),
    ("dena", "base"):      ("دنا", ("dena", "دنا")),

    ("tara", "v1"):        ("تارا V1", ("tara v1", "تارا v1", "v1")),
    ("tara", "v4"):        ("تارا V4", ("tara v4", "تارا v4", "v4")),
    ("tara", "base"):      ("تارا", ("tara", "تارا")),

    ("shahin", "base"):    ("شاهین", ("shahin", "شاهین")),
    ("rana", "plus"):      ("رانا پلاس", ("rana plus", "رانا پلاس")),
    ("rana", "base"):      ("رانا", ("rana", "runna", "رانا")),
    ("saina", "base"):     ("ساینا", ("saina", "ساینا")),
    ("zamyad", "z24"):     ("زامیاد Z24", ("z 24", "z24", "زد ۲۴")),

    ("renault", "l90"):    ("تندر ۹۰", ("l90", "tondar", "تندر", "ال ۹۰", "logan")),
    ("renault", "sandero"):("ساندرو", ("sandero", "ساندرو")),
    ("renault", "megane"): ("مگان", ("megane", "مگان")),

    ("kia", "cerato"):     ("سراتو", ("cerato", "سراتو")),
    ("kia", "optima"):     ("اپتیما", ("optima", "اپتیما")),
    ("kia", "sportage"):   ("اسپورتیج", ("sportage", "اسپورتیج")),
    ("kia", "rio"):        ("ریو", ("rio", "ریو")),

    ("hyundai", "elantra"):("النترا", ("elantra", "النترا")),
    ("hyundai", "sonata"): ("سوناتا", ("sonata", "سوناتا")),
    ("hyundai", "tucson"): ("توسان", ("tucson", "توسان")),
    ("hyundai", "accent"): ("اکسنت", ("accent", "اکسنت")),
    ("hyundai", "santafe"):("سانتافه", ("santafe", "santa fe", "سانتافه")),

    ("toyota", "corolla-cross"): ("کرولا کراس", ("corolla cross", "کرولا کراس")),
    ("toyota", "corolla"): ("کرولا", ("corolla", "کرولا")),
    ("toyota", "camry"):   ("کمری", ("camry", "کمری")),
    ("toyota", "rav4"):    ("راو۴", ("rav4", "rav 4", "راو 4", "راو۴")),
    ("toyota", "landcruiser"): ("لندکروز", ("landcruiser", "land cruiser", "لندکروز")),
    ("toyota", "prado"):   ("پرادو", ("prado", "پرادو")),
    ("toyota", "yaris"):   ("یاریس", ("yaris", "یاریس")),

    ("mvm", "x22"):        ("X22", ("x22", "x 22")),
    ("mvm", "x33"):        ("X33", ("x33", "x 33")),
    ("mvm", "x55"):        ("X55", ("x55", "x 55")),
    ("mvm", "315"):        ("۳۱۵", ("315", "۳۱۵")),
    ("mvm", "550"):        ("۵۵۰", ("550", "۵۵۰")),

    ("chery", "arrizo5"):  ("آریزو ۵", ("arizo 5", "arrizo 5", "آریزو 5", "آریزو ۵")),
    ("chery", "arrizo6"):  ("آریزو ۶", ("arizo 6", "arrizo 6", "آریزو 6", "آریزو ۶")),
    ("chery", "tiggo7"):   ("تیگو ۷", ("tiggo 7", "تیگو 7", "تیگو ۷")),
    ("chery", "tiggo8"):   ("تیگو ۸", ("tiggo 8", "تیگو 8", "تیگو ۸")),

    ("kmc", "j7"):         ("J7", ("j7", "j 7")),
    ("kmc", "k7"):         ("K7", ("k7", "k 7")),
    ("kmc", "t8"):         ("T8", ("t8", "t 8")),
    ("kmc", "x5"):         ("X5", ("x5", "x 5")),

    ("jac", "s3"):         ("S3", ("s3", "s 3")),
    ("jac", "s5"):         ("S5", ("s5", "s 5")),
    ("jac", "j4"):         ("J4", ("j4", "j 4")),

    ("dongfeng", "h30"):   ("H30", ("h30", "h 30")),
    ("haima", "s5"):       ("S5", ("s5", "s 5")),
    ("haima", "s7"):       ("S7", ("s7", "s 7")),

    ("mazda", "3"):        ("مزدا ۳", ("mazda 3", "مزدا 3", "مزدا۳")),
    ("mazda", "pickup"):   ("وانت مزدا", ("pickup", "کارا", "cara", "وانت")),
    ("nissan", "maxima"):  ("ماکسیما", ("maxima", "ماکسیما")),
    ("nissan", "juke"):    ("جوک", ("juke", "جوک")),
    ("nissan", "qashqai"): ("قشقایی", ("qashqai", "قشقایی")),
}

# Trim aliases are brand-agnostic: «تیپ ۵» means the same on every feed.
#
# Transmission is deliberately NOT here. «تیپ ۵ اتوماتیک» carries two
# independent facts — a model variant and a gearbox — and folding them into one
# field made "اتوماتیک" win over "تیپ ۵" purely because it is a longer string.
# Transmission is normalized separately and enters the spec key on its own.
TRIMS: dict[str, tuple[str, ...]] = {
    "type-1": ("تیپ 1", "تیپ ۱", "type 1", "تیپ1"),
    "type-2": ("تیپ 2", "تیپ ۲", "type 2", "تیپ2"),
    "type-3": ("تیپ 3", "تیپ ۳", "type 3", "تیپ3"),
    "type-5": ("تیپ 5", "تیپ ۵", "type 5", "تیپ5"),
    "type-6": ("تیپ 6", "تیپ ۶", "type 6", "تیپ6"),
    "elx":    ("elx", "ای ال ایکس"),
    "glx":    ("glx",),
    "slx":    ("slx",),
    "lx":     ("lx",),
    "xu7":    ("xu7", "xu 7"),
    "xu7p":   ("xu7p", "xu7 p"),
    "tu5":    ("tu5", "tu 5"),
    "ef7":    ("ef7",),
    "plus":   ("plus", "پلاس"),
    "turbo":  ("turbo", "توربو"),
    "panorama": ("panorama", "پانوراما", "پانورama"),
    "full":   ("فول", "full"),
}
