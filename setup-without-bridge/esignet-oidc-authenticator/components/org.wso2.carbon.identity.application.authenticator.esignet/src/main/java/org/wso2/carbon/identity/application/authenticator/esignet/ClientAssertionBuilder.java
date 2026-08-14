/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 */
package org.wso2.carbon.identity.application.authenticator.esignet;

import com.nimbusds.jose.JOSEException;
import com.nimbusds.jose.JWSAlgorithm;
import com.nimbusds.jose.JWSHeader;
import com.nimbusds.jose.JWSSigner;
import com.nimbusds.jose.crypto.RSASSASigner;
import com.nimbusds.jwt.JWTClaimsSet;
import com.nimbusds.jwt.SignedJWT;

import java.security.PrivateKey;
import java.util.Date;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

/**
 * Builds the {@code private_key_jwt} client assertion eSignet requires at its token
 * endpoint.
 * <p>
 * eSignet enforces all four of the following, so none of them is cosmetic:
 * <ul>
 *   <li>{@code iss} and {@code sub} both equal the client id;</li>
 *   <li>{@code aud} equals the token endpoint URL <em>as configured</em>, compared as a
 *       string — no normalisation, no trailing-slash tolerance;</li>
 *   <li>{@code jti} is unique across requests (replay cache);</li>
 *   <li>the signature algorithm is one of RS256/PS256/ES256.</li>
 * </ul>
 */
public final class ClientAssertionBuilder {

    private ClientAssertionBuilder() {

    }

    /**
     * Build and sign a client assertion.
     *
     * @param clientId         Client id registered with eSignet; becomes {@code iss} and {@code sub}.
     * @param tokenEndpoint    Token endpoint URL, used verbatim as {@code aud}.
     * @param privateKey       RSA private key whose public half is registered with eSignet.
     * @param keyId            JWK thumbprint of the matching public key, sent as {@code kid}.
     * @param validityMinutes  Assertion lifetime in minutes.
     * @return The serialised, signed JWT.
     * @throws JOSEException if signing fails.
     */
    public static String build(String clientId, String tokenEndpoint, PrivateKey privateKey, String keyId,
                               int validityMinutes) throws JOSEException {

        long now = System.currentTimeMillis();
        JWTClaimsSet claims = new JWTClaimsSet.Builder()
                .issuer(clientId)
                .subject(clientId)
                .audience(tokenEndpoint)
                .jwtID(UUID.randomUUID().toString())
                .issueTime(new Date(now))
                .expirationTime(new Date(now + TimeUnit.MINUTES.toMillis(validityMinutes)))
                .build();

        JWSHeader header = new JWSHeader.Builder(JWSAlgorithm.RS256)
                .keyID(keyId)
                .type(com.nimbusds.jose.JOSEObjectType.JWT)
                .build();

        SignedJWT assertion = new SignedJWT(header, claims);
        JWSSigner signer = new RSASSASigner(privateKey);
        assertion.sign(signer);
        return assertion.serialize();
    }
}
