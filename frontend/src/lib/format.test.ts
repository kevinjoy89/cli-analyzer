import {describe, expect, it} from 'vitest';
import {hb, fmtTime} from './format';

describe('hb 字节格式化', () => {
    it('小于 1024 显示为 B', () => {
        expect(hb(0)).toBe('0 B');
        expect(hb(500)).toBe('500 B');
        expect(hb(1023)).toBe('1023 B');
    });
    it('KB/MB/GB 进位', () => {
        expect(hb(1024)).toBe('1.0 KB');
        expect(hb(1536)).toBe('1.5 KB');
        expect(hb(1024 * 1024)).toBe('1.0 MB');
        expect(hb(1024 * 1024 * 1024)).toBe('1.0 GB');
        expect(hb(5 * 1024 * 1024 * 1024)).toBe('5.0 GB');
    });
});

describe('fmtTime 时间格式化', () => {
    it('RFC3339 压缩为 YYYY-MM-DD HH:MM', () => {
        expect(fmtTime('2026-08-12T01:41:08+08:00')).toBe('2026-08-12 01:41');
    });
});

describe('hb 超大值（PB/EB，与 Go 端 humanBytes 后缀一致）', () => {
    it('T 以上继续进位', () => {
        expect(hb(1024 * 1024 * 1024 * 1024)).toBe('1.0 TB');
        expect(hb(1125899906842624)).toBe('1.0 PB'); // 2^50
        expect(hb(1152921504606846976)).toBe('1.0 EB'); // 2^60
    });
});
