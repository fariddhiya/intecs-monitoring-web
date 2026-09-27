#!/bin/bash
set -e

echo "[INFO] Configuring HiveMQ authentication..."

# Use plain-text passwords for reliable verification
# (matches the official ESE file-realm example format)

# Create ESE file-realm config
mkdir -p /opt/hivemq/extensions/hivemq-enterprise-security-extension/conf

cat > /opt/hivemq/extensions/hivemq-enterprise-security-extension/conf/config.xml << 'ESECONFIG'
<?xml version="1.0" encoding="UTF-8" ?>
<enterprise-security-extension
        xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
        xsi:noNamespaceSchemaLocation="config.xsd"
        version="1">
    <realms>
        <file-realm>
            <name>file-realm</name>
            <enabled>true</enabled>
            <configuration>
                <file-path>/opt/hivemq/conf/ese-file-realm.xml</file-path>
            </configuration>
        </file-realm>
    </realms>
    <pipelines>
        <listener-pipeline listener="ALL">
            <file-authentication-manager>
                <realm>file-realm</realm>
            </file-authentication-manager>
            <file-authorization-manager>
                <realm>file-realm</realm>
            </file-authorization-manager>
        </listener-pipeline>
        <control-center-pipeline>
            <file-authentication-manager>
                <realm>file-realm</realm>
            </file-authentication-manager>
            <file-authorization-manager>
                <realm>file-realm</realm>
            </file-authorization-manager>
        </control-center-pipeline>
        <rest-api-pipeline listener="ALL">
            <authentication-preprocessors>
                <http-headers-preprocessor>
                    <basic-auth-extraction/>
                </http-headers-preprocessor>
            </authentication-preprocessors>
            <file-authentication-manager>
                <realm>file-realm</realm>
            </file-authentication-manager>
            <file-authorization-manager>
                <realm>file-realm</realm>
            </file-authorization-manager>
        </rest-api-pipeline>
    </pipelines>
</enterprise-security-extension>
ESECONFIG

# Create ESE file-realm users config with plain-text passwords
cat > /opt/hivemq/conf/ese-file-realm.xml << 'EOF'
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<ese-file-realm xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                xsi:noNamespaceSchemaLocation="ese-file-realm.xsd">
    <mqtt>
        <users>
            <user>
                <name>intecs</name>
                <password>intecs123</password>
                <permissions>
                    <permission>
                        <topic>#</topic>
                        <qos>ALL</qos>
                        <activity>ALL</activity>
                        <retain>ALL</retain>
                    </permission>
                </permissions>
            </user>
        </users>
    </mqtt>
    <control-center>
        <users>
            <user>
                <name>intecs</name>
                <password>intecs123</password>
                <permissions>
                    <permission>HIVEMQ_SUPER_ADMIN</permission>
                </permissions>
            </user>
        </users>
    </control-center>
    <rest-api>
        <users>
            <user>
                <name>intecs</name>
                <password>intecs123</password>
                <permissions>
                    <permission>HIVEMQ_SUPER_ADMIN</permission>
                </permissions>
            </user>
        </users>
    </rest-api>
</ese-file-realm>
EOF
echo "[OK] ESE configs created (config.xml + file-realm.xml)"

# Remove allow-all extension completely  
if [ -d "/opt/hivemq/extensions/hivemq-allow-all-extension" ]; then
    rm -rf /opt/hivemq/extensions/hivemq-allow-all-extension
    echo "[OK] Allow-all extension removed"
fi

# Enable Enterprise Security Extension
rm -f /opt/hivemq/extensions/hivemq-enterprise-security-extension/DISABLED 2>/dev/null || true
echo "[OK] Enterprise Security Extension enabled"

# Enable HiveMQ Client Event History (insert before </broker-config>)
if ! grep -q "<client-event-history>" /opt/hivemq/conf/config.xml; then
    sed -i 's|^\(\s*</broker-config>\)|    <client-event-history>\n        <enabled>true</enabled>\n        <storage-lifetime-seconds>604800</storage-lifetime-seconds>\n    </client-event-history>\n\1|' /opt/hivemq/conf/config.xml
    echo "[OK] Client Event History enabled (1 week storage)"
else
    echo "[INFO] Client Event History already configured"
fi

# Start HiveMQ via run.sh
exec /opt/hivemq/bin/run.sh
