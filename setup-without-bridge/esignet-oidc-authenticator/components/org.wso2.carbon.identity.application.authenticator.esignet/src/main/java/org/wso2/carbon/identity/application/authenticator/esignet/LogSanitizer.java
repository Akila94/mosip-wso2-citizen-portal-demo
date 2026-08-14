/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 */
package org.wso2.carbon.identity.application.authenticator.esignet;

/**
 * Strips line breaks from externally supplied text before it reaches a log or an
 * exception message.
 * <p>
 * Required by the WSO2 secure engineering guidelines (log injection / log forging, CRLF):
 * a claim name or configured URL that contains CR or LF can otherwise inject a complete,
 * fabricated line into {@code wso2carbon.log}. Values are also length-capped, so a hostile
 * UserInfo response cannot flood the log.
 */
public final class LogSanitizer {

    private static final int DEFAULT_MAX_LENGTH = 256;
    private static final String ELLIPSIS = "…";

    private LogSanitizer() {

    }

    /**
     * @param value Untrusted text.
     * @return The text with CR, LF and tab removed and its length capped, or {@code "null"}.
     */
    public static String clean(Object value) {

        return clean(value, DEFAULT_MAX_LENGTH);
    }

    /**
     * @param value     Untrusted text.
     * @param maxLength Maximum number of characters to keep.
     * @return The text with CR, LF and tab removed and its length capped, or {@code "null"}.
     */
    public static String clean(Object value, int maxLength) {

        if (value == null) {
            return "null";
        }
        String cleaned = value.toString().replace('\r', ' ').replace('\n', ' ').replace('\t', ' ');
        if (cleaned.length() > maxLength) {
            return cleaned.substring(0, maxLength) + ELLIPSIS;
        }
        return cleaned;
    }
}
