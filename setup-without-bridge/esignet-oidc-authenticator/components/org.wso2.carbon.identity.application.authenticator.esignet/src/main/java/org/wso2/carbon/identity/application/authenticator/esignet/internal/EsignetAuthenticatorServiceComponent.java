/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 */
package org.wso2.carbon.identity.application.authenticator.esignet.internal;

import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import org.osgi.service.component.ComponentContext;
import org.osgi.service.component.annotations.Activate;
import org.osgi.service.component.annotations.Component;
import org.osgi.service.component.annotations.Deactivate;
import org.wso2.carbon.identity.application.authentication.framework.ApplicationAuthenticator;
import org.wso2.carbon.identity.application.authenticator.esignet.EsignetOIDCAuthenticator;

/**
 * Registers the eSignet federated authenticator with the authentication framework.
 * <p>
 * No service references are declared: the authenticator inherits every collaborator it
 * needs from the product OIDC connector, whose own service component populates the data
 * holder those inherited code paths read.
 */
@Component(
        name = "identity.application.authenticator.esignet.component",
        immediate = true
)
public class EsignetAuthenticatorServiceComponent {

    private static final Log LOG = LogFactory.getLog(EsignetAuthenticatorServiceComponent.class);

    @Activate
    protected void activate(ComponentContext context) {

        try {
            context.getBundleContext().registerService(ApplicationAuthenticator.class.getName(),
                    new EsignetOIDCAuthenticator(), null);
            LOG.info("MOSIP eSignet federated authenticator bundle is activated.");
        } catch (Throwable e) {
            LOG.fatal("Error while activating the MOSIP eSignet federated authenticator.", e);
        }
    }

    @Deactivate
    protected void deactivate(ComponentContext context) {

        if (LOG.isDebugEnabled()) {
            LOG.debug("MOSIP eSignet federated authenticator bundle is deactivated.");
        }
    }
}
