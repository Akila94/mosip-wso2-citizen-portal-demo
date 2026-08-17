# `wso2is-local.pem`

The leaf certificate WSO2 IS 7.3.0 ships by default under alias `wso2carbon` in
`repository/resources/security/wso2carbon.p12` (password `wso2carbon`, the product default —
not a secret, and not specific to this install). `CN=localhost`, self-signed, exported with:

```bash
keytool -exportcert -alias wso2carbon \
  -keystore wso2is-7.3.0/repository/resources/security/wso2carbon.p12 \
  -storetype PKCS12 -storepass wso2carbon -rfc -file wso2is-local.pem
```

The BFF loads this file into a dedicated `x509.CertPool` used only for its connection to IS —
**never `InsecureSkipVerify`** — so it can verify `https://localhost:9443` without trusting the
system root store. This is the standard shipped WSO2 demo certificate; regenerating IS's
keystore (or pointing at a different WSO2 IS instance) requires re-exporting this file.
