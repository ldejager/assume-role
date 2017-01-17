Custom `assume-role` binary that calls the `aws sts assume-role` and `aws configure` commands in the background to obtain and write out temporary credentials.

This tool has been adapted from this [assume-role](https://github.com/remind101/assume-role) tool.

## Configuration

The first step is to setup "aliases" of the roles to assume, this is done in a yaml formatted configuration file in `~/.aws/roles`.

**Example**

```yaml
prod:
  role: arn:aws:iam::1234:role/SuperUser
  mfa: arn:aws:iam::5678:mfa/username # Enable MFA for this role.
```

## Usage

Obtain temporary credentials and set `AWS_PROFILE`

```bash
$ assume-role prod
MFA code: 123456
```

The utility can also be called using `eval` if required;

```bash
$ eval $(assume-role prod)
```
