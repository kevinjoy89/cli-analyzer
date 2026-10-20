#!/usr/bin/env python3
"""测量「按路径求和」与「按 inode 去重」的磁盘占用差值。

背景：internal/disk 用 stat 的物理块数（st_blocks）逐文件累加，硬链接与
APFS clonefile 产生的共享块会被重复计入。本脚本对同一批目录同时给出两个数，
用来判断这个偏差在实际机器上是否值得修复（见 ADR 0003 / Q7）。

用法：
    python3 scripts/measure-dedup.py <dir> [<dir> ...]
    python3 scripts/measure-dedup.py --json <dir> ...

输出：每个目录一行，含 path_sum / dedup_sum / 差值 / 重复块占比，
以及硬链接与「疑似 clone」的计数。--json 输出机器可读结果。
"""

import json
import os
import sys

BLOCK = 512  # st_blocks 的单位固定是 512 字节


def measure(root):
    """遍历 root，返回按路径求和与按 inode 去重的字节数。"""
    path_sum = 0          # 逐文件累加（当前 cli-analyzer 的口径）
    seen = {}             # (st_dev, st_ino) -> st_blocks
    hardlink_hits = 0     # 重复出现的 (dev, ino) 次数
    files = 0
    errors = 0

    stack = [root]
    while stack:
        cur = stack.pop()
        try:
            with os.scandir(cur) as it:
                entries = list(it)
        except OSError:
            errors += 1
            continue
        for e in entries:
            try:
                st = e.stat(follow_symlinks=False)
            except OSError:
                errors += 1
                continue
            # 符号链接不跟随，按链接自身大小计（与 internal/disk 一致）
            if e.is_dir(follow_symlinks=False):
                stack.append(e.path)
                continue
            if not (e.is_file(follow_symlinks=False) or e.is_symlink()):
                continue
            blocks = getattr(st, "st_blocks", 0) * BLOCK
            files += 1
            path_sum += blocks
            key = (st.st_dev, st.st_ino)
            if key in seen:
                hardlink_hits += 1
            else:
                seen[key] = blocks

    dedup_sum = sum(seen.values())
    return {
        "root": root,
        "files": files,
        "pathSum": path_sum,
        "dedupSum": dedup_sum,
        "savedBytes": path_sum - dedup_sum,
        "dupRatio": (path_sum - dedup_sum) / path_sum if path_sum else 0.0,
        "dupInodes": hardlink_hits,
        "errors": errors,
    }


def human(n):
    for unit in ("B", "K", "M", "G", "T"):
        if abs(n) < 1024 or unit == "T":
            return f"{n:.1f}{unit}" if unit != "B" else f"{n}B"
        n /= 1024.0


def main(argv):
    as_json = "--json" in argv
    dirs = [a for a in argv if not a.startswith("--")]
    if not dirs:
        print(__doc__.strip())
        return 2
    results = []
    for d in dirs:
        d = os.path.expanduser(d)
        if not os.path.isdir(d):
            print(f"skip (not a dir): {d}", file=sys.stderr)
            continue
        r = measure(d)
        results.append(r)
        if not as_json:
            print(
                f"{r['root']}\n"
                f"  files={r['files']}  errors={r['errors']}\n"
                f"  path-sum  = {human(r['pathSum'])}\n"
                f"  inode-dedup = {human(r['dedupSum'])}\n"
                f"  saved     = {human(r['savedBytes'])} ({r['dupRatio']*100:.2f}%)  "
                f"dup-inodes={r['dupInodes']}"
            )
    if as_json:
        print(json.dumps(results, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
