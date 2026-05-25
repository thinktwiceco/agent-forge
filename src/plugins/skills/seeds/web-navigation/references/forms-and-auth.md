# Forms And Auth

Inspect before filling. Start with `get_snapshot`, then use `interactive_tree` if you need a compact control list.

Prefer semantic selectors over generated classes:

- `input[type='email']`
- `input[name='email']`
- `input[autocomplete='username']`
- `input[type='password']`
- `button[type='submit']`
- `input[type='file']`

Use `fill` for normal values such as email, username, search text, or codes.

Use `fill_secret` for passwords and other secrets. Pass the vault key name in `resolveSecretVaultKey`, not the plaintext secret.

If the flow is multi-step, inspect after every submit or continue action before filling the next field.

Use `upload_file` for file inputs. It sets the file directly on the browser element and does not rely on the OS file picker.

Example:

```json
{"action":"get_snapshot","session":"login","settle_ms":1500}
{"action":"fill","session":"login","selector":"input[type='email']","value":"user@example.com"}
{"action":"click","session":"login","selector":"button[type='submit']"}
{"action":"get_snapshot","session":"login","settle_ms":1500}
{"action":"fill_secret","session":"login","selector":"input[type='password']","resolveSecretVaultKey":"service-password"}
{"action":"click","session":"login","selector":"button[type='submit']","timeout":60}
```
