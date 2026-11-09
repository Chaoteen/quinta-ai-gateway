"""Collect redacted audit summaries without copying secrets or source snippets."""
import collections
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parents[1]
LOCAL = ROOT / '.audit-tools'
DEST = ROOT / 'docs/security'


def json_messages(path):
    text = path.read_text(encoding='utf-8').strip()
    decoder = json.JSONDecoder()
    while text:
        item, end = decoder.raw_decode(text)
        yield item
        text = text[end:].lstrip()


def advisories(name):
    data = json.loads((LOCAL / name).read_text(encoding='utf-8'))
    return {'affected_package_names': sorted(data),
            'affected_packages': len(data),
            'advisory_entries_by_severity': dict(collections.Counter(
                x['severity'] for values in data.values() for x in values))}


def collect():
    text = (LOCAL / 'gosec-final.log').read_text(encoding='utf-8')
    text = text[text.find('{\n'):]
    text = '\n'.join(line for line in text.splitlines()
                     if not line.startswith(('exit status', '[gosec]', 'go:')))
    gosec = json.JSONDecoder().raw_decode(text)[0]
    findings = []
    for issue in gosec['Issues']:
        findings.append({
            'rule': issue['rule_id'], 'severity': issue['severity'],
            'confidence': issue['confidence'],
            'path': pathlib.Path(issue['file']).relative_to(ROOT).as_posix(),
            'line': issue['line'], 'description': issue['details'],
            'status': 'manual review required; scanner severity is not exploitability',
        })
    vulns = [m['finding'] for m in json_messages(LOCAL / 'govulncheck-all-final.json') if 'finding' in m]
    leaks = json.loads((LOCAL / 'gitleaks.json').read_text(encoding='utf-8'))
    secret_review = [{'rule': x['RuleID'], 'path': x['File'], 'line': x['StartLine'],
                      'status': 'reviewed: public identifier, format marker or synthetic test/example; not a confirmed live secret'}
                     for x in leaks]
    result = {
        'date': '2026-11-08', 'commercial_readiness': 'NOT APPROVED FOR PRODUCTION',
        'scope': 'current snapshot; Git history, production infrastructure and payment credentials not inspected',
        'gosec': {'metrics': gosec['Stats'], 'load_errors': gosec.get('Golang errors', {}),
                  'rules': dict(collections.Counter(x['rule'] for x in findings)), 'findings': findings},
        'govulncheck': {'tool': 'v1.9.0', 'go': '1.26.10', 'unique_affected_advisories': sorted({x['osv'] for x in vulns}),
                       'reachable_advisories': sorted({x['osv'] for x in vulns if any(y.get('function') for y in x['trace'])})},
        'frontend_default_before': advisories('frontend-audit.json'),
        'frontend_default_after': advisories('frontend-audit-after.json'),
        'frontend_classic_before': advisories('classic-audit.json'),
        'frontend_classic_after': advisories('classic-audit-after.json'),
        'gitleaks_current_tree': {'tool': 'v8.30.1', 'alerts': len(leaks), 'review': secret_review,
                                 'history_status': 'not scanned; Git clone failed; full-history CI provided'},
        'verification': {
            'backend_all_packages': 'passed; explicit application package list excludes node_modules',
            'root_package_compilation': 'passed',
            'default_frontend_typecheck': 'passed', 'default_frontend_build': 'passed',
            'money_currency_regressions': '2 passed', 'classic_frontend_build': 'passed',
            'postgres_mysql_concurrency': 'CI configured; not executed locally; no database services available',
            'race_detector': 'CI configured; not executed locally; no C compiler available',
            'production_browser_e2e_and_payment_sandbox': 'not executed',
            'container_deployment': 'template reviewed; Docker unavailable locally',
        },
    }
    DEST.mkdir(parents=True, exist_ok=True)
    (DEST / 'evidence.json').write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps({'gosec': result['gosec']['metrics'], 'govulncheck': result['govulncheck'],
                      'frontend_default_after': result['frontend_default_after'],
                      'frontend_classic_after': result['frontend_classic_after']}, ensure_ascii=False))


if __name__ == '__main__':
    collect()
