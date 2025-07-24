/analysis Phase IV: O1 Complete  

TASKS:
1. Create `pkg/o1/netconf_server.go` implementing SSH-based NETCONF (RFC 6241) with `golang.org/x/crypto/ssh`.
2. Create `pkg/o1/yang/` loader using `github.com/openconfig/ygot`.
3. Implement `GetConfig`, `EditConfig`, and `Lock` in `pkg/o1/netconf_datastore.go`.
4. Add end-to-end tests in `pkg/o1/netconf_integration_test.go`.

VALIDATION:
!git diff pkg/o1/netconf_server.go pkg/o1/netconf_datastore.go
!go test pkg/o1 -coverprofile=coverage_o1.out
!go tool cover -func=coverage_o1.out | tail -1
