# Webhooks

Webhooks is a sample application that can be used with
[`step-ca`](https://github.com/smallstep/certificates) to allow requests with an
ACME device-attest-01 flow.

It uses a small sqlite3 database that contains a table with the devices that are
allowed to get a certificate using the ACME DA flow.

## Building Webhooks

To build webhooks, just run the following commands:

```console
$ make
go build -o bin/webhooks main.go
dbmate --url "sqlite:db/database.sqlite3" --no-dump-schema up
```

In the second command, we are using the database migration tool
[dbmate](https://github.com/amacneil/dbmate) to setup the database.

## Running Webhooks

Webhooks can run with or without TLS support. For this example, we will use TLS,
more specifically, mTLS.

First, let's add our device to the database. If Jane is using a YubiKey with serial
number `112233`, we need to register it in the database:

```console
$ sqlite3 db/database.sqlite3 \
"insert into devices(id, type, owner, allow, data, created_at) values ('112233', 'YubiKey', 'jane@example.org', true, '{\"email\":\"jane@example.org\"}', unixepoch())"
```

We will generate a TLS certificate for webhooks using an already configured
[`step-ca`](https://github.com/smallstep/certificates). For this example
everything is running in localhost:

```console
$ step ca certificate localhost localhost.crt localhost.key
✔ Provisioner: admin (JWK)
✔ CA: https://localhost:9000
✔ Certificate: localhost.crt
✔ Private Key: localhost.key
$ step ca root > root_ca.crt
```

And will run webhooks with TLS support using the `--cert` and  `--key` flags,
and we will require a client certificate of a specific CA using the flag `--root`.

```console
$ bin/webhooks --cert localhost.crt --key localhost.key --root root_ca.crt
2023/03/02 17:29:38 starting https server at :3000
```

And finally, we need to configure `step-ca` with a provisioner that might look like this:

```json
{
    "type": "ACME",
    "name": "attestation",
    "challenges": [
        "device-attest-01"
    ],
    "attestationFormats": [
        "step"
    ],
    "options": {
        "webhooks": [
            {"name": "devices", "url": "https://localhost:3000/devices", "kind": "ENRICHING"}
        ]
    }
}
```

Now with `step-ca` running, Jane will be able to get a certificate:

```console
$ step ca certificate --provisioner attestation --attestation-uri 'yubikey:slot-id=9a?piv-value=123456' 112233 jane.crt jane.key
✔ Provisioner: attestation (ACME)
Using Device Attestation challenge to validate "112233" . done!
Waiting for Order to be 'ready' for finalization .. done!
Finalizing Order .. done!
✔ Certificate: jane.crt
✔ Private Key: yubikey:slot-id=9a?piv-value=123456
```

But John won't be able to get a certificate with his YubiKey:

```console
$ step ca certificate --provisioner attestation --attestation-uri 'yubikey:slot-id=9a?piv-value=123456' 998877 john.crt john.key
✔ Provisioner: attestation (ACME)
Using Device Attestation challenge to validate "998877" . done!
Waiting for Order to be 'ready' for finalization .. done!
Finalizing Order .error finalizing order: The server experienced an internal error
```

In our example, the response from `webhooks` is like this:

```json
{
    "data": {
        "email": "jane@example.org"
    },
    "allow": true
}
```

We can use that `data` property to extend the final certificate. With an
`step-ca` certificate template like this:

```tpl
{
    "subject": {{ toJson .Subject }},
    "sans": {{ toJson .SANs }},
{{- with .Webhooks.devices.email }}
    "emailAddresses": [{{ toJson . }}],
{{- end }}
{{- if typeIs "*rsa.PublicKey" .Insecure.CR.PublicKey }}
    "keyUsage": ["keyEncipherment", "digitalSignature"],
{{- else }}
    "keyUsage": ["digitalSignature"],
{{- end }}
    "extKeyUsage": ["serverAuth", "clientAuth"]
}
```

Jane will get a certificate with her permanent identifier and her email address:

```text
X509v3 Subject Alternative Name:
    email:jane@example.org
    Permanent Identifier: 112233
```
