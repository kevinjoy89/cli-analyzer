// 与 Go JSON 契约一致的共享类型，供 main.ts 与 lib/ 纯逻辑模块使用
export interface Binary { name: string; path: string; real: string; size: number }
export interface DataDir { path: string; bytes: number; tier: string; kind: string; root?: string }
export interface SubEntry { id: string; path: string; bytes: number }

export interface Cleanable {
    id: string; tool: string; path: string; bytes: number; tier: string;
    kind: string; keep: string; desc: string;
    sub: SubEntry[];
}

export interface Tool {
    name: string; aliases: string[]; installer: string;
    version: string; updatedAt: string; homepage: string; description: string;
    binaries: Binary[]; dataDirs: DataDir[]; cleanables: Cleanable[];
    footprintBytes: number; cleanableBytes: number; userBytes: number;
}

export interface ScanResult {
    scannedAt: string; scanTimeMs: number; platform: string; goVersion: string;
    tools: Tool[];
    totals: { footprintBytes: number; cleanableBytes: number; userBytes: number };
    roots: Record<string, string[]>; walkErrors: number;
    unattributed?: DataDir[];
}

export interface Point { date: string; footprint: number; cleanable: number; user: number }
export interface Grower { tool: string; deltaBytes: number }
export interface TrendsResult { points: Point[]; topGrowers: Grower[] }
export interface ReminderConfig { thresholdBytes: number }
