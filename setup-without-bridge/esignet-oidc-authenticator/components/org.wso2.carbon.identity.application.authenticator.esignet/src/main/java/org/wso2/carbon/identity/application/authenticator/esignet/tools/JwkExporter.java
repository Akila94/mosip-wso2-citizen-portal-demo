/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 */
package org.wso2.carbon.identity.application.authenticator.esignet.tools;

import org.wso2.carbon.identity.application.authenticator.esignet.JwkUtil;

import java.io.BufferedReader;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.security.KeyStore;
import java.security.cert.Certificate;
import java.security.interfaces.RSAPublicKey;
import java.util.Arrays;

/**
 * Prints the public JWK of a keystore entry to stdout, for registration as an eSignet OIDC
 * client's {@code publicKey}.
 * <p>
 * Standalone on purpose — plain JDK only, no Nimbus, no OSGi — so setup automation can run
 * it straight out of the built JAR:
 * <pre>
 * ESIGNET_KEYSTORE_PASSWORD=... java -cp target/*.jar \
 *   org.wso2.carbon.identity.application.authenticator.esignet.tools.JwkExporter \
 *   wso2is-7.3.0/repository/resources/security/wso2carbon.jks wso2carbon
 * </pre>
 * The {@code kid} it emits is computed by {@link JwkUtil}, the same code the authenticator
 * uses for the client assertion header, so the two can never drift apart.
 * <p>
 * The keystore password is read from the {@code ESIGNET_KEYSTORE_PASSWORD} environment
 * variable, or from stdin when that is unset. It is never taken from the command line,
 * because process arguments are world-readable.
 */
public final class JwkExporter {

    private static final String PASSWORD_ENV_VAR = "ESIGNET_KEYSTORE_PASSWORD";
    private static final String TYPE_ENV_VAR = "ESIGNET_KEYSTORE_TYPE";
    private static final String DEFAULT_KEYSTORE_TYPE = "JKS";
    private static final int USAGE_ERROR = 2;

    private JwkExporter() {

    }

    /**
     * @param args Keystore path and alias.
     */
    public static void main(String[] args) throws Exception {

        if (args.length != 2) {
            System.err.println("Usage: JwkExporter <keystore-path> <alias>");
            System.err.println("Keystore password is read from $" + PASSWORD_ENV_VAR + ", or from stdin.");
            System.exit(USAGE_ERROR);
        }
        String keyStorePath = args[0];
        String alias = args[1];

        char[] password = readPassword();
        try {
            System.out.println(exportPublicJwk(keyStorePath, alias, password));
        } finally {
            Arrays.fill(password, '\0');
        }
    }

    /**
     * Load a keystore entry's certificate and render its RSA public key as a JWK.
     *
     * @param keyStorePath Path to the keystore file.
     * @param alias        Alias of the entry holding the signing certificate.
     * @param password     Keystore password.
     * @return The public JWK as JSON.
     * @throws Exception if the keystore cannot be read or the alias is not an RSA entry.
     */
    static String exportPublicJwk(String keyStorePath, String alias, char[] password) throws Exception {

        KeyStore keyStore = KeyStore.getInstance(keyStoreType());
        try (InputStream in = Files.newInputStream(Paths.get(keyStorePath))) {
            keyStore.load(in, password);
        }

        Certificate certificate = keyStore.getCertificate(alias);
        if (certificate == null) {
            throw new IllegalArgumentException("No certificate found under alias '" + alias + "' in " + keyStorePath);
        }
        if (!(certificate.getPublicKey() instanceof RSAPublicKey)) {
            throw new IllegalArgumentException("Alias '" + alias + "' does not hold an RSA key; eSignet client "
                    + "assertions must be RS256, PS256 or ES256.");
        }
        return JwkUtil.toPublicJwkJson((RSAPublicKey) certificate.getPublicKey());
    }

    /**
     * @return Keystore type, overridable for a PKCS12 keystore.
     */
    private static String keyStoreType() {

        String type = System.getenv(TYPE_ENV_VAR);
        return type == null || type.trim().isEmpty() ? DEFAULT_KEYSTORE_TYPE : type.trim();
    }

    /**
     * @return The keystore password from the environment, or the first line of stdin.
     */
    private static char[] readPassword() throws Exception {

        String fromEnv = System.getenv(PASSWORD_ENV_VAR);
        if (fromEnv != null && !fromEnv.isEmpty()) {
            return fromEnv.toCharArray();
        }
        try (BufferedReader reader = new BufferedReader(
                new InputStreamReader(System.in, StandardCharsets.UTF_8))) {
            String line = reader.readLine();
            if (line == null || line.isEmpty()) {
                throw new IllegalArgumentException("No keystore password supplied on $" + PASSWORD_ENV_VAR
                        + " or stdin.");
            }
            return line.toCharArray();
        }
    }
}
