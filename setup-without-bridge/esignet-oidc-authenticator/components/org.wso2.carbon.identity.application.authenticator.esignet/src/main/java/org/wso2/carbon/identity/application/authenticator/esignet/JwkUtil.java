/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 */
package org.wso2.carbon.identity.application.authenticator.esignet;

import java.math.BigInteger;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.interfaces.RSAPublicKey;
import java.util.Base64;

import static java.nio.charset.StandardCharsets.UTF_8;

/**
 * RSA public key to JWK conversion, on plain JDK APIs only.
 * <p>
 * Deliberately free of Nimbus (and of every other) dependency: the same code has to run
 * inside the OSGi runtime, where it produces the {@code kid} of the client assertion
 * header, and standalone under {@code java -cp <jar>} in
 * {@link org.wso2.carbon.identity.application.authenticator.esignet.tools.JwkExporter},
 * where it produces the JWK registered with eSignet. Both {@code kid} values must be
 * byte-identical or eSignet cannot select the key that verifies the assertion, so there is
 * exactly one implementation.
 */
public final class JwkUtil {

    private JwkUtil() {

    }

    /**
     * RFC 7638 JWK thumbprint of an RSA public key: SHA-256 over the canonical JSON of the
     * required members in lexicographic order ("e", "kty", "n"), base64url encoded.
     *
     * @param publicKey RSA public key.
     * @return The thumbprint, used as the JWK {@code kid}.
     */
    public static String thumbprint(RSAPublicKey publicKey) {

        String canonical = "{\"e\":\"" + base64UrlUInt(publicKey.getPublicExponent())
                + "\",\"kty\":\"RSA\",\"n\":\"" + base64UrlUInt(publicKey.getModulus()) + "\"}";
        try {
            byte[] digest = MessageDigest.getInstance("SHA-256").digest(canonical.getBytes(UTF_8));
            return Base64.getUrlEncoder().withoutPadding().encodeToString(digest);
        } catch (NoSuchAlgorithmException e) {
            // SHA-256 is mandated by the JCA specification; unreachable on any supported JVM.
            throw new IllegalStateException("SHA-256 is not available in this JVM.", e);
        }
    }

    /**
     * Serialise an RSA public key as an RFC 7517 public JWK for signature verification.
     *
     * @param publicKey RSA public key.
     * @return Compact JSON JWK, with a {@code kid} equal to {@link #thumbprint}.
     */
    public static String toPublicJwkJson(RSAPublicKey publicKey) {

        return "{\"kty\":\"RSA\""
                + ",\"kid\":\"" + thumbprint(publicKey) + "\""
                + ",\"use\":\"sig\""
                + ",\"alg\":\"" + EsignetAuthenticatorConstants.RS256 + "\""
                + ",\"n\":\"" + base64UrlUInt(publicKey.getModulus()) + "\""
                + ",\"e\":\"" + base64UrlUInt(publicKey.getPublicExponent()) + "\"}";
    }

    /**
     * Encode a positive integer as a JWK Base64urlUInt value: the minimal big-endian octet
     * representation, without the sign byte BigInteger prepends, base64url encoded.
     *
     * @param value Positive integer, i.e. an RSA modulus or public exponent.
     * @return Base64urlUInt encoding of the value.
     */
    static String base64UrlUInt(BigInteger value) {

        byte[] bytes = value.toByteArray();
        int offset = 0;
        while (offset < bytes.length - 1 && bytes[offset] == 0) {
            offset++;
        }
        byte[] trimmed = new byte[bytes.length - offset];
        System.arraycopy(bytes, offset, trimmed, 0, trimmed.length);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(trimmed);
    }
}
