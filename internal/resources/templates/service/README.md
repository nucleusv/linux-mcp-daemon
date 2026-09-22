# service

This package implements the `service` tool/resource for the MCP daemon.

## Overview

This module provides the core implementation for retrieving or modifying the relevant system data.

## Usage & Permissions

Refer to `configs/mcp-sudo.yaml` to see the default privilege requirements for this feature.
If this tool wraps a privileged binary, the worker execution will run as root if allowed by the configuration.
