"""Export implementation changes from a connector snapshot, without private reports."""
import difflib
import hashlib
import json
import os
import pathlib

ROOT = pathlib.Path(__file__).resolve().parents[1]
EXCLUDE = {'.git', '.audit-tools', 'node_modules', 'dist', '__pycache__', 'artifacts'}


def blob_hash(data):
    return hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()


def export():
    baseline = json.loads((ROOT / 'docs/security/source-snapshot.json').read_text(encoding='utf-8'))
    entries = {x['path']: x for x in baseline['files']}
    originals = json.loads((ROOT / '.audit-tools/original-source.json').read_text(encoding='utf-8'))
    check_root = ROOT / '.audit-tools/patch-check'
    changes = []
    patch = []
    candidates = []
    for directory, names, filenames in os.walk(ROOT):
        names[:] = [x for x in names if x not in EXCLUDE]
        for name in filenames:
            path = pathlib.Path(directory) / name
            relative = path.relative_to(ROOT).as_posix()
            if relative.startswith(('docs/security/', '.agents/', '.codex/')):
                continue
            candidates.append((relative, path))
    for relative, path in sorted(candidates):
        data = path.read_bytes()
        if relative in entries and blob_hash(data) == entries[relative]['blob_sha']:
            continue
        if relative in entries and relative not in originals:
            raise RuntimeError('Original source missing for ' + relative)
        before = originals.get(relative, '')
        after = data.decode('utf-8')
        check_path = check_root / relative
        check_path.parent.mkdir(parents=True, exist_ok=True)
        if relative in entries:
            check_path.write_bytes(before.encode('utf-8'))
        patch.append('diff --git a/' + relative + ' b/' + relative + '\n')
        if relative not in entries:
            patch.append('new file mode 100644\n')
        delta = difflib.unified_diff(
            before.splitlines(keepends=True), after.splitlines(keepends=True),
            fromfile='a/' + relative if relative in entries else '/dev/null', tofile='b/' + relative)
        for line in delta:
            if line.endswith('\n'):
                patch.append(line)
            else:
                patch.append(line + '\n\\ No newline at end of file\n')
        changes.append({'path': relative, 'new': relative not in entries,
                        'sha256': hashlib.sha256(data).hexdigest()})
    destination = ROOT / 'artifacts'
    destination.mkdir(exist_ok=True)
    (destination / 'security-hardening.patch').write_text(''.join(patch), encoding='utf-8', newline='\n')
    (ROOT / 'docs/security/change-manifest.json').write_text(
        json.dumps({'base_commit': baseline['commit'], 'files': changes}, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({'changed_files': len(changes), 'patch_bytes': (destination / 'security-hardening.patch').stat().st_size}))


if __name__ == '__main__':
    export()
