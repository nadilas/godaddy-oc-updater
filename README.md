# Godaddy-oc-updater

A package to update the IP address in A records of a godaddy.com managed domain.

## Environment variables
#### Required

- API_KEY - the `godaddy` API Key
- API_SECRET - the `godaddy` API secret
- API_DOMAIN - the domain to be managed

#### Optional
- API_NEW_TTL - set it to an integer to update TTL as well 
- DOMAIN_NAMES_WHITELIST - a whitelist of A record names to be updated.
  > Note: if no whitelist is provided all A records will be considered for update
## Run

When the application runs it grabs the current public IP.
It checks out all A records on the domain and verifies if they match with the current public IP and whether an update is required.