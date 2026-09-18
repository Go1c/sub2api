import json
from pathlib import Path
import sys
from types import SimpleNamespace
from unittest.mock import patch


def test_node_cookie_is_not_in_process_arguments():
    sys.path.insert(0, str(Path(__file__).parent / 'third_party/codex_protocol'))
    from core.sentinel_runner import generate_sentinel_token
    token = json.dumps({'p': 'p', 'c': 'c', 'id': 'device', 'flow': 'mfa_verify'})
    with patch('core.sentinel_runner.subprocess.run', return_value=SimpleNamespace(returncode=0, stdout=token, stderr='')) as run:
        generate_sentinel_token({}, 'mfa_verify', 'device', cookie='login_session=private-session-cookie')
    assert 'private-session-cookie' not in ' '.join(run.call_args.args[0])
    assert run.call_args.kwargs['input'] == 'login_session=private-session-cookie'
