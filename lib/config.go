/*
Copyright 2015 Google Inc. All rights reserved.

Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file or at
https://developers.google.com/open-source/licenses/bsd
*/

package lib

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/codegangsta/cli"

	"launchpad.net/go-xdg/v0"
)

const (
	// A website with user-friendly information.
	ConnectorHomeURL = "https://github.com/google/cups-connector"

	GCPAPIVersion = "2.0"
)

var (
	ConfigFilenameFlag = cli.StringFlag{
		Name:  "config-filename",
		Usage: fmt.Sprintf("Connector config filename (default \"%s\")", defaultConfigFilename),
		Value: defaultConfigFilename,
	}
)

var (
	// To be populated by something like:
	// go install -ldflags "-X github.com/google/cups-connector/lib.BuildDate=`date +%Y.%m.%d`"
	BuildDate = "DEV"

	ShortName = "CUPS Connector " + BuildDate + "-" + runtime.GOOS

	FullName = "Google Cloud Print CUPS Connector version " + BuildDate + "-" + runtime.GOOS
)

type Config struct {
	// Associated with root account. XMPP credential.
	XMPPJID string `json:"xmpp_jid,omitempty"`

	// Associated with robot account. Used for acquiring OAuth access tokens.
	RobotRefreshToken string `json:"robot_refresh_token,omitempty"`

	// Associated with user account. Used for sharing GCP printers; may be omitted.
	UserRefreshToken string `json:"user_refresh_token,omitempty"`

	// Scope (user, group, domain) to share printers with.
	ShareScope string `json:"share_scope,omitempty"`

	// User-chosen name of this proxy. Should be unique per Google user account.
	ProxyName string `json:"proxy_name,omitempty"`

	// XMPP server FQDN.
	XMPPServer string `json:"xmpp_server,omitempty"`

	// XMPP server port number.
	XMPPPort uint16 `json:"xmpp_port,omitempty"`

	// XMPP ping timeout (give up waiting after this time).
	XMPPPingTimeout string `json:"gcp_xmpp_ping_timeout,omitempty"`

	// XMPP ping interval (time between ping attempts).
	// This value is used when a printer is registered, and can
	// be overridden through the GCP API update method.
	// TODO: Rename with "_default" removed.
	XMPPPingInterval string `json:"gcp_xmpp_ping_interval_default,omitempty"`

	// GCP API URL prefix.
	GCPBaseURL string `json:"gcp_base_url,omitempty"`

	// OAuth2 client ID (not unique per client).
	GCPOAuthClientID string `json:"gcp_oauth_client_id,omitempty"`

	// OAuth2 client secret (not unique per client).
	GCPOAuthClientSecret string `json:"gcp_oauth_client_secret,omitempty"`

	// OAuth2 auth URL.
	GCPOAuthAuthURL string `json:"gcp_oauth_auth_url,omitempty"`

	// OAuth2 token URL.
	GCPOAuthTokenURL string `json:"gcp_oauth_token_url,omitempty"`

	// Maximum quantity of jobs (data) to download concurrently.
	GCPMaxConcurrentDownloads uint `json:"gcp_max_concurrent_downloads,omitempty"`

	// CUPS job queue size.
	// TODO: rename without cups_ prefix
	NativeJobQueueSize uint `json:"cups_job_queue_size"`

	// Interval (eg 10s, 1m) between CUPS printer state polls.
	// TODO: rename without cups_ prefix
	NativePrinterPollInterval string `json:"cups_printer_poll_interval"`

	// Add the job ID to the beginning of the job title. Useful for debugging.
	PrefixJobIDToJobTitle bool `json:"prefix_job_id_to_job_title"`

	// Prefix for all GCP printers hosted by this connector.
	DisplayNamePrefix string `json:"display_name_prefix"`

	// Enable SNMP to augment native printer information.
	SNMPEnable bool `json:"snmp_enable"`

	// Community string to use.
	SNMPCommunity string `json:"snmp_community"`

	// Maximum quantity of open SNMP connections.
	SNMPMaxConnections uint `json:"snmp_max_connections"`

	// Ignore printers with native names.
	PrinterBlacklist []string `json:"printer_blacklist"`

	// Enable local discovery and printing.
	LocalPrintingEnable bool `json:"local_printing_enable"`

	// Enable cloud discovery and printing.
	CloudPrintingEnable bool `json:"cloud_printing_enable"`

	// Least severity to log.
	LogLevel string `json:"log_level"`

	// CUPS only: Where to place log file.
	LogFileName string `json:"log_file_name"`

	// CUPS only: Maximum log file size.
	LogFileMaxMegabytes uint `json:"log_file_max_megabytes"`

	// CUPS only: Maximum log file quantity.
	LogMaxFiles uint `json:"log_max_files"`

	// CUPS only: Log to the systemd journal instead of to files?
	LogToJournal bool `json:"log_to_journal"`

	// CUPS only: Filename of unix socket for connector-check to talk to connector.
	MonitorSocketFilename string `json:"monitor_socket_filename"`

	// CUPS only: Maximum quantity of open CUPS connections.
	CUPSMaxConnections uint `json:"cups_max_connections,omitempty"`

	// CUPS only: timeout for opening a new connection.
	CUPSConnectTimeout string `json:"cups_connect_timeout,omitempty"`

	// CUPS only: printer attributes to copy to GCP.
	CUPSPrinterAttributes []string `json:"cups_printer_attributes,omitempty"`

	// CUPS only: use the full username (joe@example.com) in CUPS job.
	CUPSJobFullUsername bool `json:"cups_job_full_username,omitempty"`

	// CUPS only: ignore printers with make/model 'Local Raw Printer'.
	CUPSIgnoreRawPrinters bool `json:"cups_ignore_raw_printers"`

	// CUPS only: ignore printers with make/model 'Local Printer Class'.
	CUPSIgnoreClassPrinters bool `json:"cups_ignore_class_printers"`

	// CUPS only: copy the CUPS printer's printer-info attribute to the GCP printer's defaultDisplayName.
	// TODO: rename with cups_ prefix
	CUPSCopyPrinterInfoToDisplayName bool `json:"copy_printer_info_to_display_name,omitempty"`
}

// getConfigFilename gets the absolute filename of the config file specified by
// the ConfigFilename flag, and whether it exists.
//
// If the (relative or absolute) ConfigFilename exists, then it is returned.
// If the ConfigFilename exists in a valid XDG path, then it is returned.
// If neither of those exist, the (relative or absolute) ConfigFilename is returned.
func getConfigFilename(context *cli.Context) (string, bool) {
	cf := context.GlobalString("config-filename")

	if filepath.IsAbs(cf) {
		// Absolute path specified; user knows what they want.
		_, err := os.Stat(cf)
		return cf, err == nil
	}

	absCF, err := filepath.Abs(cf)
	if err != nil {
		// syscall failure; treat as if file doesn't exist.
		return cf, false
	}
	if _, err := os.Stat(absCF); err == nil {
		// File exists on relative path.
		return absCF, true
	}

	if xdgCF, err := xdg.Config.Find(cf); err == nil {
		// File exists in an XDG directory.
		return xdgCF, true
	}

	// Default to relative path. This is probably what the user expects if
	// it wasn't found anywhere else.
	return absCF, false
}

// GetConfig reads a Config object from the config file indicated by the config
// filename flag. If no such file exists, then DefaultConfig is returned.
func GetConfig(context *cli.Context) (*Config, string, error) {
	cf, exists := getConfigFilename(context)
	if !exists {
		return &DefaultConfig, "", nil
	}

	b, err := ioutil.ReadFile(cf)
	if err != nil {
		return nil, "", err
	}

	var config Config
	if err = json.Unmarshal(b, &config); err != nil {
		return nil, "", err
	}
	return &config, cf, nil
}

// ToFile writes this Config object to the config file indicated by ConfigFile.
func (c *Config) ToFile(context *cli.Context) (string, error) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}

	cf, _ := getConfigFilename(context)
	if err = ioutil.WriteFile(cf, b, 0600); err != nil {
		return "", err
	}
	return cf, nil
}
