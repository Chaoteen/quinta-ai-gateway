"""Whole-tree review inventory. Findings are hotspots, not proven vulnerabilities.

No matched source or secret value is included in output. Excludes dependencies,
generated builds and local audit tooling. This supplements gosec/govulncheck.
"""
import argparse
import collections
import json
import os
import pathlib
import re

ROOT = pathlib.Path(__file__).resolve().parents[1]
EXCLUDE = {'.git', '.audit-tools', 'node_modules', 'dist', '.gocache', '.gomodcache', '__pycache__'}
RULES = {
    'private-key-candidate': r'-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----',
    'aws-access-key-candidate': r'\b(?:AKIA|ASIA)[A-Z0-9]{16}\b',
    'github-token-candidate': r'\b(?:gh[pousr]_[A-Za-z0-9]{36,}|github_pat_[A-Za-z0-9_]{50,})\b',
    'provider-key-candidate': r'\bsk-(?:proj-|ant-)[A-Za-z0-9_-]{30,}',
    'legacy-row-lock': r'gorm:query_option',
    'tls-verification-bypass': r'InsecureSkipVerify\s*[:=]\s*true',
    'raw-html-sink': r'dangerouslySetInnerHTML|template\.HTML\(',
    'sql-review': r'\.(?:Raw|Exec)\(',
    'balance-write-review': r'Update\("quota"|"quota"\s*:\s*gorm\.Expr',
    'process-execution-review': r'exec\.Command\(',
    'workflow-mutable-action': r'uses:\s*[^\s]+@(?:v\d|main|master)',
    'math-random-review': r'"math/rand"',
}


def scan():
    findings = []
    files = []
    extensions = collections.Counter()
    candidates = []
    for directory, names, filenames in os.walk(ROOT):
        names[:] = sorted(n for n in names if n not in EXCLUDE)
        candidates.extend(pathlib.Path(directory) / n for n in sorted(filenames))
    for path in candidates:
        rel = path.relative_to(ROOT)
        if not path.is_file() or any(p in EXCLUDE for p in rel.parts) or rel.parts[:2] == ('docs', 'security'):
            continue
        if path.stat().st_size > 4_000_000:
            continue
        try:
            text = path.read_text(encoding='utf-8')
        except (UnicodeError, OSError):
            continue
        if '\x00' in text:
            continue
        files.append(rel.as_posix())
        extensions[path.suffix or '(none)'] += 1
        for line_no, line in enumerate(text.splitlines(), 1):
            for rule, expression in RULES.items():
                if rule in {'legacy-row-lock', 'tls-verification-bypass', 'sql-review', 'balance-write-review', 'process-execution-review', 'math-random-review'} and path.suffix != '.go':
                    continue
                if rule == 'raw-html-sink' and path.suffix not in {'.go', '.js', '.jsx', '.ts', '.tsx'}:
                    continue
                if rule == 'workflow-mutable-action' and path.suffix not in {'.yml', '.yaml'}:
                    continue
                if re.search(expression, line):
                    findings.append({'rule': rule, 'path': rel.as_posix(), 'line': line_no,
                                     'test_fixture': bool(re.search(r'(?:_test\.go|\.test\.[jt]sx?)$', path.name))})
    return {'scope': 'current working tree; no Git history; heuristic review inventory',
            'files_scanned': len(files), 'extensions': dict(extensions),
            'counts': dict(collections.Counter(x['rule'] for x in findings)),
            'findings': findings, 'files': files}


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    result = scan()
    output = ROOT / args.output
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(result, indent=2, ensure_ascii=False) + '\n', encoding='utf-8')
    print(json.dumps({'files_scanned': result['files_scanned'], 'counts': result['counts']}))
