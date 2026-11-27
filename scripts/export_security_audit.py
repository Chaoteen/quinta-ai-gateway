#!/usr/bin/env python3
"""SQLite ECS deployment adapter: committed audit rows -> collector JSON lines.

No business DB writes, credentials, request bodies or user supplied notes.
Local fsync is NOT an acknowledgement from independent storage.
"""
import argparse
from contextlib import closing
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import time
import uuid

SOURCES = {
    'role_assignment_audits': ('id', 'actor_id', 'target_id', 'tenant_id', 'previous_role', 'new_role', 'created_at'),
    'credential_reset_audits': ('id', 'actor_id', 'target_id', 'target_tenant_id', 'credential_type', 'created_at'),
    'supplier_audits': ('id', 'actor_id', 'action', 'resource_id', 'created_at'),
}


def atomic_json(path, value):
    temp = path.with_suffix('.tmp')
    with temp.open('w', encoding='utf-8') as stream:
        os.chmod(temp, 0o600)
        json.dump(value, stream, sort_keys=True)
        stream.flush()
        os.fsync(stream.fileno())
    os.replace(temp, path)
    if os.name == 'posix':
        fd = os.open(path.parent, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(fd)
        finally:
            os.close(fd)


def recover_partial_line(path):
    # A killed write can leave half a JSON line. It was never checkpointed;
    # replay will emit it again. Preserve complete lines and remove only the tail.
    if not path.exists() or path.stat().st_size == 0:
        return
    with path.open('r+b') as stream:
        size = stream.seek(0, os.SEEK_END)
        stream.seek(max(0, size - 65536))
        start = stream.tell()
        tail = stream.read()
        if tail.endswith(b'\n'):
            return
        last = tail.rfind(b'\n')
        if last < 0 and start:
            raise ValueError('oversized incomplete audit line; operator review required')
        stream.truncate(start + last + 1)
        stream.flush()
        os.fsync(stream.fileno())


def export_once(database, directory, limit=1000):
    directory = Path(directory)
    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    state_path = directory / 'checkpoint.json'
    if state_path.exists():
        state = json.loads(state_path.read_text(encoding='utf-8'))
        if state.get('version') == 1 and set(state.get('cursors', {})) == {'credential_reset_audits', 'supplier_audits'}:
            state['cursors']['role_assignment_audits'] = 0
            atomic_json(state_path, state)
        if state.get('version') != 1 or set(state.get('cursors', {})) != set(SOURCES):
            raise ValueError('invalid checkpoint; operator review required')
        uuid.UUID(state['instance'])
    else:
        state = {'version': 1, 'instance': str(uuid.uuid4()), 'cursors': dict.fromkeys(SOURCES, 0)}
        # Persist identity before output, including when first output fails.
        atomic_json(state_path, state)
    now = int(time.time())
    events = []
    next_cursors = dict(state['cursors'])
    # mode=ro does not create a missing DB; no immutable=1 (unsafe with live WAL).
    with closing(sqlite3.connect(Path(database).resolve().as_uri() + '?mode=ro', uri=True, timeout=5)) as db:
        db.row_factory = sqlite3.Row
        db.execute('PRAGMA query_only = ON')
        db.execute('BEGIN')
        for table, columns in SOURCES.items():
            cursor = state['cursors'][table]
            if type(cursor) is not int or cursor < 0:
                raise ValueError('invalid audit cursor')
            maximum = db.execute(f'SELECT COALESCE(MAX(id), 0) FROM {table}').fetchone()[0]
            if maximum < cursor:
                raise ValueError('audit source rewound; preserve evidence and review restore')
            rows = db.execute(f'SELECT {", ".join(columns)} FROM {table} WHERE id > ? ORDER BY id LIMIT ?', (cursor, limit)).fetchall()
            for row in rows:
                fields = dict(row)
                canonical = json.dumps(fields, sort_keys=True, separators=(',', ':'))
                identity = state['instance'] + ':' + table + ':' + canonical
                events.append({'schema_version': 1, 'event_type': table, 'event_id': hashlib.sha256(identity.encode()).hexdigest(), 'exported_at': now, **fields})
                next_cursors[table] = row['id']
    # Heartbeats distinguish an idle source from a stopped exporter. Downstream
    # SLS must alert on absent heartbeats; this file alone proves no remote receipt.
    events.append({'schema_version': 1, 'event_type': 'exporter_heartbeat', 'instance': state['instance'], 'exported_at': now, 'rows_exported': len(events), 'remote_delivery_verified': False})
    destination = directory / (time.strftime('audit-%Y-%m-%d', time.gmtime(now)) + '.jsonl')
    recover_partial_line(destination)
    with destination.open('a', encoding='utf-8', newline='\n') as stream:
        os.chmod(destination, 0o600)
        for event in events:
            stream.write(json.dumps(event, ensure_ascii=True, separators=(',', ':')) + '\n')
        stream.flush()
        os.fsync(stream.fileno())
    # Crash before this write replays the batch. Deduplicate by event_id in SLS.
    state['cursors'] = next_cursors
    state['last_export_at'] = now
    atomic_json(state_path, state)
    return len(events) - 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--database', required=True)
    parser.add_argument('--directory', required=True)
    args = parser.parse_args()
    # systemd prevents overlapping unit runs; flock also protects manual runs.
    import fcntl
    directory = Path(args.directory)
    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    with (directory / 'export.lock').open('a') as lock:
        os.chmod(lock.name, 0o600)
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        count = export_once(args.database, directory)
    print(json.dumps({'exported_rows': count, 'remote_delivery_verified': False}))


if __name__ == '__main__':
    main()
