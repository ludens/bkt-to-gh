# Encrypted Config Design

## Goal

Store migration configuration in encrypted `config.yaml` under the OS user config directory.

## Storage

The CLI uses `os.UserConfigDir()` and stores config at:

- Linux: `$XDG_CONFIG_HOME/bkt2gh/config.yaml`, or `~/.config/bkt2gh/config.yaml`
- macOS: `~/Library/Application Support/bkt2gh/config.yaml`
- Windows: `%AppData%\bkt2gh\config.yaml`

## Encryption

`config.yaml` stores only an encryption envelope:

```yaml
version: 1
encryption: os-keyring-aes-gcm
nonce: <base64>
data: <base64>
```

The decrypted payload contains Bitbucket and GitHub migration fields. The payload is encrypted with AES-256-GCM. The random 256-bit key is stored in the OS credential store/keychain through `github.com/zalando/go-keyring`.

## Compatibility

Environment variables continue to override file configuration for CI and automation.

## Testing

Tests use an in-memory keyring. They verify encrypted config round trips, file contents do not leak config values, environment variables override encrypted config, and missing files can still use environment variables.
