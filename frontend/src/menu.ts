// Windows 自绘菜单条模块（macOS/Linux 用原生菜单，不渲染此条）。
// 依赖 flows（openPrefs/manualCheck/openAbout）+ dom。
import {manualCheck, openAbout, openPrefs} from './flows';
import {OpenURL} from '../wailsjs/go/gui/ScannerService';
import {Quit} from '../wailsjs/runtime/runtime';

// 关闭所有打开的下拉菜单
function closeMenus() {
    document.querySelectorAll('.menu-btn.open').forEach(b => b.classList.remove('open'));
    document.querySelectorAll('.menu-pop.open').forEach(p => p.classList.remove('open'));
}

// 初始化 Windows 自绘菜单条：文件/帮助下拉 + 动作分发。
// 用 getElementById + 空值守卫（el() 缺失即 throw，会炸掉整条 init 链）：
// 元素不存在时静默跳过（macOS/Linux 该条隐藏但仍在 DOM；防御未来重构）。
export function initMenuBar() {
    const bar = document.getElementById('menuBar') as HTMLElement | null;
    if (!bar) return;
    bar.querySelectorAll<HTMLButtonElement>('.menu-btn').forEach(btn => {
        btn.onclick = (e) => {
            e.stopPropagation();
            const wasOpen = btn.classList.contains('open');
            closeMenus();
            if (!wasOpen) {
                btn.classList.add('open');
                const pop = bar.querySelector<HTMLElement>(`.menu-pop[data-pop="${btn.dataset.pop}"]`);
                if (pop) pop.classList.add('open');
            }
        };
    });
    bar.querySelectorAll<HTMLButtonElement>('.menu-opt').forEach(opt => {
        opt.onclick = () => {
            closeMenus();
            switch (opt.dataset.act) {
                case 'prefs': openPrefs(); break;
                case 'quit': Quit(); break;
                case 'about': openAbout(); break;
                case 'check-updates': manualCheck(); break;
                case 'github': OpenURL('https://github.com/kevinjoy89/cli-analyzer'); break;
                case 'issue': OpenURL('https://github.com/kevinjoy89/cli-analyzer/issues/new'); break;
            }
        };
    });
    document.addEventListener('click', closeMenus);
}
