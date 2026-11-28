"""Browser acceptance using synthetic users and fully mocked APIs; no production calls."""
import sys, json, argparse, threading, atexit
from http.server import ThreadingHTTPServer, SimpleHTTPRequestHandler
from pathlib import Path
from urllib.parse import urlsplit
repo = Path(__file__).resolve().parents[1]
local_dependencies = repo / '.audit-tools/browserdeps'
if local_dependencies.is_dir():
    sys.path.insert(0, str(local_dependencies))
parser = argparse.ArgumentParser(description='Offline mocked RBAC browser acceptance; build web/default first.')
parser.add_argument('--browser-executable', default=None)
parser.add_argument('--artifact-dir', type=Path, default=repo / '.audit-tools/rbac-ui')
args = parser.parse_args()
args.artifact_dir.mkdir(parents=True, exist_ok=True)
dist = (repo / 'web/default/dist').resolve()
if not (dist / 'index.html').is_file():
    raise SystemExit('Build web/default with VITE_BASE_PATH=/quinta first')

class Handler(SimpleHTTPRequestHandler):

    def translate_path(self, path):
        path = urlsplit(path).path.removeprefix('/quinta').lstrip('/')
        target = (dist / path).resolve()
        if not target.is_relative_to(dist):
            return str(dist / 'index.html')
        return str(target if target.is_file() else dist / 'index.html')

    def log_message(self, *args):
        pass
server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
threading.Thread(target=server.serve_forever, daemon=True).start()
atexit.register(server.shutdown)
origin = 'http://127.0.0.1:' + str(server.server_port)
from playwright.sync_api import sync_playwright, expect
status = {'success': True, 'data': {'version': 'rbac-fixture', 'system_name': 'Quinta fixture', 'quota_per_unit': 500000, 'usd_exchange_rate': 7.2, 'display_in_currency_enabled': True, 'is_demo_site': False, 'chat_link': []}}
actions = ['view', 'use', 'create', 'operate', 'update', 'delete']

def grant(scope='tenant', **kw):
    return {'scope': scope, **dict.fromkeys(actions, False), **kw}
caps = {r: grant(**dict.fromkeys(actions, True)) for r in ['keys', 'invoice_requests', 'payments', 'vouchers', 'members']}
caps['model_calls'] = grant('self', view=True, use=True)
state = {'policy': {'family': 'enterprise_admin', 'version': 0, 'delegate': False, 'ceiling': caps, 'grants': caps}, 'roles': [], 'verified': False, 'assigned': False, 'saved': False}
requests = []
errors = []

def run_context(browser, kind):
    root = kind == 'root'
    primary = kind == 'primary'
    permissions = caps if primary or root else {k: grant(v['scope'], view=True) for k, v in caps.items() if k != 'members'}
    user = {'id': 1 if root else 2 if primary else 3, 'username': 'fixture-' + kind, 'role': 100 if root else 10, 'role_key': 'root' if root else 'enterprise_admin', 'status': 1, 'tenant_id': 1 if root else 2, 'group': 'default', 'quota': 10000, 'setting': '{}', 'resource_permissions': permissions, 'can_delegate_roles': primary or root, 'permissions': {'sidebar_settings': False}}
    context = browser.new_context(locale='zh-CN', viewport={'width': 1360, 'height': 900})
    context.add_init_script("localStorage.setItem('user'," + json.dumps(json.dumps(user)) + ");localStorage.setItem('uid'," + json.dumps(str(user['id'])) + ");localStorage.setItem('i18nextLng','zh');")
    page = context.new_page()
    page.on('pageerror', lambda e: errors.append(str(e)))
    page.on('console', lambda message: errors.append(message.text) if message.type == 'error' and ('same key' in message.text or 'React' in message.text) else None)

    def api(route):
        path = urlsplit(route.request.url).path.removeprefix('/quinta')
        method = route.request.method
        requests.append((kind, method, path))
        own = {'family': user['role_key'], 'tenant_id': user['tenant_id'], 'role_id': 0 if primary or root else 7, 'can_delegate': root or (primary and (not state.get('revoked', False))), 'grants': permissions}
        if path == '/api/rbac/workspace':
            result = {'state': own, 'policies': [state['policy']], 'roles': state['roles'], 'audits': [], 'truncated': False}
        elif path == '/api/rbac/self':
            result = own
        elif path == '/api/verify':
            state['verified'] = True
            result = {'verified': True}
        elif path == '/api/user/2fa/status':
            result = {'enabled': True}
        elif path.startswith('/api/user/passkey'):
            result = {'enabled': False}
        elif path in ['/api/rbac/policy', '/api/rbac/custom-roles', '/api/rbac/assignments']:
            if not state['verified']:
                route.fulfill(status=403, json={'success': False, 'code': 'VERIFICATION_REQUIRED', 'message': 'Verification required'})
                return
            data = route.request.post_data_json
            if path.endswith('policy'):
                assert root and data['delegate'] and (data['version'] == 0)
                state['policy'] = {**state['policy'], **data, 'version': 1}
                state['saved'] = True
                result = state['policy']
            elif path.endswith('custom-roles'):
                assert primary and data['tenant_id'] == 2 and (data['name'] == '企业只读财务')
                assert data['grants']['invoice_requests']['view'] and (not data['grants']['invoice_requests']['create'])
                state['roles'] = [{**data, 'id': 7, 'version': 1}]
                result = state['roles'][0]
            else:
                assert primary and data == {'user_id': 3, 'role_id': 7}
                state['assigned'] = True
                result = None
        elif path == '/api/status':
            route.fulfill(json=status)
            return
        elif path == '/api/setup':
            result = {'status': True, 'root_init': True}
        elif path == '/api/user/self':
            result = user
        elif path == '/api/notice':
            result = ''
        elif path == '/api/invoices/capabilities':
            result = {'enabled': False, 'provider': 'manual', 'supported_types': ['normal']}
        elif path.rstrip('/') == '/api/token':
            result = {'items': [{'id': 23, 'name': 'fixture-readonly-key', 'key': 'test********masked', 'status': 1, 'remain_quota': 1000, 'used_quota': 0, 'unlimited_quota': False, 'expired_time': -1, 'created_time': 1700000000, 'accessed_time': 1700000000, 'group': 'default', 'model_limits_enabled': False, 'model_limits': '', 'allow_ips': ''}], 'total': 1, 'page': 1, 'page_size': 10}
        elif path.startswith('/api/invoices/') or path == '/api/payment/orders' or path.startswith('/api/vouchers/') or (path.rstrip('/') == '/api/token'):
            result = {'items': [], 'total': 0, 'page': 1, 'page_size': 10}
        elif path == '/api/user/self/groups':
            result = {'default': {'desc': 'default', 'ratio': 1}}
        else:
            result = []
        route.fulfill(json={'success': True, 'data': result})
    page.route('**/api/**', api)
    return (context, page)

def verify(page):
    expect(page.get_by_role('dialog')).to_be_visible()
    page.get_by_role('dialog').locator('input').first.fill('123456')
    page.get_by_role('dialog').get_by_role('button', name='验证', exact=True).click()
    expect(page.get_by_role('dialog')).to_have_count(0)
with sync_playwright() as p:
    browser = p.chromium.launch(executable_path=args.browser_executable, headless=True)
    context, page = run_context(browser, 'root')
    page.goto(origin + '/quinta/role-permissions')
    page.wait_for_load_state('networkidle')
    expect(page.get_by_role('heading', name='角色与权限')).to_be_visible()
    page.get_by_label('允许主管理员创建角色并为本租户成员分配角色').check()
    page.get_by_role('button', name='验证并保存平台授权').click()
    expect(page.get_by_text('角色配置已保存', exact=True)).to_have_count(0)
    verify(page)
    expect(page.get_by_text('角色配置已保存', exact=True)).to_be_visible()
    assert state['saved']
    page.set_viewport_size({'width': 390, 'height': 844})
    page.wait_for_timeout(100)
    assert page.evaluate('document.documentElement.scrollWidth <= window.innerWidth')
    page.screenshot(path=str(args.artifact_dir / 'delegated-root-mobile.png'))
    context.close()
    state['verified'] = False
    context, page = run_context(browser, 'primary')
    page.goto(origin + '/quinta/role-permissions')
    page.wait_for_load_state('networkidle')
    expect(page.get_by_role('button', name='验证并保存平台授权')).to_have_count(0)
    page.get_by_label('角色名称').fill('企业只读财务')
    page.get_by_label('发票申请 / 可见', exact=True).check()
    page.get_by_role('button', name='验证并保存角色').click()
    expect(page.get_by_text('角色配置已保存', exact=True)).to_have_count(0)
    verify(page)
    expect(page.get_by_text('角色配置已保存', exact=True)).to_be_visible()
    page.get_by_label('用户 ID', exact=True).fill('3')
    page.get_by_label('租户角色', exact=True).select_option('7')
    page.get_by_role('button', name='验证并分配角色').click()
    page.wait_for_timeout(150)
    assert state['assigned']
    page.screenshot(path=str(args.artifact_dir / 'delegated-primary.png'))
    state['revoked'] = True
    page.evaluate("Object.defineProperty(document, 'visibilityState', {configurable:true,value:'hidden'}); window.dispatchEvent(new Event('visibilitychange')); Object.defineProperty(document, 'visibilityState', {configurable:true,value:'visible'}); window.dispatchEvent(new Event('visibilitychange'))")
    expect(page.get_by_role('button', name='验证并保存角色')).to_have_count(0, timeout=10000)
    expect(page.get_by_role('button', name='验证并分配角色')).to_have_count(0)
    context.close()
    context, page = run_context(browser, 'member')
    page.goto(origin + '/quinta/dashboard')
    page.wait_for_load_state('networkidle')
    expect(page.get_by_role('link', name='角色与权限', exact=True)).to_have_count(0)
    for url, missing in [('/keys', '添加 API 密钥'), ('/vouchers', '兑换'), ('/invoices', '提交申请'), ('/wallet', '添加资金')]:
        page.goto(origin + '/quinta' + url)
        page.wait_for_load_state('networkidle')
        expect(page.get_by_role('button', name=missing, exact=True)).to_have_count(0)
        if url == '/keys':
            expect(page.get_by_text('fixture-readonly-key', exact=True)).to_be_visible()
            expect(page.get_by_role('button', name='编辑', exact=True)).to_have_count(0)
            expect(page.get_by_role('button', name='删除', exact=True)).to_have_count(0)
            expect(page.get_by_role('button', name='sk-test********masked', exact=True)).to_have_count(0)
        assert '/403' not in page.url, page.url
    page.screenshot(path=str(args.artifact_dir / 'delegated-member-wallet.png'))
    context.close()
    assert not [(k, m, path) for k, m, path in requests if k == 'member' and (m != 'GET' or path in ['/api/user/topup/info', '/api/user/amount', '/api/log/stat', '/api/user/self/chat-key'])]
    assert not errors, errors
    browser.close()
    print(json.dumps({'platform_policy_verified': True, 'tenant_role_verified': True, 'assignment': True, 'delegation_withdrawal_refresh': True, 'no_premature_success': True, 'read_only_pages': 4, 'mobile': True, 'page_errors': errors}))
