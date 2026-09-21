import asyncio
from unittest.mock import AsyncMock, Mock, patch
from types import SimpleNamespace
import pytest
import server
from login_core import LoginError


def test_no_group_proxy_never_starts_login():
    with patch.object(server, 'login', new=AsyncMock()) as login:
        with pytest.raises(LoginError) as error:
            asyncio.run(server.run({'email': 'a@example.com', 'password': 'private', 'totp_secret': 'private'}))
        assert error.value.code == 'proxy_required'
        login.assert_not_awaited()


def test_probe_shuffles_then_skips_dead_proxy_before_account_login():
    candidates = [{'id': 1, 'url': 'http://dead:80'}, {'id': 2, 'url': 'socks5://working:1080'}]
    with patch.object(server, 'probe_proxy', new=AsyncMock(side_effect=[False, True])) as probe, \
         patch.object(server.random, 'shuffle') as shuffle:
        selected = asyncio.run(server.select_proxy(candidates))
        shuffle.assert_called_once()
        assert selected == {'id': 2, 'url': 'socks5h://working:1080'}
        assert probe.await_args_list[0].args == ('http://dead:80',)
        assert probe.await_args_list[1].args == ('socks5h://working:1080',)


def test_no_working_proxy_fails_closed():
    with patch.object(server, 'probe_proxy', new=AsyncMock(return_value=False)), \
         patch.object(server, 'login', new=AsyncMock()) as login:
        with pytest.raises(LoginError) as error:
            asyncio.run(server.run({'proxy_candidates': [{'id': 1, 'url': 'http://dead:80'}]}))
        assert error.value.code == 'proxy_unavailable'
        login.assert_not_awaited()


def test_no_proxy_environment_cannot_bypass_explicit_proxy():
    import os
    import threading
    from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
    received = []
    class ProxyHandler(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass
        def do_GET(self):
            received.append(self.path)
            self.send_response(200)
            self.send_header('Content-Length', '0')
            self.end_headers()
    proxy = ThreadingHTTPServer(('127.0.0.1', 0), ProxyHandler)
    thread = threading.Thread(target=proxy.serve_forever, daemon=True)
    thread.start()
    async def check():
        # Port 1 is intentionally not a target server: only the mock proxy can return 200.
        async with server.proxy_session(f'http://127.0.0.1:{proxy.server_port}', 2) as client:
            response = await client.get('http://127.0.0.1:1/probe', allow_redirects=False)
            assert response.status_code == 200
    try:
        with patch.dict(os.environ, {'NO_PROXY': '*', 'no_proxy': '*'}):
            asyncio.run(check())
        assert received == ['http://127.0.0.1:1/probe']
    finally:
        proxy.shutdown(); proxy.server_close(); thread.join()


def test_login_and_quota_use_exact_selected_proxy_without_rotation():
    from types import SimpleNamespace
    selected = {'id': 7, 'url': 'socks5h://working:1080'}
    quota = AsyncMock()
    quota.get.return_value = SimpleNamespace(status_code=200, json=lambda: {'rate_limit': {}})
    context = AsyncMock()
    context.__aenter__.return_value = quota
    with patch.object(server, 'select_proxy', new=AsyncMock(return_value=selected)), \
         patch.object(server, 'login', new=AsyncMock(return_value={'tokens': {'access_token': 'private'}, 'account_id': 'team'})) as login, \
         patch.object(server, 'proxy_session', return_value=context) as session:
        result = asyncio.run(server.run({'email': 'demo@example.com', 'password': 'private', 'totp_secret': 'private'}))
    assert login.await_args.kwargs['proxy'] == selected['url']
    assert session.call_args.args == (selected['url'], 20)
    assert result['proxy_id'] == 7


def test_child_login_rejects_empty_proxy_before_spawning():
    from login_core import login, LoginMaterial
    with patch('login_core.asyncio.create_subprocess_exec', new=AsyncMock()) as spawn:
        with pytest.raises(LoginError):
            asyncio.run(login(LoginMaterial('demo@example.com', 'private', 'private'), proxy=''))
        spawn.assert_not_awaited()


def test_mfa_rejection_after_probe_keeps_selected_exit_and_never_rotates():
    candidates = [{'id': 22, 'url': 'http://mock:80'}, {'id': 23, 'url': 'http://mock2:80'}]
    with patch.object(server, 'probe_proxy', new=AsyncMock(return_value=True)) as probe, \
         patch.object(server.random, 'shuffle'), \
         patch.object(server, 'login', new=AsyncMock(side_effect=LoginError(
             'mfa_rejected', {'stage': 'totp', 'http_status': 403, 'body': 'SECRET'}))) as login:
        with pytest.raises(LoginError) as error:
            asyncio.run(server.run({'proxy_candidates': candidates, 'email': 'mock@example.com',
                                    'password': 'SECRET', 'totp_secret': 'SECRET'}))
        result = server.failure_response(error.value)
        assert result == {'success': False, 'code': 'mfa_rejected',
                          'diagnostics': {'stage': 'totp', 'http_status': 403, 'proxy_id': 22}}
        assert probe.await_count == login.await_count == 1


def test_probe_budget_reports_tried_count_without_selected_proxy():
    candidates = [{'id': i, 'url': 'http://mock:80'} for i in range(1, 11)]
    with patch.object(server, 'probe_proxy', new=AsyncMock(return_value=False)), \
         patch.object(server, 'time', SimpleNamespace(monotonic=Mock(side_effect=[0, 0, 6, 12, 18, 24, 30]))):
        with pytest.raises(LoginError) as error:
            asyncio.run(server.select_proxy(candidates))
    assert server.failure_response(error.value)['diagnostics'] == {
        'stage': 'proxy_selection', 'candidate_count': 10, 'tried_count': 5}


def test_single_dead_member_reports_one_candidate_without_selection():
    with patch.object(server, 'probe_proxy', new=AsyncMock(return_value=False)):
        with pytest.raises(LoginError) as error:
            asyncio.run(server.select_proxy([{'id': 24, 'url': 'http://mock:80'}]))
    assert server.failure_response(error.value) == {'success': False, 'code': 'proxy_unavailable',
        'diagnostics': {'stage': 'proxy_selection', 'candidate_count': 1, 'tried_count': 1}}


@pytest.mark.parametrize('status,reachable', [(200, True), (302, False), (403, False)])
def test_probe_does_not_treat_unvalidated_redirect_as_login_reachability(status, reachable):
    from types import SimpleNamespace
    client = AsyncMock()
    client.get.return_value = SimpleNamespace(status_code=status, headers={'location': '/challenge?SECRET'})
    context = AsyncMock()
    context.__aenter__.return_value = client
    with patch.object(server, 'proxy_session', return_value=context):
        assert asyncio.run(server.probe_proxy('http://mock:80')) is reachable
    client.get.assert_awaited_once_with('https://auth.openai.com/log-in', allow_redirects=False)


def test_all_invalid_candidates_are_configuration_failure_not_unavailable():
    with patch.object(server, 'probe_proxy', new=AsyncMock()) as probe:
        with pytest.raises(LoginError) as error:
            asyncio.run(server.select_proxy([{'id': 24, 'url': ''}]))
    assert error.value.code == 'proxy_required'
    probe.assert_not_awaited()


def test_unexpected_failure_after_selection_keeps_proxy_without_secret():
    import json
    with patch.object(server, 'select_proxy', new=AsyncMock(return_value={'id': 22, 'url': 'http://mock:80'})), \
         patch.object(server, 'login', new=AsyncMock(side_effect=RuntimeError('SECRET'))):
        with pytest.raises(LoginError) as error:
            asyncio.run(server.run({'email': 'mock@example.com', 'password': 'SECRET', 'totp_secret': 'SECRET'}))
    result = server.failure_response(error.value)
    assert result['code'] == 'login_failed'
    assert result['diagnostics'] == {'proxy_id': 22, 'exception_type': 'RuntimeError'}
    assert 'SECRET' not in json.dumps(result)
