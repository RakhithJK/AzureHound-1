// Copyright (C) 2022 Specter Ops, Inc.
//
// This file is part of AzureHound.
//
// AzureHound is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// AzureHound is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package azure

// Lists the client applications that are pre-authorized with the specified permissions to access this application's
// APIs. Users are not required to consent to any pre-authorized application (for the permissions specified). However,
// any additional permissions not listed in preAuthorizedApplications (requested through incremental consent for
// example) will require user consent.
type PreAuthorizedApplication struct {
	// The unique identifier for the application.
	AppId string `json:"appId"`
	// The unique identifiers for the OAuth2PermissionScopes the application requires.
	PermissionIds []string `json:"permissionIds"`
	// The unique identifiers for the OAuth2PermissionScopes the application requires.
	DelegatedPermissionIds []string `json:"delegatedPermissionIds"`
}
