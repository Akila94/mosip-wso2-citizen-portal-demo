/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 */
package org.wso2.carbon.identity.application.authenticator.esignet;

/**
 * Constants of the MOSIP eSignet federated authenticator.
 */
public final class EsignetAuthenticatorConstants {

    private EsignetAuthenticatorConstants() {

    }

    /**
     * Authenticator name persisted in the IdP configuration and referenced by the
     * authentication sequence. Changing it orphans every existing connection.
     */
    public static final String AUTHENTICATOR_NAME = "EsignetOIDCAuthenticator";
    public static final String AUTHENTICATOR_FRIENDLY_NAME = "MOSIP eSignet";
    public static final String AUTHENTICATOR_I18N_KEY = "authenticator.esignet";

    /** Authenticator property holding eSignet's JWKS endpoint, used to verify signed UserInfo. */
    public static final String JWKS_URL = "EsignetJwksUrl";
    /** Authenticator property: client assertion lifetime in minutes. */
    public static final String ASSERTION_VALIDITY_MINUTES = "AssertionValidityMinutes";
    /** Identity provider property name under which IS stores a connection's JWKS URI. */
    public static final String IDP_JWKS_URI_PROPERTY = "jwksUri";

    public static final int DEFAULT_ASSERTION_VALIDITY_MINUTES = 5;

    public static final String CLIENT_ASSERTION_TYPE_PARAM = "client_assertion_type";
    public static final String CLIENT_ASSERTION_PARAM = "client_assertion";
    public static final String CLIENT_ASSERTION_TYPE_JWT_BEARER =
            "urn:ietf:params:oauth:client-assertion-type:jwt-bearer";
    public static final String CODE_VERIFIER_PARAM = "code_verifier";
    public static final String CLIENT_ID_PARAM = "client_id";

    public static final String RS256 = "RS256";
    public static final String JWS_SEGMENT_SEPARATOR = "\\.";
    public static final int JWS_SEGMENTS = 3;
    public static final int JWE_SEGMENTS = 5;

    /**
     * Upper bound on the UserInfo response we are willing to parse. A signed JWT carrying
     * the seven claims eSignet is registered to release is a few kilobytes; this leaves room
     * for a base64 photo while still bounding what a remote endpoint can make us process.
     */
    public static final int MAX_USERINFO_BODY_CHARS = 256 * 1024;

    /**
     * Error codes reported through AuthenticationFailedException. The 65xxx range mirrors
     * the numbering the stock OIDC connector uses for its own errors.
     */
    public static final class ErrorMessages {

        private ErrorMessages() {

        }

        public static final String SIGNING_KEY_UNAVAILABLE = "ESIGNET-65001";
        public static final String ASSERTION_SIGNING_FAILED = "ESIGNET-65002";
        public static final String TOKEN_REQUEST_BUILD_FAILED = "ESIGNET-65003";
        public static final String USERINFO_REQUEST_FAILED = "ESIGNET-65004";
        public static final String USERINFO_JWKS_URL_MISSING = "ESIGNET-65005";
        public static final String USERINFO_SIGNATURE_INVALID = "ESIGNET-65006";
        public static final String USERINFO_ENCRYPTED = "ESIGNET-65007";
        public static final String USERINFO_MALFORMED = "ESIGNET-65008";
        public static final String USERINFO_SUBJECT_MISMATCH = "ESIGNET-65009";
        public static final String USERINFO_TOO_LARGE = "ESIGNET-65010";
    }
}
