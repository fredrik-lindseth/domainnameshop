# Dependency review

Reviewed the pinned `github.com/libdns/domainnameshop v0.2.2` source against
the [Domeneshop API contract](https://api.domeneshop.no/docs/) and libdns 1.1.1.
This document records unresolved upstream limitations, not fixes delivered by
this wrapper. Review these before deploying to production.

## Findings in libdns/domainnameshop v0.2.2

1. **Cleanup may fail in zones containing MX or SRV records.** `models.go`
   represents `priority`, `weight`, and `port` as Go strings. The API defines
   them as JSON integers, so decoding a zone containing these values fails.
   `DeleteRecords` may fetch the entire zone to locate an ACME TXT record.

2. **Deletion can select the wrong record type.**
   `client.go:getRecordFromKnownRecords` compares only host and data when no
   record ID is available. It does not compare type or MX/SRV fields. Records
   of different types with the same owner and data are ambiguous.

3. **SetRecords does not replace RRsets.** Conversion from libdns records does
   not set the internal ID, so `createOrUpdateDNSRecord` always creates a new
   record. Existing siblings remain. Its update path also uses POST where the
   API specifies PUT and expects JSON where the API returns an empty 204.

4. **Domain lookup treats a substring filter as an exact match.** The API
   `/domains?domain=...` returns domain names containing the query. The client
   requires exactly one result without selecting an exact domain name. An
   account with `example.no` and `anotherexample.no` can fail lookup; a lone
   nonexact result can target a different zone.

5. **Conversions and return values need regression coverage.** Typed MX input
   includes preference in `data` as well as the separate priority field. Generic
   MX/SRV records do not populate their separate fields. Zero TTL is sent as
   120 seconds but returned unchanged as zero. Batch errors discard completed
   results, and `DeleteRecords` reports missing records as deleted.

6. **Record contents are logged unconditionally.** `GetRecords` and
   `SetRecords` write all returned records with `log.Printf`. Remove this
   debug logging upstream to avoid disclosing zone contents in ordinary logs.

## Recommended follow-up

Fix these in the libdns provider with mock HTTP regression tests for numeric
MX/SRV fields, exact zone selection, typed and generic record conversion,
record identity, whole-RRset replacement, and partial failures. Then update
this wrapper to a reviewed release and repeat the Caddy integration checks.

Do not duplicate the provider inside this wrapper as a routine configuration
fix. Maintaining a local provider fork is a separate ownership decision. No
live DNS operations were performed during this review.
