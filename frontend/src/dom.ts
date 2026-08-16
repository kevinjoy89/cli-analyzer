// DOM 原语：元素获取、HTML 转义、toast。无内部依赖，被所有模块引用。
export function el<T extends HTMLElement>(id: string): T {
    const e = document.getElementById(id);
    if (!e) throw new Error(`missing element #${id}`);
    return e as T;
}

export function esc(s: string): string {
    return s.replace(/[&<>"']/g, c => ({'&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'}[c]!));
}

export function showToast(msg: string, isError = false) {
    const t = el<HTMLDivElement>('toast');
    t.textContent = msg;
    t.className = 'toast' + (isError ? ' error' : '');
    window.clearTimeout((t as unknown as { _t?: number })._t);
    (t as unknown as { _t?: number })._t = window.setTimeout(() => { t.className = 'toast hidden'; }, 2800);
}
