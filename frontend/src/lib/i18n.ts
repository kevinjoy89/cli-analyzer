// 轻量 i18n：字典由 Go 侧经 GetTranslations 绑定下发（与 CLI/菜单/后端错误
// 同一份语言文件），此处只做取值、插值与复数。零第三方依赖。
export const LOCALES = ['zh-CN', 'zh-TW', 'en'] as const;
export type Locale = (typeof LOCALES)[number];

let locale: Locale = 'zh-CN';
let dict: Record<string, string> = {};

export function setDict(l: Locale, d: Record<string, string>) {
    locale = l;
    dict = d;
}

export function activeLocale(): Locale {
    return locale;
}

// 将 navigator.language 规范化到受支持语言；不支持时回退 zh-CN。
export function normalizeNavigator(lang: string): Locale {
    const l = lang.toLowerCase().replace('_', '-');
    if (l.startsWith('zh-hant') || l.startsWith('zh-tw') || l.startsWith('zh-hk') || l.startsWith('zh-mo')) return 'zh-TW';
    if (l.startsWith('zh')) return 'zh-CN';
    if (l.startsWith('en')) return 'en';
    return 'zh-CN';
}

// t(key, vars)：{name} 占位符插值；缺失键返回键名（便于发现漏翻）；
// 复数规则与 Go 侧一致：键带 _one/_other 且 vars 含 n 时，
// en 按 n==1 选 _one，其余选 _other；zh 恒 _other（两变体同值）。
export function t(key: string, vars?: Record<string, unknown>): string {
    let v = dict[key] ?? key;
    if (vars && vars.n !== undefined && dict[key + '_other'] !== undefined) {
        const n = Number(vars.n);
        if (locale === 'en' && n === 1 && dict[key + '_one'] !== undefined) v = dict[key + '_one'];
        else v = dict[key + '_other'];
    }
    if (!vars) return v;
    return v.replace(/\{(\w+)\}/g, (m, name) => (name in vars ? String(vars[name]) : m));
}

// 将 RFC3339 时间按当前语言区域格式化为紧凑可读形式；非法输入回退原样截断。
export function fmtTime(ts: string): string {
    const d = new Date(ts);
    if (isNaN(d.getTime())) return ts.slice(0, 16).replace('T', ' ');
    try {
        return new Intl.DateTimeFormat(locale, {dateStyle: 'medium', timeStyle: 'short'}).format(d);
    } catch {
        return ts.slice(0, 16).replace('T', ' ');
    }
}
