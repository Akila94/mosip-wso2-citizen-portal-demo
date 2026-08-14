/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 */
package org.wso2.carbon.identity.application.authenticator.esignet;

import com.nimbusds.jose.JOSEException;
import com.nimbusds.jwt.JWTClaimsSet;
import com.nimbusds.jwt.SignedJWT;
import org.apache.commons.lang3.StringUtils;
import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.apache.oltu.oauth2.client.request.OAuthClientRequest;
import org.apache.oltu.oauth2.client.response.OAuthClientResponse;
import org.apache.oltu.oauth2.common.exception.OAuthSystemException;
import org.apache.oltu.oauth2.common.message.types.GrantType;
import org.wso2.carbon.identity.application.authentication.framework.context.AuthenticationContext;
import org.wso2.carbon.identity.application.authentication.framework.exception.AuthenticationFailedException;
import org.wso2.carbon.identity.application.authenticator.oidc.OIDCAuthenticatorConstants;
import org.wso2.carbon.identity.application.authenticator.oidc.OpenIDConnectAuthenticator;
import org.wso2.carbon.identity.application.authenticator.oidc.util.OIDCCommonUtil;
import org.wso2.carbon.identity.application.common.model.ClaimMapping;
import org.wso2.carbon.identity.application.common.model.Property;
import org.wso2.carbon.identity.application.common.util.IdentityApplicationConstants;
import org.wso2.carbon.identity.base.IdentityRuntimeException;
import org.wso2.carbon.identity.core.ServiceURLBuilder;
import org.wso2.carbon.identity.core.URLBuilderException;
import org.wso2.carbon.identity.core.IdentityKeyStoreResolver;
import org.wso2.carbon.identity.core.util.IdentityKeyStoreResolverConstants.InboundProtocol;
import org.wso2.carbon.identity.core.util.IdentityKeyStoreResolverException;
import org.wso2.carbon.identity.core.util.IdentityUtil;
import org.wso2.carbon.identity.oauth2.IdentityOAuth2Exception;
import org.wso2.carbon.identity.oauth2.util.JWTSignatureValidationUtils;

import java.io.IOException;
import java.security.Key;
import java.security.PrivateKey;
import java.security.cert.Certificate;
import java.security.interfaces.RSAPublicKey;
import java.text.ParseException;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Federated authenticator for MOSIP eSignet.
 * <p>
 * The stock outbound OIDC connector cannot federate to eSignet for exactly two reasons,
 * both verified in source (see RUNBOOK-DELTA.md for citations):
 * <ol>
 *   <li>eSignet authenticates clients at the token endpoint with {@code private_key_jwt}
 *       only — {@code client_assertion} is {@code @NotBlank} on its {@code TokenRequest}
 *       and no {@code client_secret} code path exists — while
 *       {@link OpenIDConnectAuthenticator#getAccessTokenRequest} branches solely on
 *       {@code IsBasicAuthEnabled}, i.e. Basic header or credentials in the body.</li>
 *   <li>eSignet always returns UserInfo as a signed JWT, while
 *       {@link OpenIDConnectAuthenticator#getSubjectAttributes} parses the response as
 *       plain JSON. eSignet also puts no user claims in the ID token, so skipping UserInfo
 *       is not a workaround.</li>
 * </ol>
 * Those are the only two overrides here. Everything else — the authorization request,
 * {@code nonce}, PKCE (S256, driven by the stock {@code IsPKCEEnabled} property), ID token
 * handling, claim dialect mapping, JIT provisioning and the session — is inherited
 * unchanged from the product connector.
 */
public class EsignetOIDCAuthenticator extends OpenIDConnectAuthenticator {

    private static final long serialVersionUID = 1L;
    private static final Log LOG = LogFactory.getLog(EsignetOIDCAuthenticator.class);

    @Override
    public String getName() {

        return EsignetAuthenticatorConstants.AUTHENTICATOR_NAME;
    }

    @Override
    public String getFriendlyName() {

        return EsignetAuthenticatorConstants.AUTHENTICATOR_FRIENDLY_NAME;
    }

    @Override
    public String getI18nKey() {

        return EsignetAuthenticatorConstants.AUTHENTICATOR_I18N_KEY;
    }

    /**
     * Same properties as the stock connector, minus the two that make no sense here
     * ({@code ClientSecret} and {@code IsBasicAuthEnabled} — this connector never
     * authenticates with a secret), plus eSignet's JWKS URL and the assertion lifetime.
     *
     * @return Configuration properties rendered by the Console.
     */
    @Override
    public List<Property> getConfigurationProperties() {

        List<Property> properties = new ArrayList<>();
        for (Property property : super.getConfigurationProperties()) {
            boolean secretBased = IdentityApplicationConstants.Authenticator.OIDC.CLIENT_SECRET
                    .equals(property.getName())
                    || IdentityApplicationConstants.Authenticator.OIDC.IS_BASIC_AUTH_ENABLED
                    .equals(property.getName());
            if (!secretBased) {
                properties.add(property);
            }
        }

        Property jwksUrl = new Property();
        jwksUrl.setName(EsignetAuthenticatorConstants.JWKS_URL);
        jwksUrl.setDisplayName("eSignet JWKS Endpoint URL");
        jwksUrl.setRequired(true);
        jwksUrl.setDescription("JWKS endpoint used to verify the signed UserInfo response, "
                + "e.g. http://localhost:8088/v1/esignet/oauth/.well-known/jwks.json. Note that this is "
                + "eSignet's API base URL, not the URL of its login UI.");
        jwksUrl.setType("string");
        jwksUrl.setDisplayOrder(12);
        properties.add(jwksUrl);

        Property assertionValidity = new Property();
        assertionValidity.setName(EsignetAuthenticatorConstants.ASSERTION_VALIDITY_MINUTES);
        assertionValidity.setDisplayName("Client Assertion Validity (minutes)");
        assertionValidity.setRequired(false);
        assertionValidity.setDescription("Lifetime of the private_key_jwt client assertion. Defaults to "
                + EsignetAuthenticatorConstants.DEFAULT_ASSERTION_VALIDITY_MINUTES + " minutes.");
        assertionValidity.setType("string");
        assertionValidity.setDefaultValue(
                String.valueOf(EsignetAuthenticatorConstants.DEFAULT_ASSERTION_VALIDITY_MINUTES));
        assertionValidity.setDisplayOrder(13);
        properties.add(assertionValidity);

        return properties;
    }

    /**
     * Build the token request with a {@code private_key_jwt} client assertion instead of
     * client credentials.
     * <p>
     * Structurally identical to the superclass implementation — same callback resolution,
     * same PKCE handling, same {@code Origin} header — with the credential fields replaced.
     * {@code super} is deliberately not called: both of its branches send a client secret.
     *
     * @param context       Authentication context of the current flow.
     * @param authzResponse Authorization response carrying the code.
     * @return The token request to POST to eSignet.
     * @throws AuthenticationFailedException if the assertion cannot be built or signed.
     */
    @Override
    protected OAuthClientRequest getAccessTokenRequest(AuthenticationContext context,
                                                       org.apache.oltu.oauth2.client.response.OAuthAuthzResponse
                                                               authzResponse) throws AuthenticationFailedException {

        Map<String, String> authenticatorProperties = context.getAuthenticatorProperties();
        String clientId = authenticatorProperties.get(OIDCAuthenticatorConstants.CLIENT_ID);
        String tokenEndpoint = getTokenEndpoint(authenticatorProperties);
        boolean isPKCEEnabled = Boolean.parseBoolean(
                authenticatorProperties.get(OIDCAuthenticatorConstants.IS_PKCE_ENABLED));
        String codeVerifier = (String) context.getProperty(OIDCAuthenticatorConstants.PKCE_CODE_VERIFIER);

        String callbackUrl = getCallbackUrlFromQueryParamMap(context);
        if (StringUtils.isBlank(callbackUrl)) {
            callbackUrl = getCallbackUrl(authenticatorProperties, context);
        }

        String assertion = buildClientAssertion(context, clientId, tokenEndpoint);

        OAuthClientRequest accessTokenRequest;
        try {
            OAuthClientRequest.TokenRequestBuilder tokenRequestBuilder = OAuthClientRequest
                    .tokenLocation(tokenEndpoint)
                    .setGrantType(GrantType.AUTHORIZATION_CODE)
                    .setRedirectURI(callbackUrl)
                    .setCode(authzResponse.getCode())
                    .setParameter(EsignetAuthenticatorConstants.CLIENT_ID_PARAM, clientId)
                    .setParameter(EsignetAuthenticatorConstants.CLIENT_ASSERTION_TYPE_PARAM,
                            EsignetAuthenticatorConstants.CLIENT_ASSERTION_TYPE_JWT_BEARER)
                    .setParameter(EsignetAuthenticatorConstants.CLIENT_ASSERTION_PARAM, assertion);

            if (isPKCEEnabled) {
                if (StringUtils.isEmpty(codeVerifier)) {
                    throw new AuthenticationFailedException("PKCE is enabled, but the code verifier is not found.");
                }
                tokenRequestBuilder.setParameter(EsignetAuthenticatorConstants.CODE_VERIFIER_PARAM, codeVerifier);
            }

            accessTokenRequest = tokenRequestBuilder.buildBodyMessage();
            context.removeProperty(OIDCAuthenticatorConstants.PKCE_CODE_VERIFIER);
            if (accessTokenRequest != null) {
                /*
                 ServiceURLBuilder.build() is deprecated in favour of build(hostname). The
                 hostname it used to resolve internally is what IdentityUtil.getHostName()
                 returns — the server's configured HostName, falling back to the local
                 hostname — which is also the pattern the framework's own current call sites
                 use (UserAssertionUtils, AuthenticationAssertionUtils).
                */
                accessTokenRequest.addHeader(OIDCAuthenticatorConstants.HTTP_ORIGIN_HEADER,
                        ServiceURLBuilder.create().build(IdentityUtil.getHostName()).getAbsolutePublicURL());
            }
        } catch (OAuthSystemException e) {
            throw new AuthenticationFailedException(
                    EsignetAuthenticatorConstants.ErrorMessages.TOKEN_REQUEST_BUILD_FAILED,
                    "Error while building the token request for eSignet endpoint: "
                            + LogSanitizer.clean(tokenEndpoint), e);
        } catch (URLBuilderException e) {
            throw new AuthenticationFailedException(
                    EsignetAuthenticatorConstants.ErrorMessages.TOKEN_REQUEST_BUILD_FAILED,
                    "Error while resolving this server's public URL for the Origin header.", e);
        }

        if (LOG.isDebugEnabled()) {
            LOG.debug("Authenticating to eSignet token endpoint " + LogSanitizer.clean(tokenEndpoint)
                    + " with a private_key_jwt client assertion.");
        }
        return accessTokenRequest;
    }

    /**
     * Fetch UserInfo and, when eSignet returns it as a signed JWT, verify that JWS against
     * eSignet's JWKS before any claim is trusted.
     * <p>
     * A plain JSON body is delegated to the superclass, which keeps this connector usable
     * against an ordinary OIDC provider.
     *
     * @param token                   Token response holding the access token.
     * @param authenticatorProperties Authenticator properties of the connection.
     * @return Claim mappings for the authenticated user.
     */
    @Override
    protected Map<ClaimMapping, String> getSubjectAttributes(OAuthClientResponse token,
                                                             Map<String, String> authenticatorProperties) {

        String userInfoEndpoint = getUserInfoEndpoint(token, authenticatorProperties);
        if (StringUtils.isBlank(userInfoEndpoint)) {
            LOG.warn("No UserInfo endpoint is configured; eSignet returns no user claims in the ID token, "
                    + "so the authenticated user will carry no attributes.");
            return new HashMap<>();
        }

        String body;
        try {
            body = sendRequest(userInfoEndpoint, token.getParam(OIDCAuthenticatorConstants.ACCESS_TOKEN));
        } catch (IOException e) {
            throw IdentityRuntimeException.error(
                    EsignetAuthenticatorConstants.ErrorMessages.USERINFO_REQUEST_FAILED
                            + " Communication error while calling the eSignet UserInfo endpoint.", e);
        }
        body = StringUtils.trimToEmpty(body);
        if (body.length() > EsignetAuthenticatorConstants.MAX_USERINFO_BODY_CHARS) {
            // Bound what a remote response can cost us before parsing it.
            throw IdentityRuntimeException.error(
                    EsignetAuthenticatorConstants.ErrorMessages.USERINFO_TOO_LARGE
                            + " The UserInfo response exceeds "
                            + EsignetAuthenticatorConstants.MAX_USERINFO_BODY_CHARS + " characters.");
        }

        int segments = body.isEmpty() ? 0 : body.split(EsignetAuthenticatorConstants.JWS_SEGMENT_SEPARATOR, -1).length;
        if (segments == EsignetAuthenticatorConstants.JWE_SEGMENTS) {
            // eSignet was registered with userinfo_response_type=JWE; decryption is not implemented.
            throw IdentityRuntimeException.error(
                    EsignetAuthenticatorConstants.ErrorMessages.USERINFO_ENCRYPTED
                            + " The UserInfo response is encrypted (JWE). Register the eSignet client with "
                            + "additionalConfig.userinfo_response_type = JWS.");
        }
        if (segments != EsignetAuthenticatorConstants.JWS_SEGMENTS) {
            // Not a JWT: an ordinary OIDC provider returning JSON. Let the product connector handle it.
            return super.getSubjectAttributes(token, authenticatorProperties);
        }

        return extractVerifiedClaims(body, token, authenticatorProperties);
    }

    /**
     * Verify a signed UserInfo response and convert its claims to claim mappings.
     * <p>
     * Two checks gate every claim: the JWS signature against eSignet's JWKS, and the
     * {@code sub} binding required by OpenID Connect Core 5.3.2 — the UserInfo {@code sub}
     * must exactly match the ID token {@code sub}, or the response must not be used. Without
     * that binding, a UserInfo response issued for one subject could contribute attributes to
     * a session authenticated as another.
     *
     * @param jws                     Serialised JWS returned by the UserInfo endpoint.
     * @param token                   Token response, for the ID token this must bind to.
     * @param authenticatorProperties Authenticator properties of the connection.
     * @return Claim mappings extracted from the verified payload.
     */
    private Map<ClaimMapping, String> extractVerifiedClaims(String jws, OAuthClientResponse token,
                                                            Map<String, String> authenticatorProperties) {

        String jwksUrl = authenticatorProperties.get(EsignetAuthenticatorConstants.JWKS_URL);
        if (StringUtils.isBlank(jwksUrl)) {
            throw IdentityRuntimeException.error(
                    EsignetAuthenticatorConstants.ErrorMessages.USERINFO_JWKS_URL_MISSING
                            + " eSignet returned a signed UserInfo response but no JWKS endpoint is configured "
                            + "on the connection, so its signature cannot be verified.");
        }

        try {
            SignedJWT signedJWT = SignedJWT.parse(jws);
            if (!JWTSignatureValidationUtils.validateUsingJWKSUri(signedJWT, jwksUrl)) {
                throw IdentityRuntimeException.error(
                        EsignetAuthenticatorConstants.ErrorMessages.USERINFO_SIGNATURE_INVALID
                                + " The UserInfo response signature did not validate against "
                                + LogSanitizer.clean(jwksUrl) + ".");
            }
            JWTClaimsSet claimsSet = signedJWT.getJWTClaimsSet();
            assertSubjectMatchesIdToken(claimsSet.getSubject(), token);
            if (LOG.isDebugEnabled()) {
                // Claim names come off the network: never log them raw (log forging).
                LOG.debug("Verified signed UserInfo response from eSignet; claims: "
                        + LogSanitizer.clean(claimsSet.getClaims().keySet()));
            }
            // Reuse the product connector's mapping so claim semantics stay identical.
            return OIDCCommonUtil.extractUserClaimsFromJsonPayload(claimsSet.toString());
        } catch (ParseException e) {
            throw IdentityRuntimeException.error(
                    EsignetAuthenticatorConstants.ErrorMessages.USERINFO_MALFORMED
                            + " The UserInfo response is not a parseable JWT.", e);
        } catch (IdentityOAuth2Exception e) {
            throw IdentityRuntimeException.error(
                    EsignetAuthenticatorConstants.ErrorMessages.USERINFO_SIGNATURE_INVALID
                            + " Error while verifying the UserInfo signature against " + LogSanitizer.clean(jwksUrl)
                            + ".", e);
        }
    }

    /**
     * Enforce the OpenID Connect Core 5.3.2 {@code sub} check between the UserInfo response
     * and the ID token that established the subject.
     *
     * @param userInfoSubject {@code sub} of the verified UserInfo response.
     * @param token           Token response holding the ID token.
     * @throws ParseException if the ID token is not a parseable JWT.
     */
    private void assertSubjectMatchesIdToken(String userInfoSubject, OAuthClientResponse token) throws ParseException {

        String idToken = token.getParam(OIDCAuthenticatorConstants.ID_TOKEN);
        if (StringUtils.isBlank(idToken)) {
            // No ID token in this response: the superclass then derives the subject from
            // UserInfo itself, so there are not two identities to reconcile.
            return;
        }
        String idTokenSubject = SignedJWT.parse(idToken).getJWTClaimsSet().getSubject();
        if (StringUtils.isBlank(userInfoSubject) || !userInfoSubject.equals(idTokenSubject)) {
            throw IdentityRuntimeException.error(
                    EsignetAuthenticatorConstants.ErrorMessages.USERINFO_SUBJECT_MISMATCH
                            + " The UserInfo 'sub' does not match the ID token 'sub'; the response was discarded.");
        }
    }

    /**
     * Sign a client assertion with this server's OAuth signing key.
     * <p>
     * The key comes from IS's own keystore through {@link IdentityKeyStoreResolver}, so it
     * honours tenant keystores and any custom keystore mapped to the OAuth protocol. Its
     * public half — exported as a JWK by {@code JwkExporter} and registered with eSignet —
     * is what eSignet verifies the assertion against, and the {@code kid} is the JWK
     * thumbprint of that same key.
     *
     * @param context       Authentication context, used for the tenant domain.
     * @param clientId      Client id registered with eSignet.
     * @param tokenEndpoint Token endpoint URL, used verbatim as the audience.
     * @return Serialised client assertion.
     * @throws AuthenticationFailedException if the key is unavailable or signing fails.
     */
    private String buildClientAssertion(AuthenticationContext context, String clientId, String tokenEndpoint)
            throws AuthenticationFailedException {

        String tenantDomain = context.getTenantDomain();
        try {
            IdentityKeyStoreResolver keyStoreResolver = IdentityKeyStoreResolver.getInstance();
            Key privateKey = keyStoreResolver.getPrivateKey(tenantDomain, InboundProtocol.OAUTH);
            Certificate certificate = keyStoreResolver.getCertificate(tenantDomain, InboundProtocol.OAUTH);

            if (!(privateKey instanceof PrivateKey) || !(certificate.getPublicKey() instanceof RSAPublicKey)) {
                throw new AuthenticationFailedException(
                        EsignetAuthenticatorConstants.ErrorMessages.SIGNING_KEY_UNAVAILABLE,
                        "The signing key resolved for tenant " + LogSanitizer.clean(tenantDomain)
                                + " is not an RSA key pair. "
                                + "eSignet accepts RS256, PS256 or ES256 client assertions.");
            }

            String keyId = JwkUtil.thumbprint((RSAPublicKey) certificate.getPublicKey());
            return ClientAssertionBuilder.build(clientId, tokenEndpoint, (PrivateKey) privateKey, keyId,
                    getAssertionValidityMinutes(context.getAuthenticatorProperties()));
        } catch (IdentityKeyStoreResolverException e) {
            throw new AuthenticationFailedException(
                    EsignetAuthenticatorConstants.ErrorMessages.SIGNING_KEY_UNAVAILABLE,
                    "Could not resolve the OAuth signing key for tenant " + LogSanitizer.clean(tenantDomain) + ".", e);
        } catch (JOSEException e) {
            throw new AuthenticationFailedException(
                    EsignetAuthenticatorConstants.ErrorMessages.ASSERTION_SIGNING_FAILED,
                    "Error while signing the private_key_jwt client assertion.", e);
        }
    }

    /**
     * Read the configured assertion lifetime, falling back to the default when it is absent
     * or not a positive integer.
     *
     * @param authenticatorProperties Authenticator properties of the connection.
     * @return Assertion lifetime in minutes.
     */
    private int getAssertionValidityMinutes(Map<String, String> authenticatorProperties) {

        String configured = authenticatorProperties.get(EsignetAuthenticatorConstants.ASSERTION_VALIDITY_MINUTES);
        if (StringUtils.isNotBlank(configured)) {
            try {
                int minutes = Integer.parseInt(configured.trim());
                if (minutes > 0) {
                    return minutes;
                }
            } catch (NumberFormatException e) {
                LOG.warn("Ignoring a non-numeric " + EsignetAuthenticatorConstants.ASSERTION_VALIDITY_MINUTES
                        + " property value; falling back to the default.");
            }
        }
        return EsignetAuthenticatorConstants.DEFAULT_ASSERTION_VALIDITY_MINUTES;
    }

    /**
     * Read {@code redirect_uri} out of the query parameter map the superclass populates
     * from the authorization request. The superclass keeps its own copy of this lookup
     * private, and the redirect URI must be byte-identical in the authorization and token
     * requests, so it is repeated here rather than approximated.
     *
     * @param context Authentication context of the current flow.
     * @return The redirect URI used in the authorization request, or null if absent.
     */
    private String getCallbackUrlFromQueryParamMap(AuthenticationContext context) {

        Object paramMap = context.getProperty(OIDCAuthenticatorConstants.OIDC_QUERY_PARAM_MAP_PROPERTY_KEY);
        if (paramMap instanceof Map) {
            Object redirectUri = ((Map<?, ?>) paramMap).get(OIDCAuthenticatorConstants.REDIRECT_URI);
            if (redirectUri instanceof String && StringUtils.isNotBlank((String) redirectUri)) {
                return (String) redirectUri;
            }
        }
        return null;
    }
}
