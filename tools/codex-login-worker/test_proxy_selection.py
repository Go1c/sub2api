import asyncio
from unittest.mock import AsyncMock, patch
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
