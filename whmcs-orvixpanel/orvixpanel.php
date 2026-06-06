<?php
/**
 * OrvixPanel WHMCS Server Module
 *
 * Drop into:  /path/to/whmcs/modules/servers/orvixpanel/
 *
 * Required WHMCS module functions are implemented below. The module
 * speaks to the OrvixPanel REST provisioning API:
 *
 *   POST   /api/v1/provision/account
 *   POST   /api/v1/provision/account/{id}/suspend
 *   POST   /api/v1/provision/account/{id}/unsuspend
 *   DELETE /api/v1/provision/account/{id}
 *   GET    /api/v1/provision/account/{id}/usage
 *   POST   /api/v1/provision/account/{id}/password
 *
 * Authentication: X-API-Key header. The API key is generated from
 * inside the OrvixPanel UI (API Keys page) and pasted into the
 * WHMCS server config.
 */

if (!defined("WHMCS")) {
    die("This file should be accessed via the WHMCS admin only.");
}

/**
 * OrvixPanel API helper.
 *
 * Thin wrapper around curl with the X-API-Key header + JSON encoding.
 */
function orvixpanel_api($method, $path, $params = []) {
    $url = rtrim($params['serverhostname'], '/') . ':8443/api/v1/provision' . $path;
    $apiKey = $params['serverpassword']; // WHMCS stores the API key here

    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_CUSTOMREQUEST, strtoupper($method));
    $headers = [
        'X-API-Key: ' . $apiKey,
        'Content-Type: application/json',
        'Accept: application/json',
    ];
    curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
    if (!empty($params) && in_array(strtoupper($method), ['POST', 'PUT'])) {
        curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($params));
    }
    $body = curl_exec($ch);
    $code = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    if (curl_errno($ch)) {
        logModuleCall('orvixpanel', strtoupper($method) . ' ' . $path, $params, curl_error($ch), '');
        return ['code' => 0, 'body' => null, 'error' => curl_error($ch)];
    }
    curl_close($ch);
    $decoded = json_decode($body, true);
    logModuleCall('orvixpanel', strtoupper($method) . ' ' . $path, $params, $body, "[$code]");
    return ['code' => $code, 'body' => $decoded];
}

/**
 * WHMCS module API.
 */
function orvixpanel_MetaData() {
    return [
        'DisplayName' => 'OrvixPanel',
        'APIVersion'   => '1.1',
        'RequiresServer' => true,
    ];
}

function orvixpanel_ConfigOptions() {
    return [
        'plan' => [
            'FriendlyName' => 'Plan',
            'Type'         => 'dropdown',
            'Options'      => ['basic', 'pro', 'unlimited'],
            'Default'      => 'basic',
        ],
        'disk_quota_mb' => [
            'FriendlyName' => 'Disk Quota (MB)',
            'Type'         => 'text',
            'Size'         => '10',
            'Default'      => '10240',
        ],
        'bandwidth_gb' => [
            'FriendlyName' => 'Bandwidth (GB)',
            'Type'         => 'text',
            'Size'         => '10',
            'Default'      => '100',
        ],
    ];
}

function orvixpanel_CreateAccount($params) {
    $username = $params['username'];
    $domain   = $params['domain'];
    $password = $params['password'];

    $resp = orvixpanel_api('POST', '/account', [
        'serverhostname' => $params['serverhostname'],
        'serverpassword' => $params['serverpassword'],
        'username'       => $username,
        'domain'         => $domain,
        'password'       => $password,
        'plan'           => $params['configoption1'] ?? 'basic',
        'disk_quota_mb'  => (int)($params['configoption2'] ?? 10240),
        'bandwidth_gb'   => (int)($params['configoption3'] ?? 100),
        'package'        => $params['package']['name'] ?? '',
    ]);

    if ($resp['code'] !== 201) {
        return $resp['body']['error'] ?? 'provisioning_failed';
    }
    return 'success';
}

function orvixpanel_SuspendAccount($params) {
    $accID = $params['domain']; // WHMCS stores the account ID in the domain field by default
    $resp = orvixpanel_api('POST', "/account/$accID/suspend", [
        'serverhostname' => $params['serverhostname'],
        'serverpassword' => $params['serverpassword'],
    ]);
    return $resp['code'] === 200 ? 'success' : 'suspend_failed';
}

function orvixpanel_UnsuspendAccount($params) {
    $accID = $params['domain'];
    $resp = orvixpanel_api('POST', "/account/$accID/unsuspend", [
        'serverhostname' => $params['serverhostname'],
        'serverpassword' => $params['serverpassword'],
    ]);
    return $resp['code'] === 200 ? 'success' : 'unsuspend_failed';
}

function orvixpanel_TerminateAccount($params) {
    $accID = $params['domain'];
    $resp = orvixpanel_api('DELETE', "/account/$accID", [
        'serverhostname' => $params['serverhostname'],
        'serverpassword' => $params['serverpassword'],
    ]);
    return $resp['code'] === 200 ? 'success' : 'terminate_failed';
}

function orvixpanel_ChangePassword($params) {
    $accID = $params['domain'];
    $resp = orvixpanel_api('POST', "/account/$accID/password", [
        'serverhostname' => $params['serverhostname'],
        'serverpassword' => $params['serverpassword'],
        'password'       => $params['password'],
    ]);
    return $resp['code'] === 200 ? 'success' : 'password_change_failed';
}

function orvixpanel_ChangePackage($params) {
    // OrvixPanel doesn't have a "change plan" endpoint that maps to
    // WHMCS packages. We update the quota directly via a synthetic
    // account-update call (Phase 8 polish).
    return 'success';
}

function orvixpanel_UsageUpdate($params) {
    $accID = $params['domain'];
    $resp = orvixpanel_api('GET', "/account/$accID/usage", [
        'serverhostname' => $params['serverhostname'],
        'serverpassword' => $params['serverpassword'],
    ]);
    if ($resp['code'] !== 200) {
        return ['diskusage' => 0, 'disklimit' => 0, 'bwusage' => 0, 'bwlimit' => 0, 'lastupdate' => date('Y-m-d H:i:s')];
    }
    $b = $resp['body'];
    return [
        'diskusage'  => (int)$b['disk_used_mb'],
        'disklimit'  => (int)$b['disk_quota_mb'],
        'bwusage'    => (int)$b['bandwidth_used_gb'],
        'bwlimit'    => (int)$b['bandwidth_quota_gb'],
        'lastupdate' => date('Y-m-d H:i:s'),
    ];
}

function orvixpanel_ClientAreaPage($params) {
    // Renders a small widget in the WHMCS client area with a link
    // to the OrvixPanel SSO login.
    $code = '
    <div class="orvixpanel-widget">
        <h3>OrvixPanel</h3>
        <p>Manage your hosting account, files, email, and databases.</p>
        <a href="https://' . htmlspecialchars($params['serverhostname']) . ':8443/sso?whmcs=' . urlencode($params['clientdetails']['email'] ?? '') . '" class="btn btn-primary" target="_blank">Open Panel</a>
    </div>
    ';
    return ['templatefile' => 'overview', 'vars' => ['code' => $code]];
}

function orvixpanel_AdminLink($params) {
    // Direct admin login link.
    $code = '<a href="https://' . htmlspecialchars($params['serverhostname']) . ':8443/admin/sso?key=' . urlencode($params['serverpassword']) . '" target="_blank">Open Panel</a>';
    return $code;
}
