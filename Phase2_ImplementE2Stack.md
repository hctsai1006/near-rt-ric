/analysis Phase II: E2 SCTP & ASN.1 PER  

TASKS:
1. In `pkg/e2/asn1.go`, replace JSON stub with `github.com/ishidawataru/sctp` + `github.com/onfio/asn1/per` PER encoder calls.
2. In `pkg/e2/sctp.go`, switch from `net.Listen("tcp")` to `sctp.ListenSCTP()`, configure InitMsg for multi-stream.
3. Add `pkg/e2/e2_integration_test.go` for Setup and Subscription procedures.

VALIDATION:
!git diff pkg/e2/asn1.go pkg/e2/sctp.go
!go test pkg/e2 -coverprofile=coverage_e2.out
!go tool cover -func=coverage_e2.out | tail -1