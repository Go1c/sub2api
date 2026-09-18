import json
import unittest

from login_core import parse_import, select_workspace, LoginError, validate_tokens


SECRET = 'GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ'


class CodexLoginTests(unittest.TestCase):
    def test_json_and_text_import_preserve_password(self):
        for value in [json.dumps({'accounts': [{'email': 'A@example.com', 'password': ' p ', '2fa_sk': SECRET}]}),
                      f'A@example.com---- p ----{SECRET}']:
            items, errors = parse_import(value)
            self.assertEqual(errors, [])
            self.assertEqual(items[0].email, 'a@example.com')
            self.assertEqual(items[0].password, ' p ')

    def test_partial_import_does_not_echo_secrets(self):
        items, errors = parse_import(json.dumps([
            {'email': 'a@example.com', 'password': 'sensitive-password', 'totp_secret': SECRET},
            {'email': 'b@example.com', 'password': 'sensitive-password', 'totp_secret': 'invalid!'},
        ]))
        self.assertEqual(len(items), 1)
        self.assertEqual(errors[0]['index'], 2)
        self.assertNotIn('sensitive-password', str(errors))
        self.assertNotIn(SECRET, repr(items))

    def test_team_wins_regardless_of_order_and_missing_team_falls_back(self):
        personal = {'id': 'p', 'kind': 'personal'}
        team = {'id': 't', 'kind': 'organization'}
        self.assertEqual(select_workspace([personal, team])['id'], 't')
        self.assertEqual(select_workspace([team, personal])['id'], 't')
        self.assertEqual(select_workspace([personal])['id'], 'p')

    def test_recovery_never_switches_workspace(self):
        with self.assertRaises(LoginError):
            select_workspace([{'id': 'p', 'kind': 'personal'}], 'missing-team')

    def test_ambiguous_team_requires_selection(self):
        with self.assertRaises(LoginError):
            select_workspace([{'id': 'a', 'kind': 'organization'}, {'id': 'b', 'kind': 'organization'}])

    def test_duplicate_email_is_reported_not_silently_overwritten(self):
        text = f'a@example.com----p----{SECRET}\na@example.com----other----{SECRET}'
        items, errors = parse_import(text)
        self.assertEqual(len(items), 1)
        self.assertEqual(errors[0]['code'], 'duplicate_email')

    def test_mismatched_token_identity_rejected(self):
        import jwt
        token = jwt.encode({'email': 'wrong@example.com', 'https://api.openai.com/auth': {'chatgpt_account_id': 't'}}, 'test-key-long-enough-for-this-test')
        with self.assertRaises(LoginError):
            validate_tokens({'access_token': token, 'id_token': token, 'refresh_token': 'rt'}, 'a@example.com', 't')
